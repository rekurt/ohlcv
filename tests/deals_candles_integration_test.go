package tests

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	pubsub "bitbucket.org/novatechnologies/common/events"
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/interfaces/matcher"
	"github.com/centrifugal/centrifuge-go"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	mg "go.mongodb.org/mongo-driver/mongo"

	cfge "bitbucket.org/novatechnologies/ohlcv/infra/centrifuge"

	"bitbucket.org/novatechnologies/ohlcv/candle"
	"bitbucket.org/novatechnologies/ohlcv/deal"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"bitbucket.org/novatechnologies/ohlcv/infra/broker"
	"bitbucket.org/novatechnologies/ohlcv/infra/mongo"
)

const (
	serviceDB    = "mongo"
	serviceQueue = "kafka"
	serviceWS    = "centrifugo"
)

func parseAsNano(RFC3339Timestamp string) (unixNano int64) {
	ts, _ := time.Parse(time.RFC3339Nano, RFC3339Timestamp)
	return ts.UnixNano()
}

func loadConfigs(configsPaths []string) error {
	return godotenv.Load(configsPaths...)
}

func getEnvsMap() map[string]string {
	configsPaths := []string{"./config/testing.env", "./config/.env"}

	err := loadConfigs(configsPaths)
	if err != nil {
		logger.DefaultLogger.Errorf(
			"can't load env data from configs %v for test: %v",
			configsPaths, err,
		)
	}

	envVars := make(map[string]string)
	for _, item := range os.Environ() {
		splits := strings.Split(item, "=")
		envVars[splits[0]] = strings.Join(splits[1:], "=")
	}

	return envVars
}

type (
	waitFn       = func(context.Context, wait.StrategyTarget) error
	waitStrategy struct {
		waitFn waitFn
	}
)

func newWaitStrategy(w waitFn) waitStrategy {
	return waitStrategy{waitFn: w}
}

func (w waitStrategy) WaitUntilReady(
	ctx context.Context,
	t wait.StrategyTarget,
) error {
	return w.waitFn(ctx, t)
}

// mongoWait implements waiting strategy while MongoDB container is starting.
func mongoWait(conf infra.MongoDbConfig) waitStrategy {
	return newWaitStrategy(
		func(ctx context.Context, db wait.StrategyTarget) error {
			cli := mongo.NewMongoClient(ctx, conf)

			timeoutDur := 10 * time.Second
			start := time.Now()

			for now := range time.Tick(2 * time.Second) {
				state, err := db.State(ctx)
				if err == nil && state.Running {
					return cli.Connect(ctx)
				}

				if err != nil {
					return err
				}

				if now.Sub(start) >= timeoutDur {
					return errors.Errorf(
						"can't start MongoDB after %v",
						timeoutDur,
					)
				}
			}

			return nil
		},
	)
}

type wsPublishHandler struct {
	outCh chan centrifuge.ServerPublishEvent
}

func newWebsocketPublishHandler() wsPublishHandler {
	return wsPublishHandler{
		outCh: make(chan centrifuge.ServerPublishEvent),
	}
}

func (h wsPublishHandler) OnServerPublish(
	_ *centrifuge.Client,
	pubEvent centrifuge.ServerPublishEvent,
) {
	h.outCh <- pubEvent
}

func (h wsPublishHandler) OnDisconnect(
	_ *centrifuge.Client,
	_ centrifuge.DisconnectEvent,
) {
	close(h.outCh)
}

// GetPublishEvent waits and returns a publish event. If time is running out,
// then empty zero-value event struct returning with false bool flag.
// If timeout is not specified ( less than zero) than it's interpreted as infinite.
func (h wsPublishHandler) GetPublishEvent(timeout time.Duration) (
	centrifuge.ServerPublishEvent, bool,
) {
	if timeout < 0 {
		timeout = math.MaxInt64
	}

	select {
	case msg, ok := <-h.outCh:
		// Channel has been closed (possible disconnect from the WS server).
		if !ok {
			return centrifuge.ServerPublishEvent{}, false
		}
		return msg, true
	case <-time.After(timeout):
		return centrifuge.ServerPublishEvent{}, false
	}
}

type candlesIntegrationTestSuite struct {
	suite.Suite

	compose *testcontainers.LocalDockerCompose
	cancel  context.CancelFunc

	mongoCli        *mg.Client
	dealsCollection *mg.Collection

	kafkaConsumer  pubsub.Subscriber
	kafkaPublisher pubsub.Publisher

	deals      *deal.Service
	dealsTopic string
	candles    *candle.Service

	eventsBroker domain.EventsBroker
	broadcaster  domain.Broadcaster

	wsPub            cfge.Centrifuge
	wsSub            *centrifuge.Client
	wsPublishHandler wsPublishHandler
}

func (suite *candlesIntegrationTestSuite) setupServicesUnderTests(
	ctx context.Context, conf infra.Config,
) (err error) {
	suite.kafkaConsumer = infra.NewConsumer(ctx, conf.KafkaConfig)
	suite.kafkaPublisher, err = infra.NewPublisher(ctx, conf.KafkaConfig)
	if err != nil {
		return err
	}

	// DB client setup
	mongoClient := mongo.NewMongoClient(ctx, conf.MongoDbConfig)
	suite.mongoCli = mongoClient
	dealsCollection := mongo.GetCollection(
		ctx,
		mongoClient,
		conf.MongoDbConfig,
		conf.MongoDbConfig.MinuteCandleCollectionName,
	)
	suite.dealsCollection = dealsCollection

	// Internal domain events broker
	eventsBroker := broker.NewInMemory()

	// Deals service setup
	suite.dealsTopic = deal.TopicName(conf.KafkaConfig.TopicPrefix)
	suite.deals = deal.NewService(
		dealsCollection,
		GetAvailableMarkets(),
		eventsBroker,
	)

	// Candles service setup
	suite.candles = candle.NewService(
		&candle.Storage{DealsDbCollection: dealsCollection},
		new(candle.Agregator),
		GetAvailableMarkets(),
		domain.GetAvailableResolutions(),
		eventsBroker,
	)

	// WS publisher and broadcaster of the market data setup
	suite.wsPub = cfge.NewPublisher(conf.CentrifugeConfig)
	broadcaster := cfge.NewBroadcaster(suite.wsPub, eventsBroker, nil)
	broadcaster.SubscribeForCharts()
	suite.broadcaster = broadcaster

	// WS subscriber setup for the correctness check
	suite.wsSub, err = cfge.NewClient(conf.CentrifugeConfig)
	if err != nil {
		return err
	}

	// WS server-side publish event handler setup
	suite.wsPublishHandler = newWebsocketPublishHandler()
	suite.wsSub.OnServerPublish(suite.wsPublishHandler)

	return nil
}

func (suite *candlesIntegrationTestSuite) SetupSuite() {
	err := loadConfigs([]string{"../config/.env.testing", "../config/.env"})
	if err != nil {
		suite.T().Fatal(err)
	}
	conf := infra.Parse()

	services := []string{serviceDB, serviceQueue, serviceWS}

	composePaths := make([]string, len(services))
	for i, svcName := range services {
		composePaths[i] = fmt.Sprintf(
			"./docker/%s.docker-compose.yml",
			svcName,
		)
	}

	suite.compose = testcontainers.NewLocalDockerCompose(
		composePaths,
		"ohlcv_test",
	)

	//execErr := suite.compose.
	//	WithCommand([]string{"up", "-d"}).
	//	WithEnv(getEnvsMap()).
	//	Invoke()
	//if execErr.Error != nil {
	//	log.Println("docker-compose output:", execErr.Stdout)
	//	log.Panicf(
	//		"Failed when running %v: %v", execErr.Command, execErr.Error,
	//	)
	//}

	// suite.compose.WaitForService(serviceDB, mongoWait(conf.MongoDbConfig))

	ctx, cancel := context.WithCancel(infra.GetContext())
	if err := suite.setupServicesUnderTests(ctx, conf); err != nil {
		cancel()
		panic(err)
	}

	if err := suite.wsSub.Connect(); err != nil {
		cancel()
		panic(err)
	}

	suite.cancel = cancel
}

func (suite *candlesIntegrationTestSuite) TearDownSuite() {
	suite.cancel()

	_ = suite.mongoCli.Disconnect(infra.GetContext())
	_ = suite.wsSub.Disconnect()

	suite.compose.Down()
}

func (suite *candlesIntegrationTestSuite) TestDealsConsumeAndSave(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	a := assert.New(t)

	// Initialize context with current test timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	metadata := make(map[string]string)

	// Usual case, must be successful
	d := matcher.Deal{
		Id:        uuid.New().String(),
		Market:    "ETH_LTC",
		Price:     "0.12345678901234567890",
		Amount:    "0.0000000000123456789",
		CreatedAt: parseAsNano("2022-04-23T15:04:05.999999999Z"),
	}
	err := suite.kafkaPublisher.Publish(ctx, suite.dealsTopic, metadata, &d)
	a.NoError(err, "error while publishing casual deal %v: %v", d.String(), err)

	// Usual case, must be successful
	d = matcher.Deal{
		Id:        uuid.New().String(),
		Market:    "USDT_BTC",
		Price:     "0.098765",
		Amount:    "0.0000000000123456789",
		CreatedAt: parseAsNano("2022-04-23T15:04:06.999999999Z"),
	}
	err = suite.kafkaPublisher.Publish(ctx, suite.dealsTopic, metadata, &d)
	a.NoError(err, "error while publishing casual deal %v: %v", d.String(), err)

	// Error case - illegal input
	d = matcher.Deal{
		Id:        "",
		Market:    "XUSDT_BTC2",
		Price:     "0.0",
		Amount:    "-0.123",
		CreatedAt: parseAsNano("2000-01-00:00:00.999999999Z"),
	}
	err = suite.kafkaPublisher.Publish(ctx, suite.dealsTopic, metadata, &d)
	a.Error(err, "error should appear on invalid deal %v: %v", d.String(), err)

	// One more usual case
	d = matcher.Deal{
		Id:        uuid.New().String(),
		Market:    "USDT_TRX",
		Price:     "0.123",
		Amount:    "0.456",
		CreatedAt: parseAsNano("2022-04-23:15:05.999999999Z"),
	}
	err = suite.kafkaPublisher.Publish(ctx, suite.dealsTopic, metadata, &d)
	a.NoError(err, "error while publishing casual deal %v: %v", d.String(), err)

	// newWaitStrategy for the 3 outcome messages from WS server about updated candle charts.
	timeout := time.Second
	for i := 0; i < 3; i++ {
		pubEvent, ok := suite.wsPublishHandler.GetPublishEvent(timeout)
		if !ok {
			t.Fatalf("can't get WS publish event within %v", timeout)
		}
		fmt.Println(pubEvent)
	}
}

func TestIntegrationCandlesTestSuite(t *testing.T) {
	suite.Run(t, &candlesIntegrationTestSuite{})
}
