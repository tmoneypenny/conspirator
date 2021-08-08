package polling

import (
	"container/list"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type EventTestSuite struct {
	suite.Suite
	events *eventQueue
}

func (s *EventTestSuite) SetupTest() {
	s.events = &eventQueue{
		list.New(),
		6,
	}
}

func (s *EventTestSuite) Queue() {
	testCases := []struct {
		data interface{}
	}{
		{
			"test1",
		},
		{
			"test2",
		},
		{
			"test3",
		},
		{
			"test4",
		},
		{
			"test5",
		},
		{
			"test6",
		},
	}

	for t := range testCases {
		assert.NoError(s.T(), s.events.queueEvent(&Event{
			Timestamp: time.Now().Unix(),
			Data:      testCases[t].data,
			Id:        uuid.New(),
		}))
	}
}

func (s *EventTestSuite) TestGet() {
	s.Queue()

	eventList := s.events.getEvents(1)
	assert.NotEmpty(s.T(), eventList)

	expectedData := []string{
		"test1",
		"test2",
		"test3",
		"test4",
		"test5",
		"test6",
	}

	for i := range eventList {
		assert.Equal(s.T(), expectedData[i], eventList[i].Data.(string))
		fmt.Println(eventList[i].Data.(string))
	}

}

func TestEventTestSuite(t *testing.T) {
	suite.Run(t, new(EventTestSuite))
}
