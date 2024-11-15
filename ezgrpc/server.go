package ezgrpc

import (
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

type Service struct {
	Desc *grpc.ServiceDesc
	Ss   any
}

type Server struct {
	options Options

	logger *log.Logger

	server *grpc.Server
	// opts   []grpc.ServerOption
}

func (d *Server) init(opts ...Option) {
	for _, o := range opts {
		o(&d.options)
	}
}

func New(opts ...Option) *Server {
	options := Options{
		host:   "",
		port:   9090,
		health: false,
	}

	srv := &Server{options: options, logger: log.New(os.Stdout, "|GRPC_SERVER| ", log.LstdFlags)}
	srv.init(opts...)

	return srv
}

func (srv *Server) Start(services []Service, opts ...grpc.ServerOption) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", srv.options.host, srv.options.port))
	if err != nil {
		return err
	}

	server := grpc.NewServer(opts...)

	if srv.options.health {
		server.RegisterService(&healthgrpc.Health_ServiceDesc, health.NewServer())
	}
	for i := range services {
		server.RegisterService(services[i].Desc, services[i].Ss)
	}

	srv.server = server
	srv.logger.Printf("grpc server started at %s:%d\n", srv.options.host, srv.options.port)

	// err = server.Serve(lis)
	// if err != nil {
	// 	panic(err)
	// }
	go func() {
		err = server.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	return nil

}

func (srv *Server) Close() {
	srv.server.Stop()
}
