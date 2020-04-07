package monitor

import (
	"context"
	"math/big"
	"sync"

	"github.com/celo-org/rosetta/celo/client"
	"github.com/celo-org/rosetta/service"
	"github.com/ethereum/go-ethereum/core/types"
)

type monitorService struct {
	running service.RunningLock
	cc      *client.CeloClient
}

// Name retrieves the name of the service, that will be used
// to identify the service in log messages
func (ms *monitorService) Name() string {
	return "celo-monitor"
}

// Running indicates if the service is currently running
func (ms *monitorService) Running() bool {
	return ms.running.Running()
}

// Start runs the service and blocks until the service finishes,
// returns an error when service failed
func (ms *monitorService) Start(ctx context.Context) error {
	if err := ms.running.EnableOrFail(); err != nil {
		return err
	}
	defer ms.running.Disable()

	// TODO implement (obtain start block from DB)
	var startBlock *big.Int

	ctx, stopAll := context.WithCancel(ctx)

	var wg sync.WaitGroup
	var errorCollector service.ErrorCollector

	headerCh := make(chan *types.Header)
	changeSetsCh := make(chan *BlockChangeSet)

	wg.Add(3)

	// 1st. Listen ot Header
	go func() {
		defer wg.Done()
		err := HeaderListener(ctx, headerCh, ms.cc, startBlock)
		if err != nil {
			errorCollector.Add(err)
			stopAll()
		}
	}()

	// 2nd. Process Headers
	go func() {
		defer wg.Done()
		err := BlockProcessor(ctx, headerCh, changeSetsCh, ms.cc)
		if err != nil {
			errorCollector.Add(err)
			stopAll()
		}
	}()

	// 3rd. Store Changes into DB
	go func() {
		defer wg.Done()
		err := ChangeSetStorer(ctx, changeSetsCh)
		if err != nil {
			errorCollector.Add(err)
			stopAll()
		}
	}()

	wg.Wait()
	return errorCollector.Error()
}
