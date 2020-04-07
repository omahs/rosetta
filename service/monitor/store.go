package monitor

import (
	"context"
	"math/big"

	"github.com/celo-org/rosetta/internal/utils"
	"github.com/ethereum/go-ethereum/common"
)

type CeloStore struct {
}

func NewCeloStore() (*CeloStore, error) {
	// TODO initialize DB connection

	return &CeloStore{}, nil
}

func (cs *CeloStore) LastPersistedBlock(ctx context.Context) (*big.Int, error) {
	// TODO implement
	return nil, utils.ErrNotImplemented
}

func (cs *CeloStore) GasPriceMinimunOn(ctx context.Context, block *big.Int) (*big.Int, error) {
	// TODO implement
	return nil, utils.ErrNotImplemented
}

func (cs *CeloStore) RegistryAddressOn(ctx context.Context, block *big.Int, txIndex uint, contractName string) (common.Address, error) {
	// TODO implement
	return common.ZeroAddress, utils.ErrNotImplemented
}

func (cs *CeloStore) RegistryAddressesOn(ctx context.Context, block *big.Int, txIndex uint, contractName ...string) ([]common.Address, error) {
	// TODO implement
	return []common.Address{}, utils.ErrNotImplemented
}

func (cs *CeloStore) ProcessChanges(ctx context.Context, changes <-chan *BlockChangeSet) error {

	return nil
}
