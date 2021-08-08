package polling

import (
	"container/list"
	"time"

	"github.com/rs/zerolog/log"
)

type manager struct {
	Events       chan []*Event
	EventRequest chan int
	EventOut     chan []*Event
	Quit         chan bool
	Queue        *eventQueue
}

func newManager(MaxBufferSize int) *manager {
	return &manager{
		Events:       make(chan []*Event, 1),
		EventRequest: make(chan int),
		EventOut:     make(chan []*Event, 1),
		Quit:         make(chan bool, 1),
		Queue: &eventQueue{
			list.New(),
			MaxBufferSize,
		},
	}
}

func (pm *pollingManager) start() error {
	for {
		select {
		case event := <-pm.Events:
			log.Debug().Msg("Captured WriteEvent request")
			pm.Manager.Queue.queueEvent(event)
		case rType := <-pm.ReadEvent:
			log.Debug().Msgf("Captured ReadEvent request %d", rType)
			pm.Manager.EventOut <- pm.Manager.Queue.getEvents(rType)
		case <-pm.Quit:
			log.Debug().Msg("Captured Quit request")
			select {
			case <-time.After(time.Duration(2) * time.Second):
				break
			case event := <-pm.Events:
				pm.Manager.Queue.queueEvent(event)
			}
		}
	}
}
