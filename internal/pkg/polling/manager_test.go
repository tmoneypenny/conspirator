package polling

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestManager(t *testing.T) {
	events := make(chan *Event, 256)
	quit := make(chan bool, 1)

	manage := pollingManager{
		Events: events,
	}

	pm := newManager(10)
	manage.Manager = pm

	go manage.start()

	pm.Events <- []*Event{{time.Now().Unix(), "Test", uuid.New()}}

	go func() {
		defer func() {
			time.Sleep(time.Second * 2)
			quit <- true
		}()
	}()

	<-quit
}
