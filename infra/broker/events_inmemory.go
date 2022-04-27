package broker

import (
	"bitbucket.org/novatechnologies/common/infra/logger"

	"bitbucket.org/novatechnologies/ohlcv/domain"
)

var _ domain.EventsBroker = new(EventsInMemory)

// EventsInMemory is in-memory manager which stores subscribtions and run
// handlers as separate goroutines.
// It's thread unsafe cause of supposed timing of read/write ops while it'll
// be using.
type EventsInMemory struct {
	log         logger.Logger
	subscribers map[domain.EventType][]domain.EventHandler
}

func NewInMemory() *EventsInMemory {
	return &EventsInMemory{
		log:         logger.DefaultLogger,
		subscribers: make(map[domain.EventType][]domain.EventHandler),
	}
}

func (ps *EventsInMemory) WithLogger(lg logger.Logger) *EventsInMemory {
	ps.log = lg
	return ps
}

func (ps *EventsInMemory) Subscribe(
	tp domain.EventType,
	h domain.EventHandler,
) {
	if tp == "" && h == nil {
		return
	}

	ps.subscribers[tp] = append(ps.subscribers[tp], h)
}

func (ps *EventsInMemory) Publish(tp domain.EventType, ev *domain.Event) {
	for _, handler := range ps.subscribers[tp] {
		currHandler := handler

		go func() {
			defer func() {
				if r := recover(); r != nil {
					ps.log.Errorf(
						"Panic while executing handler %+v for %s tp: %+v",
						currHandler, tp, r,
					)
				}
			}()

			if err := currHandler(ev); err != nil {
				ps.log.Errorf(
					"Error while executing handler %+v for %s tp: %v",
					currHandler, tp, err,
				)
			}
		}()
	}
}
