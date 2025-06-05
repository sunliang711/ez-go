package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sunliang711/ez-go/ezlog"
)

func main() {
	ezlog.SetLog(
		ezlog.WithCaller(),
		ezlog.WithTimestamp(),
		ezlog.WithLevel(zerolog.DebugLevel),
		ezlog.WithServiceName("aa"),
		ezlog.WithWriter(ezlog.Split),
	)

	log.Debug().Msg("This is a debug message")
	log.Info().Msg("This is an info message")
	log.Warn().Msg("This is a warning message")
	log.Error().Msg("This is an error message")
	log.Fatal().Msg("This is a fatal message")

}
