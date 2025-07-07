package ezcache

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Log 如果 enable 为 true，则打印日志
// 与 ezhttp 的日志风格保持一致
func Log(enable bool, level zerolog.Level, format string, args ...any) {
	if !enable {
		return
	}

	switch level {
	case zerolog.TraceLevel:
		log.Trace().Msgf(format, args...)
	case zerolog.DebugLevel:
		log.Debug().Msgf(format, args...)
	case zerolog.InfoLevel:
		log.Info().Msgf(format, args...)
	case zerolog.WarnLevel:
		log.Warn().Msgf(format, args...)
	case zerolog.ErrorLevel:
		log.Error().Msgf(format, args...)
	case zerolog.FatalLevel:
		log.Fatal().Msgf(format, args...)
	}
}
