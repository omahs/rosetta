package monitor

import (
	"context"

	"github.com/celo-org/rosetta/db"
	"github.com/ethereum/go-ethereum/log"
)

func ProcessChanges(ctx context.Context, changes <-chan *db.BlockChangeSet, dbWriter db.RosettaDBWriter, logger log.Logger) error {
	for changeSet := range changes {
		if err := dbWriter.ApplyChanges(ctx, changeSet); err != nil {
			return err
		}
		logger.Info("Stored block changes", "block", changeSet.BlockNumber, "registryUpdates", len(changeSet.RegistryChanges))
	}
	return nil
}
