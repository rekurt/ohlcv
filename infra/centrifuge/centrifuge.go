package centrifuge

import (
	"bitbucket.org/novatechnologies/common/infra/logger"
	"bitbucket.org/novatechnologies/ohlcv/domain"
	"bitbucket.org/novatechnologies/ohlcv/infra"
	"encoding/json"
	"fmt"
	"github.com/centrifugal/centrifuge-go"
)

type eventHandler struct{}

func NewEventHandler() *eventHandler {
	return &eventHandler{}
}

func (h *eventHandler) OnConnect(_ *centrifuge.Client, e centrifuge.ConnectEvent) {
	logger.FromContext(infra.GetContext()).Infof("Connected to channel with ID %s", e.ClientID)
}

func (h *eventHandler) OnError(_ *centrifuge.Client, e centrifuge.ErrorEvent) {
	logger.FromContext(infra.GetContext()).Infof("Error: %s", e.Message)
}

func (h *eventHandler) OnMessage(_ *centrifuge.Client, e centrifuge.MessageEvent) {
	logger.FromContext(infra.GetContext()).Infof("Message from server: %s", string(e.Data))
}

func (h *eventHandler) OnDisconnect(_ *centrifuge.Client, e centrifuge.DisconnectEvent) {
	logger.FromContext(infra.GetContext()).Infof("Disconnected from channel: %s", e.Reason)
}

func (h *eventHandler) OnServerSubscribe(_ *centrifuge.Client, e centrifuge.ServerSubscribeEvent) {
	logger.FromContext(infra.GetContext()).Infof("Subscribe to server-side channel %s: (resubscribe: %t, recovered: %t)", e.Channel, e.Resubscribed, e.Recovered)
}

func (h *eventHandler) OnServerUnsubscribe(_ *centrifuge.Client, e centrifuge.ServerUnsubscribeEvent) {
	logger.FromContext(infra.GetContext()).Infof("Unsubscribe from server-side channel %s", e.Channel)
}

func (h *eventHandler) OnServerJoin(_ *centrifuge.Client, e centrifuge.ServerJoinEvent) {
	logger.FromContext(infra.GetContext()).Infof("Server-side join to channel %s: %s (%s)", e.Channel, e.User, e.Client)
}

func (h *eventHandler) OnServerLeave(_ *centrifuge.Client, e centrifuge.ServerLeaveEvent) {
	logger.FromContext(infra.GetContext()).Infof("Server-side leave from channel %s: %s (%s)", e.Channel, e.User, e.Client)
}

func (h *eventHandler) OnServerPublish(_ *centrifuge.Client, e centrifuge.ServerPublishEvent) {
	logger.FromContext(infra.GetContext()).Infof("Publication from server-side channel %s: %s", e.Channel, e.Data)
}

func (h *eventHandler) OnPublish(sub *centrifuge.Subscription, e centrifuge.PublishEvent) {
	var candle *domain.Candle
	err := json.Unmarshal(e.Data, &candle)
	if err != nil {
		return
	}
	logger.FromContext(infra.GetContext()).Infof("Updated candle publish via channel %s: %s", sub.Channel(), candle.Timestamp)
}

func (h *eventHandler) OnSubscribeSuccess(sub *centrifuge.Subscription, e centrifuge.SubscribeSuccessEvent) {
	logger.FromContext(infra.GetContext()).Infof("Subscribed on channel %s, resubscribed: %v, recovered: %v", sub.Channel(), e.Resubscribed, e.Recovered)
}

func (h *eventHandler) OnSubscribeError(sub *centrifuge.Subscription, e centrifuge.SubscribeErrorEvent) {
	logger.FromContext(infra.GetContext()).Infof("Subscribed on channel %s failed, error: %s", sub.Channel(), e.Error)
}

func (h *eventHandler) OnUnsubscribe(sub *centrifuge.Subscription, _ centrifuge.UnsubscribeEvent) {
	logger.FromContext(infra.GetContext()).Infof("Unsubscribed from channel %s", sub.Channel())
}


func NewClient(handler *eventHandler, config infra.CentrifugeConfig) *centrifuge.Client {
	wsURL := fmt.Sprintf("ws://%s/connection/websocket", config.Host)
	c := centrifuge.NewJsonClient(wsURL, centrifuge.DefaultConfig())

	// Uncomment to make it work with Centrifugo and its JWT auth.
	//c.SetToken(connToken("49", 0))

	c.OnConnect(handler)
	c.OnDisconnect(handler)
	c.OnMessage(handler)
	c.OnError(handler)

	c.OnServerPublish(handler)
	c.OnServerSubscribe(handler)
	c.OnServerUnsubscribe(handler)
	c.OnServerJoin(handler)
	c.OnServerLeave(handler)

	return c
}
