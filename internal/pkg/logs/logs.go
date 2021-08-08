package logs

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogs initiates logger and global level
func InitLogs(logLevel string) {

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	switch logLevel {
	case "QUIET":
		zerolog.SetGlobalLevel(zerolog.Disabled)
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "DEBUG":
		log.Logger = log.With().Caller().Logger()
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Debug().Msg("Logger initalized")
}
