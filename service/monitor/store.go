package monitor

import (
	"context"

	"github.com/celo-org/rosetta/db"
	"github.com/ethereum/go-ethereum/log"
)

func ProcessChanges(ctx context.Context, changes <-chan *db.BlockChangeSet, dbWriter db.RosettaDBWriter, logger log.Logger) error {
	for {
		select {
		case changeSet := <-changes:
			if err := dbWriter.ApplyChanges(ctx, changeSet); err != nil {
				return err
			}
			logger.Info("Stored block changes", "block", changeSet.BlockNumber, "registryUpdates", len(changeSet.RegistryChanges))
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
