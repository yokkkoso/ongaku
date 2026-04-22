package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetupLogger() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC1123}).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Caller().
		Logger()
}
