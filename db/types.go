package db

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var ErrNotFound = errors.New("db: record not found")

type RosettaDBReader interface {
	LastPersistedBlock(ctx context.Context) (*big.Int, error)
	GasPriceMinimunOn(ctx context.Context, block *big.Int) (*big.Int, error)
	RegistryAddressOn(ctx context.Context, block *big.Int, txIndex uint, contractName string) (common.Address, error)
	RegistryAddressesOn(ctx context.Context, block *big.Int, txIndex uint, contractName ...string) (map[string]common.Address, error)
}

type RosettaDBWriter interface {
	ApplyChanges(ctx context.Context, changeSet *BlockChangeSet) error
}

type RosettaDB interface {
	RosettaDBReader
	RosettaDBWriter
}

type RegistryChange struct {
	TxIndex    uint
	Contract   string
	NewAddress common.Address
}

type BlockChangeSet struct {
	BlockNumber     *big.Int
	GasPriceMinimun *big.Int
	RegistryChanges []RegistryChange
}
