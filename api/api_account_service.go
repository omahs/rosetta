/*
 * Rosetta
 *
 * A standard for blockchain interaction
 *
 * API version: 1.2.3
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package api

import (
	"context"

	"github.com/celo-org/rosetta/celo"
	"github.com/celo-org/rosetta/celo/client"
	"github.com/ethereum/go-ethereum/common"
)

// AccountApiService is a service that implents the logic for the AccountApiServicer
// This service should implement the business logic for every endpoint for the AccountApi API.
// Include any external packages or services that will be required by this service.
type AccountApiService struct {
	celoClient  *client.CeloClient
	chainParams *celo.ChainParameters
}

// NewAccountApiService creates a default api service
func NewAccountApiService(celoClient *client.CeloClient, chainParams *celo.ChainParameters) AccountApiServicer {
	return &AccountApiService{
		celoClient:  celoClient,
		chainParams: chainParams,
	}
}

// AccountBalance - Get an Account Balance
func (s *AccountApiService) AccountBalance(ctx context.Context, accountBalanceRequest AccountBalanceRequest) (interface{}, error) {

	err := ValidateNetworkId(&accountBalanceRequest.NetworkIdentifier, s.chainParams)
	if err != nil {
		return nil, err
	}

	address := common.HexToAddress(accountBalanceRequest.AccountIdentifier.Address)

	latestHeader, err := s.celoClient.Eth.HeaderByNumber(ctx, nil) // nil == latest
	if err != nil {
		err = client.WrapRpcError(err)
		return nil, ErrRpcError("HeaderByNumber", err)
	}

	goldBalance, err := s.celoClient.Eth.BalanceAt(ctx, address, latestHeader.Number)
	if err != nil {
		err = client.WrapRpcError(err)
		return nil, ErrRpcError("BalanceAt", err)
	}

	// TODO Not Supported for now
	// registryWrapper, err := wrapper.NewRegistry(s.celoClient)
	// if err != nil {
	// 	return nil, err
	// }

	// lockedGoldAddr, err := registryWrapper.GetAddressFor(&bind.CallOpts{
	// 	BlockNumber: latestHeader.Number,
	// 	Context:     ctx,
	// }, params.LockedGoldRegistryId)
	// if err != nil {
	// 	return nil, err
	// }

	// lockedGold, err := contract.NewLockedGold(lockedGoldAddr, s.celoClient.Eth)
	// if err != nil {
	// 	err = client.WrapRpcError(err)
	// 	return nil, ErrRpcError("NewLockedGold", err)
	// }

	// lockedGoldBalance, err := lockedGold.GetAccountTotalLockedGold(&bind.CallOpts{
	// 	BlockNumber: latestHeader.Number,
	// 	Context:     ctx,
	// }, address)
	// if err != nil {
	// 	err = client.WrapRpcError(err)
	// 	return nil, ErrRpcError("GetAccountTotalLockedGold", err)
	// }

	response := AccountBalanceResponse{
		BlockIdentifier: *HeaderToBlockIdentifier(latestHeader),
		Balances: []Balance{
			Balance{
				AccountIdentifier: accountBalanceRequest.AccountIdentifier,
				Amounts: []Amount{
					Amount{
						Value:    goldBalance.String(),
						Currency: CeloGold,
					},
				},
			},
			// Balance{
			// 	AccountIdentifier: AccountIdentifier{
			// 		Address: accountBalanceRequest.AccountIdentifier.Address,
			// 		SubAccount: SubAccountIdentifier{
			// 			SubAccount: "LockedGold",
			// 		},
			// 	},
			// 	Amounts: []Amount{
			// 		Amount{
			// 			Value:    lockedGoldBalance.String(),
			// 			Currency: CeloGold,
			// 		},
			// 	},
			// },
		},
	}
	return response, nil
}
