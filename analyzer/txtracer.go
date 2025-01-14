// Copyright 2020 Celo Org
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package analyzer

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/celo-org/celo-blockchain/common"
	"github.com/celo-org/celo-blockchain/core/types"
	"github.com/celo-org/celo-blockchain/eth/tracers"
	"github.com/celo-org/celo-blockchain/log"
	"github.com/celo-org/kliento/client"
	"github.com/celo-org/kliento/client/debug"
	"github.com/celo-org/kliento/contracts"
	"github.com/celo-org/kliento/registry"
	"github.com/celo-org/rosetta/db"
	"github.com/celo-org/rosetta/internal/utils"
)

type Tracer struct {
	ctx          context.Context
	cc           *client.CeloClient
	db           db.RosettaDBReader
	logger       log.Logger
	traceTimeout time.Duration
	gingerbread  bool
}

func NewTracer(ctx context.Context, cc *client.CeloClient, db db.RosettaDBReader, traceTimeout time.Duration, gingerbread bool) *Tracer {
	logger := log.New("module", "tracer")
	return &Tracer{
		ctx:          ctx,
		cc:           cc,
		db:           db,
		logger:       logger,
		traceTimeout: traceTimeout,
		gingerbread:  gingerbread,
	}
}

func (tr *Tracer) GetRegistryAddresses(receipt *types.Receipt, contractNames ...string) (map[string]common.Address, error) {
	contractMap, err := tr.db.RegistryAddressesStartOf(tr.ctx, receipt.BlockNumber, receipt.TransactionIndex, contractNames...)
	if err != nil {
		return nil, fmt.Errorf("Error fetching registry addresses: %w", err)
	}

	return contractMap, nil
}

func (tr *Tracer) TraceTransaction(blockHeader *types.Header, tx *types.Transaction, receipt *types.Receipt) ([]Operation, error) {
	ops := make([]Operation, 0)

	if tx.FeeCurrency() == nil { // nil implies cGLD
		gasOp, err := tr.TxGasDetails(blockHeader, tx, receipt)
		if err != nil {
			return nil, err
		}
		ops = append(ops, *gasOp)
	}

	if receipt.Status == types.ReceiptStatusSuccessful {
		contractMap, err := tr.GetRegistryAddresses(receipt, registry.ReserveContractID.String(), registry.LockedGoldContractID.String(), registry.ElectionContractID.String(), registry.GovernanceContractID.String(), registry.AccountsContractID.String())
		if err != nil {
			return nil, err
		}

		logOps, err := tr.TxOpsFromLogs(tx, receipt, contractMap)
		if err != nil {
			return nil, err
		}

		transferOps, err := tr.TxTransfers(tx, receipt)
		if err != nil {
			return nil, err
		}

		reconciledOps, err := ReconcileLogOpsWithTransfers(logOps, transferOps)
		if err != nil {
			return nil, err
		}

		ops = append(ops, reconciledOps...)
	}

	return ops, nil
}

func (tr *Tracer) TxGasDetails(blockHeader *types.Header, tx *types.Transaction, receipt *types.Receipt) (*Operation, error) {
	balanceChanges := NewBalanceSet()

	var gpm *big.Int
	var feeHandler string

	if tr.gingerbread {
		// BaseFee is used directly because we only track balance changes from CELO gas fees
		gpm = blockHeader.BaseFee
		feeHandler = registry.FeeHandlerContractID.String()
	} else {
		var err error
		gpm, err = tr.db.GasPriceMinimumFor(tr.ctx, receipt.BlockNumber)
		if err != nil {
			return nil, fmt.Errorf("can't get gasPriceMinimun: %w", err)
		}
		feeHandler = registry.GovernanceContractID.String()
	}

	gasUsed := new(big.Int).SetUint64(receipt.GasUsed)
	baseTxFee := new(big.Int).Mul(gpm, gasUsed)
	effectiveTip, err := tx.EffectiveGasTip(gpm)
	if err != nil {
		return nil, fmt.Errorf("error computing EffectiveGasTip: %w", err)
	}

	// Convert tip to wei
	effectiveTip.Mul(effectiveTip, gasUsed)

	runningTotalTxFee := new(big.Int).Set(effectiveTip)
	// The "tip" goes to the coinbase address
	balanceChanges.Add(blockHeader.Coinbase, effectiveTip)

	// We want to get state AFTER the tx, since gas fees are processed by the end of the TX
	feeHandlerAddress, err := tr.db.RegistryAddressStartOf(tr.ctx, receipt.BlockNumber, receipt.TransactionIndex+1, feeHandler)
	if err == nil {
		// User is charged baseFee iff community fund exists
		balanceChanges.Add(feeHandlerAddress, baseTxFee)
		runningTotalTxFee.Add(runningTotalTxFee, baseTxFee)
	} else if err != db.ErrContractNotFound {
		return nil, fmt.Errorf("can't get feeHandlerAddress: %w", err)
	}

	if tx.GatewayFeeRecipient() != nil {
		balanceChanges.Add(*tx.GatewayFeeRecipient(), tx.GatewayFee())
		runningTotalTxFee.Add(runningTotalTxFee, tx.GatewayFee())
	}

	// TODO find a better way to do this?
	from, err := tr.cc.Eth.TransactionSender(tr.ctx, tx, receipt.BlockHash, receipt.TransactionIndex)
	if err != nil {
		return nil, fmt.Errorf("can't get transaction sender: %w", err)
	}
	balanceChanges.Add(from, new(big.Int).Neg(runningTotalTxFee))
	return NewFee(balanceChanges.ToMap()), nil
}

func (tr *Tracer) TxTransfers(tx *types.Transaction, receipt *types.Receipt) ([]Operation, error) {
	if receipt.Status == types.ReceiptStatusFailed {
		return nil, nil
	}

	res := debug.TransferTracerResponse{}
	timeout := tr.traceTimeout.String()
	cfg := &tracers.TraceConfig{Tracer: &debug.TransferTracer, Timeout: &timeout}
	err := tr.cc.Debug.TraceTransaction(tr.ctx, &res, tx.Hash(), cfg)
	if err != nil {
		return nil, fmt.Errorf("can't run celo-rpc tx-tracer: %w", err)
	}
	return InternalTransfersToOperations(res.Transfers), nil
}

func (tr *Tracer) TxOpsFromLogs(tx *types.Transaction, receipt *types.Receipt, contractMap map[string]common.Address) ([]Operation, error) {
	if receipt.Status == types.ReceiptStatusFailed {
		return nil, nil
	}

	accountsAddr, ok := contractMap[registry.AccountsContractID.String()]
	if !ok {
		// Accounts not found (not deployed) => no transfers
		return nil, nil
	}
	accounts, err := contracts.NewAccounts(accountsAddr, tr.cc.Eth)
	if err != nil {
		return nil, fmt.Errorf("can't initialize Accounts contract: %w", err)
	}

	lockedGoldAddr, ok := contractMap[registry.LockedGoldContractID.String()]
	if !ok {
		// LockedGold not found (not deployed) => no transfers
		return nil, nil
	}
	lockedGold, err := contracts.NewLockedGold(lockedGoldAddr, tr.cc.Eth)
	if err != nil {
		return nil, fmt.Errorf("can't initialize LockedGold contract: %w", err)
	}

	electionAddr, ok := contractMap[registry.ElectionContractID.String()]
	if !ok {
		// Election not found (not deployed) => no transfers
		return nil, nil
	}
	election, err := contracts.NewElection(electionAddr, tr.cc.Eth)
	if err != nil {
		return nil, fmt.Errorf("can't initialize Election contract: %w", err)
	}

	// We only need governace for slashing and you can't slash if there's no governance contract, so we ignore if not found
	governanceAddr, _ := contractMap[registry.GovernanceContractID.String()]

	logs := utils.RemoveProxyLogs(receipt.Logs)

	transfers := make([]Operation, 0, len(logs))
	for _, eventLog := range logs {
		if eventLog.Address == electionAddr {
			eventName, eventRaw, ok, err := election.TryParseLog(*eventLog)
			if err != nil {
				if strings.HasPrefix(err.Error(), "no event with id") {
					tr.logger.Warn("Ignoring unknown Election event: %w", err)
					continue
				} else {
					return nil, fmt.Errorf("can't parse Election event: %w", err)
				}
			}
			if !ok {
				continue
			}

			// Election:

			switch eventName {
			case "ValidatorGroupVoteCast":
				// vote() [ValidatorGroupVoteCast] => lockNonVoting->lockVotingPending
				event := eventRaw.(*contracts.ElectionValidatorGroupVoteCast)
				transfers = append(transfers, *NewVote(event.Account, event.Group, event.Value))
			case "ValidatorGroupVoteActivated":
				// activate() [ValidatorGroupVoteActivated] => lockVotingPending->lockVotingActive
				event := eventRaw.(*contracts.ElectionValidatorGroupVoteActivated)
				transfers = append(transfers, *NewActiveVotes(event.Account, event.Group, event.Value))
			case "ValidatorGroupPendingVoteRevoked":
				// revokePending() [ValidatorGroupPendingVoteRevoked] => lockVotingPending->lockNonVoting
				event := eventRaw.(*contracts.ElectionValidatorGroupPendingVoteRevoked)
				transfers = append(transfers, *NewRevokePendingVotes(event.Account, event.Group, event.Value))
			case "ValidatorGroupActiveVoteRevoked":
				// revokeActive() [ValidatorGroupActiveVoteRevoked] => lockVotingActive->lockNonVoting
				event := eventRaw.(*contracts.ElectionValidatorGroupActiveVoteRevoked)
				transfers = append(transfers, *NewRevokeActiveVotes(event.Account, event.Group, event.Value))
			}

		} else if eventLog.Address == accountsAddr {
			eventName, eventRaw, ok, err := accounts.TryParseLog(*eventLog)
			if err != nil {
				if strings.HasPrefix(err.Error(), "no event with id") {
					tr.logger.Warn("Ignoring unknown Accounts event: %w", err)
					continue
				} else {
					return nil, fmt.Errorf("can't parse Accounts event: %w", err)
				}
			}
			if !ok {
				continue
			}

			// Accounts:

			switch eventName {
			case "AccountCreated":
				event := eventRaw.(*contracts.AccountsAccountCreated)
				transfers = append(transfers, *NewCreateAccount(event.Account))
			case "VoteSignerAuthorized":
				event := eventRaw.(*contracts.AccountsVoteSignerAuthorized)
				transfers = append(transfers, *NewAuthorizeSigner(event.Account, event.Signer, OpAuthorizeVoteSigner))
			case "ValidatorSignerAuthorized":
				event := eventRaw.(*contracts.AccountsValidatorSignerAuthorized)
				transfers = append(transfers, *NewAuthorizeSigner(event.Account, event.Signer, OpAuthorizeValidatorSigner))
			case "AttestationSignerAuthorized":
				event := eventRaw.(*contracts.AccountsAttestationSignerAuthorized)
				transfers = append(transfers, *NewAuthorizeSigner(event.Account, event.Signer, OpAuthorizeAttestationSigner))
			}

		} else if eventLog.Address == lockedGoldAddr {
			eventName, eventRaw, ok, err := lockedGold.TryParseLog(*eventLog)
			if err != nil {
				if strings.HasPrefix(err.Error(), "no event with id") {
					tr.logger.Warn("Ignoring unknown LockedGold event: %w", err)
					continue
				} else {
					return nil, fmt.Errorf("can't parse LockedGold event: %w", err)
				}
			}
			if !ok {
				continue
			}

			// LockedGold:

			switch eventName {
			case "GoldLocked":
				// lock() [GoldLocked + transfer] => main->lockNonVoting
				event := eventRaw.(*contracts.LockedGoldGoldLocked)
				// Edge case: locking 0 CELO means there isn't a matching transfer;
				// Only store balance-changing (>0) GoldLocked logs.
				if event.Value.Cmp(big.NewInt(0)) > 0 {
					transfers = append(transfers, *NewLockGold(event.Account, lockedGoldAddr, event.Value))
				}
			case "GoldRelocked":
				// relock() [GoldRelocked] => lockPending->lockNonVoting
				event := eventRaw.(*contracts.LockedGoldGoldRelocked)
				transfers = append(transfers, *NewRelockGold(event.Account, event.Value))

			case "GoldUnlocked":
				// unlock() [GoldUnlocked] => lockNonVoting->lockPending
				event := eventRaw.(*contracts.LockedGoldGoldUnlocked)
				transfers = append(transfers, *NewUnlockGold(event.Account, event.Value))

			case "GoldWithdrawn":
				// withdraw() [GoldWithdrawn + transfer] => lockPending->main
				event := eventRaw.(*contracts.LockedGoldGoldWithdrawn)
				// Edge case: withdrawing 0 CELO means there isn't a matching transfer;
				// Only store balance-changing (>0) GoldLocked logs.
				if event.Value.Cmp(big.NewInt(0)) > 0 {
					transfers = append(transfers, *NewWithdrawGold(event.Account, lockedGoldAddr, event.Value))
				}

			case "AccountSlashed":
				// slash() [AccountSlashed + transfer] => account:lockNonVoting -> beneficiary:lockNonVoting + governance:main
				event := eventRaw.(*contracts.LockedGoldAccountSlashed)
				transfers = append(transfers, *NewSlash(event.Slashed, event.Reporter, governanceAddr, lockedGoldAddr, event.Penalty, event.Reward))

			}
		}

	}

	return transfers, nil
}
