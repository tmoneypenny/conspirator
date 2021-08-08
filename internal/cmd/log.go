package cmd

import (
	"github.com/spf13/viper"
	"github.com/tmoneypenny/conspirator/internal/pkg/logs"
)

var defaultLoggerLevel = "INFO"

func initLogger() {
	if viper.GetString("logLevel") == "" {
		viper.Set("logLevel", defaultLoggerLevel)
	}
	logs.InitLogs(viper.GetString("logLevel"))
	// logging stuff
}
