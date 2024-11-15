package main

import (
	"context"
	"fmt"
	"time"

	"gitlab.atom8.io/hkbitex/exchange-protos/pbgo/ledger_manager_service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	opt := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.NewClient("localhost:9000", opt)
	if err != nil {
		panic(err)
	}

	client := ledger_manager_service.NewLedgerManagerServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	res, err := client.ListAsset(ctx, &ledger_manager_service.ListAssetReq{
		Type:   0,
		Limit:  0,
		Offset: 0,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("res: %v\n", res)
}
