package ezlog

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// levelFilterWriter filters logs based on their level
type levelFilterWriter struct {
	writer   io.Writer
	minLevel zerolog.Level
	maxLevel zerolog.Level
}

// Write implements the io.Writer interface.
// It checks if the log entry's level falls within the defined range
func (w levelFilterWriter) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}

// WriteLevel implements the zerolog.LevelWriter interface
func (w levelFilterWriter) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	if level >= w.minLevel && level <= w.maxLevel {
		return w.writer.Write(p)
	}
	return len(p), nil
}

type Writer int8

const (
	Stdout Writer = iota + 1
	Stderr
	Split
)

type Options struct {
	withServiceName bool
	serviceName     string

	withCaller    bool
	withTimestamp bool

	level zerolog.Level

	writer Writer
}

type Option func(*Options)

func WithServiceName(name string) Option {
	return func(o *Options) {
		o.withServiceName = true
		o.serviceName = name
	}
}

func WithCaller(v bool) Option {
	return func(o *Options) {
		o.withCaller = v
	}
}

func WithTimestamp(v bool) Option {
	return func(o *Options) {
		o.withTimestamp = v
	}
}

func WithLevel(level zerolog.Level) Option {
	return func(o *Options) {
		o.level = level
	}
}

func WithWriter(writer Writer) Option {
	return func(o *Options) {
		o.writer = writer
	}
}

var options Options

func init() {
	// default options
	options = Options{
		level:  zerolog.ErrorLevel,
		writer: Stdout,
	}
}

func SetLog(opts ...Option) {
	for _, opt := range opts {
		opt(&options)
	}

	var w io.Writer
	switch options.writer {
	case Stdout:
		w = os.Stdout
	case Stderr:
		w = os.Stderr
	case Split:
		stdoutWriter := levelFilterWriter{
			writer:   os.Stdout,
			minLevel: zerolog.TraceLevel,
			maxLevel: zerolog.InfoLevel,
		}

		stderrWriter := levelFilterWriter{
			writer:   os.Stderr,
			minLevel: zerolog.WarnLevel,
			maxLevel: zerolog.PanicLevel,
		}

		// Create multi-writer for all outputs
		multi := zerolog.MultiLevelWriter(stdoutWriter, stderrWriter)
		w = multi
	}

	// Configure global logger
	logContext := zerolog.New(w).With()

	if options.withServiceName {
		logContext = logContext.Str("service", options.serviceName)
	}
	if options.withCaller {
		logContext = logContext.Caller()
	}
	if options.withTimestamp {
		logContext = logContext.Timestamp()
	}

	log.Logger = logContext.Logger()
	// Set global log level
	zerolog.SetGlobalLevel(options.level)
}
