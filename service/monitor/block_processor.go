package monitor

import (
	"context"
	"math/big"

	"github.com/celo-org/rosetta/celo/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type registryChange struct {
	txIndex    uint
	contract   string
	newAddress common.Address
}

type BlockChangeSet struct {
	blockNumber     *big.Int
	gasPriceMinimun *big.Int
	registryChanges []registryChange
}

func BlockProcessor(ctx context.Context, headers <-chan *types.Header, changes chan<- *BlockChangeSet, cc *client.CeloClient) error {

	for header := range headers {
		bcs := BlockChangeSet{
			blockNumber: header.Number,
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case changes <- &bcs:
		}
	}

	return nil
}
