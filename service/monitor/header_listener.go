package monitor

import (
	"context"
	"math/big"
	"sync"

	"github.com/celo-org/rosetta/celo/client"
	"github.com/celo-org/rosetta/service"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

func HeaderListener(ctx context.Context, results chan<- *types.Header, cc *client.CeloClient, logger log.Logger, startBlock *big.Int) error {
	logger = logger.New("pipe", "header_listener")

	lastBlock, err := lastNodeBlockNumber(ctx, cc)
	if err != nil {
		return err
	}

	logger.Info("Start fetching old blocks", "start", startBlock, "end", lastBlock)
	if err = fetchHeaderRange(ctx, results, cc, logger, startBlock, lastBlock); err != nil {
		return err
	}
	logger.Info("Finished fetching old blocks", "start", startBlock, "end", lastBlock)

	return newHeadersSubscriber(ctx, results, cc, logger, new(big.Int).Add(lastBlock, big.NewInt(1)))
}

func fetchHeaderRange(ctx context.Context, results chan<- *types.Header, cc *client.CeloClient, logger log.Logger, startBlock, endBlock *big.Int) error {

	for i := startBlock; i.Cmp(endBlock) < 0; i.Add(i, big.NewInt(1)) {
		results := make([]*types.Header, 10)

		startBlock.Bytes()
		var wg sync.WaitGroup
		wg.Add(10)
		var errorCollector service.ErrorCollector
		for ri := 0; ri < 10; ri++ {
			go func(index int) {
				defer wg.Done()
				h, err := cc.Eth.HeaderByNumber(ctx, i)
				if err != nil {
					errorCollector.Add(err)
				}
				results[index] = h // Is this thread safe?
			}(ri)
		}
		wg.Wait()

		h, err := cc.Eth.HeaderByNumber(ctx, i)
		if err != nil {
			return err
		}

		logger.Trace("Block Fetched", "block", i)
		if i.Uint64()%100 == 0 {
			logger.Info("Fetched 100 Blocks", "from", i.Uint64()-99, "to", i)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case results <- h:
		}
	}
	return nil
}

func newHeadersSubscriber(ctx context.Context, result chan<- *types.Header, cc *client.CeloClient, logger log.Logger, startBlock *big.Int) error {
	intermediateCh := make(chan *types.Header, 1000000)

	sub, err := cc.Eth.SubscribeNewHead(ctx, intermediateCh)
	if err != nil {
		return err
	}

	defer sub.Unsubscribe()

	// Get the first header
	var firstHeader *types.Header
	select {
	case err := <-sub.Err():
		return err
	case <-ctx.Done():
		return nil
	case firstHeader = <-intermediateCh:
	}

	lastBlock := new(big.Int).Sub(firstHeader.Number, big.NewInt(1))
	// Now fetch intermediate blocks
	if err := fetchHeaderRange(ctx, result, cc, logger, startBlock, lastBlock); err != nil {
		return err
	}

	// Write the first header
	select {
	case <-ctx.Done():
		return ctx.Err()
	case result <- firstHeader:
	}

	// Normal operation Mode
	for {
		select {
		case err := <-sub.Err():
			return err
		case <-ctx.Done():
			return nil
		case h := <-intermediateCh:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case result <- h:
			}
		}
	}
}

func lastNodeBlockNumber(ctx context.Context, cc *client.CeloClient) (*big.Int, error) {
	latest, err := cc.Eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	return latest.Number, nil
}
