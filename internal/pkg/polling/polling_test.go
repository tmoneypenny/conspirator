package polling

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type PollingTestSuite struct {
	suite.Suite
	Server *PollingServer
}

func (s *PollingTestSuite) SetupTest() {
	pm := New(&PollingConfig{
		MaxBufferSize: 250,
		DeleteAfter:   true,
	})

	s.Server = pm.Start()
}

func (s *PollingTestSuite) TestPublish() {
	s.Server.Publish("test_message")
	time.Sleep(time.Second * 1)
}

func (s *PollingTestSuite) TestGet() {
	defer func() {
		time.Sleep(time.Second * 1)
		s.Server.GetAll()
		time.Sleep(time.Second * 1)
		s.Server.GetAll()
		s.Server.Publish("test_get3")
		s.Server.Publish("test_get4")
		time.Sleep(time.Second * 1)
		s.Server.GetAll()
	}()
	s.Server.Publish("test_get1")
	s.Server.Publish("test_get2")
}

func (s *PollingTestSuite) TestStop() {
	fmt.Println("Calling Stop", time.Now().Local())
	s.Server.Stop()
}

func TestPollingTestSuite(t *testing.T) {
	suite.Run(t, new(PollingTestSuite))
}
