package centrifuge

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"context"
	"github.com/centrifugal/gocent/v3"
)

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
	log.Infof("Publish into channel %s successful, stream position {offset: %d, epoch: %s}", message.Channel, result.Offset, result.Epoch)
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
