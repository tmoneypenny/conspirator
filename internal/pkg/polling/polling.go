package polling

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// PollingServer contains the eventHandler and polling manager
// that should be passed around to each package. Once the polling
// server is started via Start(), a package can begin publishing
// messages via Publish(), or fetch all messages in the queue via
// GetAll()
type PollingServer struct {
	manager      *pollingManager
	eventHandler chan<- *Event
	readHandler  chan<- int
	quitHandler  chan<- bool
	Config       *PollingConfig
}

type pollingManager struct {
	Manager   *manager
	Events    <-chan *Event
	ReadEvent <-chan int
	Quit      <-chan bool
}

// PollingConfig is used to configure the polling manager
type PollingConfig struct {
	// MaxBufferSize defines the maximum number of events
	// per subscription before the oldest events are expired.
	MaxBufferSize int
	// DeleteEventAfterRetrieval defines if the event should be
	// deleted after it is retreived.
	DeleteAfter bool
}

// New returns a PollingServer struct used to managed the polling server
func New(cfg *PollingConfig) *PollingServer {
	return &PollingServer{Config: cfg}
}

// Start will create a new polling server and queue
func (s *PollingServer) Start() *PollingServer {
	events := make(chan *Event, viper.Get("maxPollingEvents").(int))
	eventRequest := make(chan int, 1)
	quit := make(chan bool, 1)

	manage := pollingManager{
		Events:    events,
		ReadEvent: eventRequest,
		Quit:      quit,
	}

	pm := newManager(s.Config.MaxBufferSize)
	manage.Manager = pm

	go manage.start()

	return &PollingServer{
		manager:      &manage,
		eventHandler: events,
		readHandler:  eventRequest,
		quitHandler:  quit,
	}
}

// Stop shuts down the polling server gracefully
func (s *PollingServer) Stop() {
	close(s.eventHandler)
	close(s.readHandler)
	s.quitHandler <- true
}

// Publish will publish an event to the polling queue
func (p *PollingServer) Publish(event interface{}) error {
	if event == nil {
		return fmt.Errorf("received nil event")
	}

	log.Debug().Msg("Adding message to Queue via Publish")
	p.eventHandler <- &Event{
		Timestamp: time.Now().Unix(),
		Data:      event,
		Id:        uuid.New(),
	}

	return nil
}

// GetAll returns and purges all events in the queue
func (p *PollingServer) GetAll() []*Event {
	log.Debug().Msg("Getting all events from queue")
	p.readHandler <- 1
	events := <-p.manager.Manager.EventOut
	return events
}

// ReadAll returns all events in the queue without purging
func (p *PollingServer) ReadAll() []*Event {
	p.readHandler <- 0
	events := <-p.manager.Manager.EventOut
	return events
}
