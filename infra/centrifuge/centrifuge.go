package centrifuge

import (
	"context"
	"fmt"

	"bitbucket.org/novatechnologies/common/infra/logger"
	cfge "github.com/centrifugal/centrifuge-go"
	"github.com/centrifugal/gocent/v3"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"bitbucket.org/novatechnologies/ohlcv/infra"
)

// NewClient returns centrifugo server-side WS client.
func NewClient(
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

	return c, nil
}

type MessageData struct {
	Channel string `json:"channel"`
	Data    string `json:"data"`
}

type Centrifuge interface {
	BatchPublish(ctx context.Context, messages []MessageData)
	Publish(ctx context.Context, message MessageData)
}

type centrifuge struct {
	Client *gocent.Client
}

func New(cfg infra.CentrifugeConfig) *centrifuge {
	clientConfig := gocent.Config{
		Addr: "http://" + cfg.Host + "/api",
		Key:  cfg.Token,
	}
	client := gocent.New(clientConfig)

	return &centrifuge{
		Client: client,
	}
}

func (c centrifuge) Publish(ctx context.Context, message MessageData) {
	log := logger.FromContext(ctx).WithField("message", message)
	result, err := c.Client.Publish(ctx, message.Channel, []byte(message.Data))
	if err != nil {
		log.Errorf("Error calling publish: %v", err)
	}
	log.Infof(
		"Publish into channel %s successful, stream position {offset: %d, epoch: %s}",
		message.Channel,
		result.Offset,
		result.Epoch,
	)
}

func (c centrifuge) BatchPublish(ctx context.Context, messages []MessageData) {
	log := logger.FromContext(ctx)
	pipe := c.Client.Pipe()
	for _, message := range messages {
		e := pipe.AddPublish(message.Channel, []byte(message.Data))
		if e != nil {
			log.Errorf("Error calling BatchPublish func: %v", e)
		}
	}
	replies, err := c.Client.SendPipe(ctx, pipe)
	if err != nil {
		log.Errorf("Error sending pipe: %v", err.Error())
	}
	for _, reply := range replies {
		if reply.Error != nil {
			log.Errorf("Error in pipe reply: %v", err)
		}
	}
	log.Infof("Sent %d publish commands in one HTTP request ", len(replies))
}
