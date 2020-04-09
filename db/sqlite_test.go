package db

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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

func TestRegisterContract(t *testing.T) {
	RegisterTestingT(t)
	ctx := context.Background()

	// celoDb, err := NewSqliteDb("/tmp/prueba.db")
	celoDb, err := NewSqliteDb(":memory:")
	Ω(err).ShouldNot(HaveOccurred())

	err = celoDb.ApplyChanges(ctx, &BlockChangeSet{
		BlockNumber: big.NewInt(10),
		RegistryChanges: []RegistryChange{
			{TxIndex: 4, Contract: "Governance", NewAddress: common.HexToAddress("0x34")},
		},
	})
	Ω(err).ShouldNot(HaveOccurred())

	err = celoDb.ApplyChanges(ctx, &BlockChangeSet{
		BlockNumber: big.NewInt(15),
		RegistryChanges: []RegistryChange{
			{TxIndex: 4, Contract: "Governance", NewAddress: common.HexToAddress("0x111")},
		},
	})
	Ω(err).ShouldNot(HaveOccurred())

	var addr common.Address

	t.Run("Before", func(t *testing.T) {
		RegisterTestingT(t)
		addr, err = celoDb.RegistryAddressOn(ctx, big.NewInt(2), 8, "Governance")
		Ω(err).Should(Equal(ErrContractNotFound))
	})

	t.Run("Same Block, Before Tx", func(t *testing.T) {
		RegisterTestingT(t)
		addr, err = celoDb.RegistryAddressOn(ctx, big.NewInt(10), 3, "Governance")
		Ω(err).Should(Equal(ErrContractNotFound))
	})

	t.Run("Same Block & Tx", func(t *testing.T) {
		RegisterTestingT(t)
		addr, err = celoDb.RegistryAddressOn(ctx, big.NewInt(10), 4, "Governance")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(addr).Should(Equal(common.HexToAddress("0x34")))
	})

	t.Run("Same Block & After Tx", func(t *testing.T) {
		RegisterTestingT(t)
		addr, err = celoDb.RegistryAddressOn(ctx, big.NewInt(10), 6, "Governance")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(addr).Should(Equal(common.HexToAddress("0x34")))
	})

	t.Run("After Block & Before Tx", func(t *testing.T) {
		RegisterTestingT(t)
		addr, err = celoDb.RegistryAddressOn(ctx, big.NewInt(11), 3, "Governance")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(addr).Should(Equal(common.HexToAddress("0x34")))
	})

	t.Run("After Next Change", func(t *testing.T) {
		RegisterTestingT(t)
		addr, err = celoDb.RegistryAddressOn(ctx, big.NewInt(16), 3, "Governance")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(addr).Should(Equal(common.HexToAddress("0x111")))
	})

}

func TestGasPriceMinimum(t *testing.T) {
	RegisterTestingT(t)
	ctx := context.Background()

	// celoDb, err := NewSqliteDb("/tmp/prueba.db")
	celoDb, err := NewSqliteDb(":memory:")
	Ω(err).ShouldNot(HaveOccurred())

	err = celoDb.ApplyChanges(ctx, &BlockChangeSet{
		BlockNumber:     big.NewInt(10),
		GasPriceMinimun: big.NewInt(50000),
	})
	Ω(err).ShouldNot(HaveOccurred())

	err = celoDb.ApplyChanges(ctx, &BlockChangeSet{
		BlockNumber:     big.NewInt(15),
		GasPriceMinimun: big.NewInt(100000),
	})
	Ω(err).ShouldNot(HaveOccurred())

	var gpm *big.Int

	t.Run("Before", func(t *testing.T) {
		RegisterTestingT(t)
		gpm, err = celoDb.GasPriceMinimunOn(ctx, big.NewInt(2))
		Ω(err).ShouldNot(HaveOccurred())
		Ω(gpm.Uint64()).Should(Equal(uint64(0)))
	})

	t.Run("Same Block", func(t *testing.T) {
		RegisterTestingT(t)
		gpm, err = celoDb.GasPriceMinimunOn(ctx, big.NewInt(10))
		Ω(err).ShouldNot(HaveOccurred())
		Ω(gpm.Uint64()).Should(Equal(uint64(50000)))
	})

	t.Run("After Block", func(t *testing.T) {
		RegisterTestingT(t)
		gpm, err = celoDb.GasPriceMinimunOn(ctx, big.NewInt(11))
		Ω(err).ShouldNot(HaveOccurred())
		Ω(gpm.Uint64()).Should(Equal(uint64(50000)))
	})

	t.Run("After Next Change", func(t *testing.T) {
		RegisterTestingT(t)
		gpm, err = celoDb.GasPriceMinimunOn(ctx, big.NewInt(16))
		Ω(err).ShouldNot(HaveOccurred())
		Ω(gpm.Uint64()).Should(Equal(uint64(100000)))
	})

}
