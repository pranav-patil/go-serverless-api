package logger

import (
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func SetGlobalLevel(l string) {
	logLevel, err := zerolog.ParseLevel(strings.ToLower(l))

	if err != nil {
		logLevel = zerolog.InfoLevel
		log.Warn().Msgf("Unsupported log level [%s]", l)
	}

	zerolog.SetGlobalLevel(logLevel)
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	log.Logger = log.With().Timestamp().Stack().Caller().Logger()
}
