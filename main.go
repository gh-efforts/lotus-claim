package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
)

func main() {
	var providerAddrStr string
	var startEpoch, endEpoch int64

	app := &cli.App{
		Name:  "claim-size-calculator",
		Usage: "计算指定provider在给定epoch范围内的所有claim的Size总和,按Client分组",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "provider",
				Usage:       "Provider地址",
				Destination: &providerAddrStr,
				Required:    true,
			},
			&cli.Int64Flag{
				Name:        "start",
				Usage:       "开始Epoch",
				Destination: &startEpoch,
				Required:    true,
			},
			&cli.Int64Flag{
				Name:        "end",
				Usage:       "结束Epoch",
				Destination: &endEpoch,
				Required:    true,
			},
		},
		Action: func(cctx *cli.Context) error {
			ctx := lcli.ReqContext(cctx)

			providerAddr, err := address.NewFromString(providerAddrStr)
			if err != nil {
				return fmt.Errorf("无效的Provider地址: %w", err)
			}

			api, closer, err := lcli.GetFullNodeAPIV1(cctx)
			if err != nil {
				return fmt.Errorf("无法获取Lotus API: %w", err)
			}
			defer closer()

			return calculateClaimSizeByClient(ctx, api, providerAddr, startEpoch, endEpoch)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func calculateClaimSizeByClient(ctx context.Context, api api.FullNode, providerAddr address.Address, startEpoch, endEpoch int64) error {
	claims, err := api.StateGetClaims(ctx, providerAddr, types.EmptyTSK)
	if err != nil {
		return fmt.Errorf("调用StateGetClaims失败: %w", err)
	}

	clientSizes := make(map[abi.ActorID]abi.PaddedPieceSize)
	for _, claim := range claims {
		if claim.TermStart >= abi.ChainEpoch(startEpoch) && claim.TermStart <= abi.ChainEpoch(endEpoch) {
			clientSizes[claim.Client] += claim.Size
		}
	}

	// 将结果排序并打印
	type clientSize struct {
		client abi.ActorID
		size   abi.PaddedPieceSize
	}
	var sortedSizes []clientSize
	for client, size := range clientSizes {
		sortedSizes = append(sortedSizes, clientSize{client, size})
	}
	sort.Slice(sortedSizes, func(i, j int) bool {
		return sortedSizes[i].size > sortedSizes[j].size
	})

	fmt.Printf("Provider %s 在Epoch %d 到 %d 之间的Claim Size统计 (按Client分组):\n",
		providerAddr, startEpoch, endEpoch)
	for _, cs := range sortedSizes {
		fmt.Printf("Client %d: %d\n", cs.client, cs.size)
	}

	return nil
}
