package domain

import "context"

type EventType = string

const (
	ETypeTrades = "trades"
	ETypeCharts = "charts"
)

type EventHandler = func(m *Event) error

// EventManager describes abstract pub-sub messaging system for internal events
// among components. Each event can contain payload and meta info, so they can
// be used not for notification purposes only.
type EventManager interface {
	Subscribe(tp EventType, h EventHandler)
	Publish(tp EventType, data *Event)
}

type (
	meta  map[string]string
	Event struct {
		Ctx     context.Context
		payload interface{}
		meta    meta
	}
)

func NewEvent(ctx context.Context, payload interface{}) *Event {
	if ctx == nil {
		ctx = context.Background()
	}

	return &Event{
		payload: payload,
		Ctx:     ctx,
		meta:    nil,
	}
}

func (m *Event) WithMetaKV(key, value string) *Event {
	if m.meta == nil {
		m.meta = make(meta)
	}
	m.meta[key] = value

	return m
}

func (m *Event) WithMeta(meta meta) *Event {
	if m.meta == nil {
		m.meta = meta
		return m
	}

	for k, v := range meta {
		m.meta[k] = v
	}

	return m
}

func (m *Event) GetMeta(key string) string {
	if m.meta == nil {
		return ""
	}

	return m.meta[key]
}

func (m *Event) MustGetChart() Chart {
	return m.payload.(Chart)
}

func (m *Event) MustGetDeal() Deal {
	return m.payload.(Deal)
}

func (m *Event) Payload() interface{} {
	return m.payload
}
