package monitor

import (
	"context"
	"math/big"

	"github.com/celo-org/rosetta/celo/client"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/k0kubun/pp"
)

func HeaderListener(ctx context.Context, results chan<- *types.Header, cc *client.CeloClient, logger log.Logger, startBlock *big.Int) error {

	lastBlock, err := lastNodeBlockNumber(ctx, cc)
	if err != nil {
		return err
	}

	logger.Debug("Fetching old block", "start", startBlock, "end", lastBlock)
	if err = fetchHeaderRange(ctx, results, cc, logger, startBlock, lastBlock); err != nil {
		return err
	}

	return newHeadersSubscriber(ctx, results, cc)
}

func fetchHeaderRange(ctx context.Context, results chan<- *types.Header, cc *client.CeloClient, logger log.Logger, startBlock, endBlock *big.Int) error {
	for i := startBlock; i.Cmp(endBlock) < 0; i.Add(i, big.NewInt(1)) {
		logger.Debug("Fetching bloc", "block", i)
		h, err := cc.Eth.HeaderByNumber(ctx, i)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case results <- h:
		}
	}
	return nil
}

func newHeadersSubscriber(ctx context.Context, result chan<- *types.Header, cc *client.CeloClient) error {
	sub, err := cc.Eth.SubscribeNewHead(ctx, result)
	if err != nil {
		return err
	}

	defer sub.Unsubscribe()

	select {
	case err := <-sub.Err():
		return err
	case <-ctx.Done():
		return nil
	}
}

func lastNodeBlockNumber(ctx context.Context, cc *client.CeloClient) (*big.Int, error) {
	latest, err := cc.Eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	pp.Println(latest.Number.String(), latest.Hash().Hex(), latest.ParentHash.Hex())
	pp.Print(latest)
	return latest.Number, nil
}
