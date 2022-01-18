package wrapper

import "github.com/tmoneypenny/conspirator/internal/pkg/polling"

type Module interface {
	Start()
	Stop()
}

type Config struct {
	PollingManager *polling.PollingServer
	PublicAddress  string
	Cfg            map[string]interface{}
}
