package polling

import (
	"container/list"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Event implements a single event inserted into queue
type Event struct {
	Timestamp int64
	Data      interface{}
	Id        uuid.UUID
}

type eventQueue struct {
	*list.List
	MaxBufferSize int
}

// QueueEvent inserts a new event into the queue
func (q *eventQueue) queueEvent(event *Event) error {
	if event == nil {
		return fmt.Errorf("Received empty event")
	}

	log.Debug().Msg("Received event to queue")

	if q.List.Len() >= q.MaxBufferSize {
		log.Debug().Msg("Buffer at max, removing oldest")
		oldest := q.List.Back()
		if oldest != nil {
			q.List.Remove(oldest)
		}
	}

	q.List.PushFront(event)

	return nil
}

// GetEvents returns all events in the buffer
func (q *eventQueue) getEvents(deleteAfterFetch int) []*Event {
	events := make([]*Event, 0)

	var lastElement *list.Element = q.List.Back()

	if lastElement != nil {
		var prev *list.Element
		for element := lastElement; element != nil; element = prev {
			event, ok := element.Value.(*Event)

			if !ok {
				log.Warn().Msg("error getting messages from queue")
				return events
			}

			events = append(events, &Event{event.Timestamp, event.Data, event.Id})
			prev = element.Prev()

			if deleteAfterFetch != 0 {
				q.List.Remove(element)
			}
		}
	}

	return events
}
