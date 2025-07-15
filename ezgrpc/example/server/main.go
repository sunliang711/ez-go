package main

import (
	"context"

	"github.com/sunliang711/ez-go/ezgrpc"
	"github.com/sunliang711/ez-go/eztools"
	"gitlab.atom8.io/hkbitex/exchange-protos/pbgo/ledger_manager_service"
	"google.golang.org/grpc"
)

func main() {
	grpcServer := ezgrpc.New(ezgrpc.WithPort(9000), ezgrpc.WithHealth())
	srv := MyService{}
	services := []ezgrpc.Service{{
		Desc: &ledger_manager_service.LedgerManagerService_ServiceDesc,
		Ss:   &srv,
	}}
	opts := []grpc.ServerOption{}

	opts = append(opts,
		grpc.ChainUnaryInterceptor(
			ezgrpc.RequestParamInterceptor([]string{"/ledger_manager_service.LedgerManagerService/ListAsset"}),
			ezgrpc.ResponseInterceptor([]string{"/ledger_manager_service.LedgerManagerService/ListAsset"}, 1024*1024),
			ezgrpc.ValidatorInterceptor(),
		),
	)

	grpcServer.Start(services, opts...)

	eztools.WaitForSignal(nil)

}

// MyService is the implementation of the LedgerManagerService
type MyService struct {
	ledger_manager_service.UnimplementedLedgerManagerServiceServer
}

func (s *MyService) ListAsset(ctx context.Context, req *ledger_manager_service.ListAssetReq) (*ledger_manager_service.ListAssetRes, error) {
	return &ledger_manager_service.ListAssetRes{
		Assets: []*ledger_manager_service.Asset{{
			Id:   0,
			Name: "HKD",
			Type: "Fiat",
			Icon: "icon",
		}},
		Total: 0,
	}, nil
}
