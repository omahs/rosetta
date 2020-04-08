package db

import (
	"context"
	"math/big"
	"testing"

	. "github.com/onsi/gomega"
)

var _ RosettaDB = (*rosettaSqlDb)(nil)

func TestApplyChanges(t *testing.T) {
	RegisterTestingT(t)

	celoDb, err := NewSqliteDb(":memory:")
	Ω(err).ShouldNot(HaveOccurred())

	blockNumber := big.NewInt(10)
	changeSet := BlockChangeSet{
		BlockNumber: blockNumber,
	}

	err = celoDb.ApplyChanges(context.Background(), &changeSet)
	Ω(err).ShouldNot(HaveOccurred())

	storedBlockNumber, err := celoDb.LastPersistedBlock(context.Background())
	Ω(err).ShouldNot(HaveOccurred())

	Ω(storedBlockNumber.String()).To(Equal(blockNumber.String()))
}
