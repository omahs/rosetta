package monitor

import (
	"context"

	"github.com/celo-org/rosetta/celo/client"
	"github.com/celo-org/rosetta/db"
	"github.com/ethereum/go-ethereum/core/types"
)

func BlockProcessor(ctx context.Context, headers <-chan *types.Header, changes chan<- *db.BlockChangeSet, cc *client.CeloClient) error {

	for header := range headers {
		bcs := db.BlockChangeSet{
			BlockNumber: header.Number,
		}

		// TODO implement
		select {
		case <-ctx.Done():
			return ctx.Err()
		case changes <- &bcs:
		}
	}

	return nil
}
