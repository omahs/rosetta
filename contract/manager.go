package contract

import (
	"sync"

	eth "github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type ContractManager struct {
	LockedGold *LockedGold
	// stableToken *StableToken
	// election    *Election
	RwMutex *sync.RWMutex
}

func NewContractManager(rpcClient *rpc.Client) (*ContractManager, error) {
	ethClient := eth.NewClient(rpcClient)

	registry, err := NewRegistry(RegistrySmartContractAddress, ethClient)
	if err != nil {
		return nil, err
	}

	lockedGoldAddr, err := registry.GetAddressFor(nil, LockedGoldRegistryId)
	if err != nil {
		return nil, err
	}

	lockedGold, err := NewLockedGold(lockedGoldAddr, ethClient)
	if err != nil {
		return nil, err
	}

	regUpdateChannel := make(chan *RegistryRegistryUpdated)
	sub, err := registry.WatchRegistryUpdated(nil, regUpdateChannel, [][32]byte{
		LockedGoldRegistryId,
		// StableTokenRegistryId,
		// ElectionRegistryId,
	})

	// TODO(yorke): close sub gracefully somewhere
	_ = sub

	var rwMutex sync.RWMutex

	go func() {
		for {
			regUpdated := *<-regUpdateChannel
			rwMutex.Lock()
			switch regUpdated.Identifier {
			case LockedGoldName:
				lockedGold, err = NewLockedGold(regUpdated.Addr, ethClient)
			}
			rwMutex.Unlock()
		}
	}()

	return &ContractManager{
		LockedGold: lockedGold,
		RwMutex:    &rwMutex,
	}, nil
}
