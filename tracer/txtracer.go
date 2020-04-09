package tracer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/celo-org/rosetta/celo/client"
	"github.com/celo-org/rosetta/celo/client/debug"
	"github.com/celo-org/rosetta/celo/contract"
	"github.com/celo-org/rosetta/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type TxTracer struct {
	blockHeader *types.Header
	tx          *types.Transaction
	receipt     *types.Receipt

	celoClient *client.CeloClient
	db         db.RosettaDBReader
	ctx        context.Context
	logger     log.Logger
}

func NewTxTracer(
	ctx context.Context,
	celoClient *client.CeloClient,
	db db.RosettaDBReader,
	blockHeader *types.Header,
	tx *types.Transaction,
	receipt *types.Receipt,
) *TxTracer {
	logger := log.New("context", "TxTracer", "txHash", tx.Hash().Hex(), "txIndex", receipt.TransactionIndex, "blockHash", blockHeader.Hash().Hex(), "blockNumber", blockHeader.Number)
	return &TxTracer{
		celoClient:  celoClient,
		db:          db,
		blockHeader: blockHeader,
		tx:          tx,
		receipt:     receipt,
		ctx:         ctx,
		logger:      logger,
	}
}

func (tc *TxTracer) GasDetail() (map[common.Address]*big.Int, error) {
	balanceChanges := make(map[common.Address]*big.Int)

	governanceAddress, err := tc.db.RegistryAddressOn(tc.ctx, tc.blockHeader.Number, tc.receipt.TransactionIndex, "Governance")
	if err != nil && err != db.ErrContractNotFound {
		return nil, err
	}

	// TODO - What happens when there's no gasPrice Minimun
	gasPriceMinimum, err := tc.db.GasPriceMinimunOn(tc.ctx, tc.blockHeader.Number)
	if err != nil {
		return nil, err
	}

	gasUsed := new(big.Int).SetUint64(tc.receipt.GasUsed)

	// baseTxFee is what goes to the community fund (if any)
	baseTxFee := new(big.Int).Mul(gasPriceMinimum, gasUsed)
	totalTxFee := new(big.Int).Mul(tc.tx.GasPrice(), gasUsed)

	// The "tip" goes to the coinbase address
	balanceChanges[tc.blockHeader.Coinbase] = new(big.Int).Sub(totalTxFee, baseTxFee)

	if governanceAddress != common.ZeroAddress {
		// The baseTxFee goes to the community fund
		balanceChanges[governanceAddress] = baseTxFee
	} else {
		// No community fund, we won't charge the user
		totalTxFee.Sub(totalTxFee, baseTxFee)
	}

	if tc.tx.GatewayFeeRecipient() != nil {
		balanceChanges[*tc.tx.GatewayFeeRecipient()] = tc.tx.GatewayFee()
		totalTxFee.Add(totalTxFee, tc.tx.GatewayFee())
	}

	// TODO find a better way to do this?
	from, err := tc.celoClient.Eth.TransactionSender(tc.ctx, tc.tx, tc.blockHeader.Hash(), tc.receipt.TransactionIndex)
	if err != nil {
		return balanceChanges, err
	}
	balanceChanges[from] = new(big.Int).Neg(totalTxFee)

	return balanceChanges, nil
}

func (tc *TxTracer) TransferDetail() ([]Transfer, error) {
	if tc.receipt.Status == types.ReceiptStatusFailed {
		return nil, nil
	}

	internalTransfers, err := tc.celoClient.Debug.TransactionTransfers(tc.ctx, tc.tx.Hash())
	if err != nil {
		return nil, err
	}

	transfers := make([]Transfer, len(internalTransfers))
	for i, it := range internalTransfers {
		transfers[i] = Transfer{
			From: Account{
				Address:    it.From,
				SubAccount: Main,
			},
			To: Account{
				Address:    it.To,
				SubAccount: Main,
			},
			Value:  it.Value,
			Status: it.Status.String() == debug.TransferStatusSuccess.String(),
		}
	}
	return transfers, nil
}

func (tc *TxTracer) LockedGoldTransferDetail() ([]Transfer, error) {

	registryAddresses, err := tc.db.RegistryAddressesOn(tc.ctx, tc.blockHeader.Number, tc.receipt.TransactionIndex, "Governance", "LockedGold")
	if err != nil {
		return nil, err
	}

	lockedGoldAddr, ok := registryAddresses["LockedGold"]

	// TODO deal with Error

	// LockedGold not found (not deployed) => no transfers
	if !ok {
		return nil, nil
	}

	lockedGold, err := contract.NewLockedGold(lockedGoldAddr, tc.celoClient.Eth)
	if err != nil {
		return nil, err
	}

	transfers := make([]Transfer, 0, len(tc.receipt.Logs))
	for _, eventLog := range tc.receipt.Logs {
		eventName, eventRaw, ok, err := lockedGold.TryParseLog(*eventLog)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		switch eventName {
		case "AccountSlashed":
			governanceAddr, ok := registryAddresses["Govenance"]
			if !ok {
				return nil, fmt.Errorf("Can't slash before governance is deployed")
			}

			event := eventRaw.(contract.LockedGoldAccountSlashed)
			transfers = append(transfers,
				Transfer{
					From: Account{
						Address:    event.Slashed,
						SubAccount: LockedGoldLocked,
					},
					To: Account{
						Address:    event.Reporter,
						SubAccount: LockedGoldLocked,
					},
					Value: event.Reward,
				},
				Transfer{
					From: Account{
						Address:    event.Slashed,
						SubAccount: LockedGoldLocked,
					},
					To: Account{
						Address:    governanceAddr,
						SubAccount: Main,
					},
					Value: new(big.Int).Sub(event.Penalty, event.Reward),
				},
			)

		case "GoldLocked":
			event := eventRaw.(contract.LockedGoldGoldLocked)
			transfers = append(transfers, Transfer{
				From: Account{
					Address:    event.Account,
					SubAccount: Main,
				},
				To: Account{
					Address:    event.Account,
					SubAccount: LockedGoldLocked,
				},
				Value: event.Value,
			})

		case "GoldUnlocked":
			event := eventRaw.(contract.LockedGoldGoldUnlocked)
			transfers = append(transfers, Transfer{
				From: Account{
					Address:    event.Account,
					SubAccount: LockedGoldLocked,
				},
				To: Account{
					Address:    event.Account,
					SubAccount: LockedGoldPending,
				},
				Value: event.Value,
			})
		case "GoldWithdrawn":
			event := eventRaw.(contract.LockedGoldGoldWithdrawn)
			transfers = append(transfers, Transfer{
				From: Account{
					Address:    event.Account,
					SubAccount: LockedGoldPending,
				},
				To: Account{
					Address:    event.Account,
					SubAccount: Main,
				},
				Value: event.Value,
			})
		}

	}

	return transfers, nil
}

/*

compute

Compute Operations
 - compute Gas Operations
 - compute Transfer Operations

Compute Gas Operations:
	- If feeCurrency != cGLD => EXIT
	- From Block => coinbase, number
	- From tx => gatewayFee, gasPrice, gatewayFeeRecipient, index
	- From receipt => gasUsed, status
	- From Registry: Governance, GasPriceMinimun
	- From GasPriceMinimun: gasPriceMinimum
	- Error Conditions:
		- Fail to fetch Governance => No communityFund Fee
		- Fail to fetch gasPriceMinimun => ?
		- Fail to obtain block, tx, receipt => error

Compute Transfer Operations;
	- From receipt => status
	- If status == failure => EXIT (no transfers)
	- Call debug_traceTransaction
	- From Registry: LockedGold
	- From LockedGold => All Events
	- Error Conditions:
		- Fail to fetch LockedGold.add => not deployed, ignore lockedGold subaccount
		- Fail to debug_traceTRansactions =>


{
	From: {
		account common.Address
		subaccount *SubAccountKind
	}
	To: {
		account common.Address
		subaccount *SubAccountKind
	}
	Value: *big.Int


}

*/
