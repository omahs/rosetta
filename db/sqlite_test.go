package db

import (
	"context"
	"math/big"
	"testing"
)

var _ RosettaDB = (*rosettaSqlDb)(nil)

func TestApplyChanges(t *testing.T) {
	celoDb, err := NewSqliteDb(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	changeSet := BlockChangeSet{
		BlockNumber: big.NewInt(10),
	}

	err = celoDb.ApplyChanges(context.Background(), &changeSet)
	if err != nil {
		t.Fatal(err)
	}
}
