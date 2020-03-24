package contract

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func nameToHash(name string) common.Hash {
	return crypto.Keccak256Hash([]byte(name))
}

const (
	LockedGoldName  = "LockedGold"
	StableTokenName = "StableToken"
	ElectionName    = "Election"
)

var (
	RegistrySmartContractAddress = common.HexToAddress("0x000000000000000000000000000000000000ce10")
	LockedGoldRegistryId         = nameToHash(LockedGoldName)
	StableTokenRegistryId        = nameToHash(StableTokenName)
	ElectionRegistryId           = nameToHash(ElectionName)
)
