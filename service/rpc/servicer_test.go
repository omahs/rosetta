package rpc

import (
	"math/big"
	"testing"

	"github.com/celo-org/rosetta/airgap"
	"github.com/celo-org/rosetta/analyzer"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	. "github.com/onsi/gomega"
)

type sample struct {
	name string
	ops  []*types.Operation
}

func TestParseSendOps(t *testing.T) {
	RegisterTestingT(t)

	validAddress := common.HexToAddress("0x123")
	fakeArgs := []interface{}{"0x123", "120"}
	validSend := []*types.Operation{
		{
			Account: &types.AccountIdentifier{
				Address: validAddress.String(),
			},
			Type: analyzer.OpSend.String(),
			Metadata: map[string]interface{}{
				"method": "StableToken.transfer",
				"args":   fakeArgs,
			},
		},
	}

	expectedTxArgs := airgap.TxArgs{
		From:   validAddress,
		Method: airgap.StableTokenTransfer,
		Args:   fakeArgs,
	}

	// Simple case that should pass
	t.Run("Valid StableToken.transfer", func(t *testing.T) {
		txArgs, rosettaErr := parseSendOps(validSend)
		Ω(rosettaErr).Should(BeNil())
		Ω(*txArgs).Should(Equal(expectedTxArgs))
	})

	// Cases that should fail
	invalidAddr := validSend
	invalidAddr[0].Account.Address = "0x123"
	failSamples := []sample{
		{
			name: "Empty Ops",
			ops:  []*types.Operation{},
		},
		{
			name: "Invalid Address",
			ops:  invalidAddr,
		},
	}
	for _, sample := range failSamples {
		t.Run(sample.name, func(t *testing.T) {
			RegisterTestingT(t)
			txArgs, rosettaErr := parseSendOps(sample.ops)
			Ω(txArgs).Should(BeNil())
			Ω(rosettaErr).ShouldNot(BeNil())
		})
	}
}

func TestParseTransferOps(t *testing.T) {
	RegisterTestingT(t)

	validFrom := common.HexToAddress("0x123")
	validTo := common.HexToAddress("0x456")
	value, _ := new(big.Int).SetString("123456789", 10)

	validTransfer := []*types.Operation{
		{
			Account: &types.AccountIdentifier{
				Address: validFrom.String(),
			},
			Type: analyzer.OpTransfer.String(),
			Amount: &types.Amount{
				Value:    new(big.Int).Neg(value).String(),
				Currency: CeloGold,
			},
		},
		{
			Account: &types.AccountIdentifier{
				Address: validTo.String(),
			},
			Type: analyzer.OpTransfer.String(),
			Amount: &types.Amount{
				Value:    value.String(),
				Currency: CeloGold,
			},
		},
	}

	expectedTxArgs := airgap.TxArgs{
		From:  validFrom,
		To:    &validTo,
		Value: value,
	}

	// Simple case that should pass
	t.Run("Valid CELO transfer", func(t *testing.T) {
		txArgs, rosettaErr := parseTransferOps(validTransfer)
		Ω(rosettaErr).Should(BeNil())
		Ω(*txArgs).Should(Equal(expectedTxArgs))
	})

	// Cases that should fail
	invalidFrom := validTransfer
	invalidFrom[0].Account.Address = "0x123"
	invalidTo := validTransfer
	invalidTo[1].Account.Address = "0x123"

	failSamples := []sample{
		{
			name: "Empty Ops",
			ops:  []*types.Operation{},
		},
		{
			name: "Invalid From",
			ops:  invalidFrom,
		},
		{
			name: "Invalid To",
			ops:  invalidTo,
		},
	}
	for _, sample := range failSamples {
		t.Run(sample.name, func(t *testing.T) {
			RegisterTestingT(t)
			txArgs, rosettaErr := parseTransferOps(sample.ops)
			Ω(txArgs).Should(BeNil())
			Ω(rosettaErr).ShouldNot(BeNil())
		})
	}
}
