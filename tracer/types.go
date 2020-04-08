package tracer

import (
	"math/big"

	"github.com/celo-org/rosetta/celo/client"
	"github.com/celo-org/rosetta/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type SubAccount string

const (
	Main              SubAccount = "Main"
	LockedGoldLocked  SubAccount = "LockedGoldLocked"
	LockedGoldPending SubAccount = "LockedGoldPending"
)

type Account struct {
	Address    common.Address
	SubAccount SubAccount
}

type Transfer struct {
	From  Account
	To    Account
	Value *big.Int
}

type AppContext struct {
	Db db.RosettaDB
	Cc client.CeloClient
}

type TxContext struct {
	blockHeader *types.Header
	tx          *types.Transaction
	receipt     *types.Receipt
}
