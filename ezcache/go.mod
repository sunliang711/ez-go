module github.com/sunliang711/ez-go/ezcache

go 1.21.1

require (
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/redis/go-redis/v9 v9.11.0
	github.com/rs/zerolog v1.31.0
	github.com/sunliang711/ez-go/ezlog v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

replace github.com/sunliang711/ez-go/ezlog => ../ezlog
