package inmemo

import (
	"bitbucket.org/novatechnologies/common/infra/logger"

	"bitbucket.org/novatechnologies/ohlcv/domain"
)

var _ domain.EventManager = new(InMemory)

// InMemory is in-memory manager which stores subscribtions and run
// handlers as separate goroutines.
// It's thread unsafe cause of supposed timing of read/write ops while it'll
// be using.
type InMemory struct {
	log         logger.Logger
	subscribers map[domain.EventType][]domain.EventHandler
}

func NewInMemory() *InMemory {
	return &InMemory{
		log:         logger.DefaultLogger,
		subscribers: make(map[domain.EventType][]domain.EventHandler),
	}
}

func (ps *InMemory) WithLogger(lg logger.Logger) *InMemory {
	ps.log = lg
	return ps
}

func (ps *InMemory) Subscribe(tp domain.EventType, h domain.EventHandler) {
	if tp == "" && h == nil {
		return
	}

	ps.subscribers[tp] = append(ps.subscribers[tp], h)
}

func (ps *InMemory) Publish(tp domain.EventType, ev *domain.Event) {
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
