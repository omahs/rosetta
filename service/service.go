package service

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

var ErrAlreadyRunning = errors.New("Service already running")

type Service interface {
	// Name retrieves the name of the service, that will be used
	// to identify the service in log messages
	Name() string

	// Running indicates if the service is currently running
	Running() bool

	// Start runs the service and blocks until the service finishes,
	// returns an error when service failed
	Start(ctx context.Context) error
}

// Rosetta HTTP Service
// Geth Node Service
// Monitor Service (package monitor)

func RunServices(ctx context.Context, services ...Service) error {
	ctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	var wg sync.WaitGroup
	wg.Add(len(services))

	errors := NewErrorCollector()
	for _, _srv := range services {
		go func(srv Service) {
			defer wg.Done()
			log.Info("Starting service", "srv", srv.Name())
			err := srv.Start(ctx)
			if err != nil {
				// Stop Services on the first error occured
				cancelFn()
				errors.Add(err)
			}
		}(_srv)
	}

	wg.Wait()
	return errors.Error()
}
