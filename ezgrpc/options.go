package ezgrpc

type Options struct {
	host      string
	port      int
	health    bool
	enableLog bool
}

type Option func(*Options)

func WithHost(host string) Option {
	return func(so *Options) {
		so.host = host
	}
}

func WithPort(port int) Option {
	return func(so *Options) {
		so.port = port
	}
}

func WithHealth() Option {
	return func(so *Options) {
		so.health = true
	}
}

func WithEnableLog(enable bool) Option {
	return func(so *Options) {
		so.enableLog = enable
	}
}
