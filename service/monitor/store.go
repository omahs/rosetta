package monitor

import (
	"context"

	"github.com/celo-org/rosetta/db"
)

func ProcessChanges(ctx context.Context, changes <-chan *db.BlockChangeSet, dbWriter db.RosettaDBWriter) error {
	for changeSet := range changes {
		if err := dbWriter.ApplyChanges(ctx, changeSet); err != nil {
			return err
		}
	}
	return nil
}
