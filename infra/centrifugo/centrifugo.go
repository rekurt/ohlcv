package centrifugo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"bitbucket.org/novatechnologies/common/infra/logger"
	cfge "github.com/centrifugal/centrifuge-go"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
)

type loggingEventHandler struct{}

func NewLoggingEventHandler() *loggingEventHandler {
	return &loggingEventHandler{}
}

func (h *loggingEventHandler) OnConnect(
	_ *cfge.Client,
	e cfge.ConnectEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Connected to channel with ID %s",
		e.ClientID,
	)
}

func (h *loggingEventHandler) OnError(
	_ *cfge.Client,
	e cfge.ErrorEvent,
) {
	logger.FromContext(infra.GetContext()).Infof("Error: %s", e.Message)
}

func (h *loggingEventHandler) OnMessage(
	_ *cfge.Client,
	e cfge.MessageEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Event from server: %s",
		string(e.Data),
	)
}

func (h *loggingEventHandler) OnDisconnect(
	_ *cfge.Client,
	e cfge.DisconnectEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Disconnected from channel: %s",
		e.Reason,
	)
}

func (h *loggingEventHandler) OnServerSubscribe(
	_ *cfge.Client,
	e cfge.ServerSubscribeEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Subscribe to server-side channel %s: (resubscribe: %t, recovered: %t)",
		e.Channel,
		e.Resubscribed,
		e.Recovered,
	)
}

func (h *loggingEventHandler) OnServerUnsubscribe(
	_ *cfge.Client,
	e cfge.ServerUnsubscribeEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Unsubscribe from server-side channel %s",
		e.Channel,
	)
}

func (h *loggingEventHandler) OnServerJoin(
	_ *cfge.Client,
	e cfge.ServerJoinEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Server-side join to channel %s: %s (%s)",
		e.Channel,
		e.User,
		e.Client,
	)
}

func (h *loggingEventHandler) OnServerLeave(
	_ *cfge.Client,
	e cfge.ServerLeaveEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Server-side leave from channel %s: %s (%s)",
		e.Channel,
		e.User,
		e.Client,
	)
}

func (h *loggingEventHandler) OnServerPublish(
	_ *cfge.Client,
	e cfge.ServerPublishEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Publication from server-side channel %s: %s",
		e.Channel,
		e.Data,
	)
}

func (h *loggingEventHandler) OnPublish(
	sub *cfge.Subscription,
	e cfge.PublishEvent,
) {
	var candle *domain.Candle
	err := json.Unmarshal(e.Data, &candle)
	if err != nil {
		return
	}
	logger.FromContext(infra.GetContext()).Infof(
		"Updated candle publish via channel %s: %s",
		sub.Channel(),
		candle.Timestamp,
	)
}

func (h *loggingEventHandler) OnSubscribeSuccess(
	sub *cfge.Subscription,
	e cfge.SubscribeSuccessEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Subscribed on channel %s, resubscribed: %v, recovered: %v",
		sub.Channel(),
		e.Resubscribed,
		e.Recovered,
	)
}

func (h *loggingEventHandler) OnSubscribeError(
	sub *cfge.Subscription,
	e cfge.SubscribeErrorEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Subscribed on channel %s failed, error: %s",
		sub.Channel(),
		e.Error,
	)
}

func (h *loggingEventHandler) OnUnsubscribe(
	sub *cfge.Subscription,
	_ cfge.UnsubscribeEvent,
) {
	logger.FromContext(infra.GetContext()).Infof(
		"Unsubscribed from channel %s",
		sub.Channel(),
	)
}

// NewWSClient returns centrifugo frontend-side WS client.
func NewWSClient(
	config infra.CentrifugoClientConfig,
) (*cfge.Client, error) {
	wsURL := fmt.Sprintf("ws://%s/connection/websocket", config.Addr)
	c := cfge.NewJsonClient(wsURL, cfge.DefaultConfig())

	if config.SignTokenKey != "" {
		// TODO: token must be gathered from auth server in the future.
		token := jwt.NewWithClaims(
			jwt.SigningMethodRS512,
			jwt.MapClaims{
				// https://centrifugal.dev/docs/server/private_channels#client
				"client": "go-client#" + uuid.New().String(),
			},
		)
		signedToken, err := token.SignedString(config.SignTokenKey)
		if err != nil {
			return nil, errors.Wrapf(
				err, "can't sign token with key %s", config.SignTokenKey,
			)
		}

		c.SetToken(signedToken)
	}

	if config.Debug {
		handler := NewLoggingEventHandler()
		c.OnConnect(handler)
		c.OnDisconnect(handler)
		c.OnMessage(handler)
		c.OnError(handler)

		c.OnServerPublish(handler)
		c.OnServerSubscribe(handler)
		c.OnServerUnsubscribe(handler)
		c.OnServerJoin(handler)
		c.OnServerLeave(handler)
	}

	return c, nil
}

const marketDataNS = "market-data"
const wsChannelSep = "_"

// RequiredParams is quantum peace of data publishing into Centrifugo.
type RequiredParams struct {
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}

// WSPubCommand base structure of JSON body for Centrifugo API commands.
type WSPubCommand struct {
	Method string         `json:"method"`
	Params RequiredParams `json:"params,omitempty"`
}

type ApiClient struct {
	postingCli *restyPostClient
	marketData domain.EventManager
}

// NewAPIClient returns server-side HTTP client based on centrifugo.ApiClient.
func NewAPIClient(
	cfg infra.CentrifugoClientConfig,
	marketDataBus domain.EventManager,
) (*ApiClient, error) {
	baseURL := "://" + cfg.Addr + cfg.ServerAPIPrefix
	cli := &ApiClient{
		postingCli: newRestyPostClient(baseURL, cfg.ServerAPIKey),
	}

	// check health.
	if info, err := cli.Info(infra.GetContext()); err == nil || len(info) == 0 {
		return nil, errors.Wrap(err, "websocket server is unhealthy")
	}

	cli.subscribe(marketDataBus)

	return cli, nil
}

func (c *ApiClient) subscribe(marketData domain.EventManager) {
	marketData.Subscribe(domain.ETypeCharts, c.handleChart)
}

// handleChart receives charts messeges from local publisher (Candle service)
// and tries to broadcast it by Centrifugo API.
func (c *ApiClient) handleChart(chartMsg *domain.Event) error {
	chart, ok := chartMsg.Payload().(domain.Chart)
	if !ok {
		err := errors.Errorf(
			"ws server can't cast %v (%T) to domain.Chart",
			chartMsg, chartMsg,
		)
		logger.DefaultLogger.Errorf(
			"Centrifugo client can't handle chart:",
			err,
		)

		return err
	}

	if err := c.PublishChart(chartMsg.Ctx, chart); err != nil {
		logger.FromContext(chartMsg.Ctx).
			WithField("interval", chart.Interval()).
			WithField("market", chart.Market()).
			Errorf(
				"CandleService.PublishChart method error: %v.",
				err,
			)

		return err
	}

	return nil
}

// Info returns JSON bytes with technical info about the WS server, but primary
// usage is for ping.
// TODO: needs to be done with health endpoint.
func (c *ApiClient) Info(ctx context.Context) (json.RawMessage, error) {
	return c.postingCli.Post(ctx, WSPubCommand{Method: "info"})
}

// Publish sends some data into channel to all subscribers.
func (c *ApiClient) Publish(
	ctx context.Context,
	channel string,
	data interface{},
) error {
	_, err := c.postingCli.Post(
		ctx,
		WSPubCommand{
			Method: "publish",
			Params: RequiredParams{channel, data},
		},
	)

	return err
}

// PublishChart does the same that do Publish but relevant for OHLCV charts
// only.
func (c *ApiClient) PublishChart(
	ctx context.Context,
	chart domain.Chart,
) error {
	channel := strings.Join(
		[]string{marketDataNS, chart.Market(), chart.Interval()},
		wsChannelSep,
	)

	return c.Publish(ctx, channel, chart)
}

type ChartsHandler struct {
	sub domain.EventManager
	cli *ApiClient
}
