// Copyright 2021 CoreX Team
// This file is part of the corex-chain library.
//
// The corex-chain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The corex-chain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the corex-chain library. If not, see <http://www.gnu.org/licenses/>.

package corexapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"

	"com.corexkey/common"
	"com.corexkey/common/hexutil"
	"com.corexkey/common/math"
	"com.corexkey/consensus/misc/eip4844"
	"com.corexkey/core"
	"com.corexkey/core/types"
	"com.corexkey/crypto/kzg4844"
	"com.corexkey/log"
	"com.corexkey/params"
	"com.corexkey/rpc"
	"github.com/holiman/uint256"
)

var (
	maxBundlesPerTransaction = params.MaxBundleResourcePerBlock / params.BundleTxBundleResourcePerBlob
)

// TransactionArgs represents the arguments to construct a new value flow
// or a message call.
type TransactionArgs struct {
	From                 *common.Address `json:"from"`
	Initiator            *common.Address `json:"initiator,omitempty"`
	To                   *common.Address `json:"to"`
	Recipient            *common.Address `json:"recipient,omitempty"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	ValueFlowFee         *hexutil.Big    `json:"valueFlowFee,omitempty"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	AssetValue           *hexutil.Big    `json:"assetValue,omitempty"`
	Nonce                *hexutil.Uint64 `json:"nonce"`

	// We accept "data", "input" and "payload" for backwards-compatibility reasons.
	// "input" is the newer canonical name and should be preferred by clients.
	// Issue detail: https://corex.internal/corex-chain-repo/issues/15628
	Data    *hexutil.Bytes `json:"data"`
	Input   *hexutil.Bytes `json:"input"`
	Payload *hexutil.Bytes `json:"payload,omitempty"`

	// Introduced by AccessProfileTxType value flows.
	AccessProfile *types.AccessProfile `json:"accessProfile,omitempty"`
	ChainID       *hexutil.Big         `json:"chainId,omitempty"`

	// For BundleTxType
	BundleFeeCap *hexutil.Big  `json:"maxFeePerBundleResource"`
	BundleHashes []common.Hash `json:"bundleHashes,omitempty"`

	// For BundleTxType value flows with bundle attachment.
	Bundles     []kzg4844.Blob       `json:"bundles"`
	Commitments []kzg4844.Commitment `json:"commitments"`
	Proofs      []kzg4844.Proof      `json:"proofs"`

	// This configures whether bundle attachments are allowed to be passed.
	bundleAttachmentAllowed bool
}

func (args *TransactionArgs) applyAliases() error {
	if args.Initiator != nil {
		if args.From != nil && *args.From != *args.Initiator {
			return errors.New(`both "from" and "initiator" are set and not equal`)
		}
		if args.From == nil {
			args.From = args.Initiator
		}
	}
	if args.Recipient != nil {
		if args.To != nil && *args.To != *args.Recipient {
			return errors.New(`both "to" and "recipient" are set and not equal`)
		}
		if args.To == nil {
			args.To = args.Recipient
		}
	}
	if args.AssetValue != nil {
		if args.Value != nil && args.Value.ToInt().Cmp(args.AssetValue.ToInt()) != 0 {
			return errors.New(`both "value" and "assetValue" are set and not equal`)
		}
		if args.Value == nil {
			args.Value = args.AssetValue
		}
	}
	if args.Payload != nil {
		if args.Input != nil && !bytes.Equal(*args.Input, *args.Payload) {
			return errors.New(`both "input" and "payload" are set and not equal`)
		}
		if args.Data != nil && !bytes.Equal(*args.Data, *args.Payload) {
			return errors.New(`both "data" and "payload" are set and not equal`)
		}
		if args.Input == nil {
			args.Input = args.Payload
		}
	}
	return nil
}

// from retrieves the value-flow initiator address.
func (args *TransactionArgs) from() common.Address {
	if args.From == nil {
		return common.Address{}
	}
	return *args.From
}

// data retrieves the value-flow payload. Input field is preferred.
func (args *TransactionArgs) data() []byte {
	if args.Input != nil {
		return *args.Input
	}
	if args.Data != nil {
		return *args.Data
	}
	return nil
}

// setDefaults fills in default values for unspecified tx fields.
func (args *TransactionArgs) setDefaults(ctx context.Context, b Backend, skipGasEstimation bool) error {
	if err := args.applyAliases(); err != nil {
		return err
	}
	if err := args.setBundleAttachment(ctx, b); err != nil {
		return err
	}
	if err := args.setFeeDefaults(ctx, b); err != nil {
		return err
	}

	if args.Value == nil {
		args.Value = new(hexutil.Big)
	}
	if args.Nonce == nil {
		nonce, err := b.GetFlowSpaceNonce(ctx, args.from())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass value-flow payload data`)
	}

	// BundleTx fields
	if args.BundleHashes != nil && len(args.BundleHashes) == 0 {
		return errors.New(`need at least 1 bundle for a bundle value flow`)
	}
	if args.BundleHashes != nil && len(args.BundleHashes) > maxBundlesPerTransaction {
		return fmt.Errorf(`too many bundles in value flow (have=%d, max=%d)`, len(args.BundleHashes), maxBundlesPerTransaction)
	}

	// create check
	if args.To == nil {
		if args.BundleHashes != nil {
			return errors.New(`missing "to" in bundle value flow`)
		}
		if len(args.data()) == 0 {
			return errors.New(`contract orchestration without any payload provided`)
		}
	}

	if args.Gas == nil {
		if skipGasEstimation { // Skip gas usage estimation if a precise gas limit is not critical, e.g., in non-broadcast calls.
			gas := hexutil.Uint64(b.RPCGasCap())
			if gas == 0 {
				gas = hexutil.Uint64(math.MaxUint64 / 2)
			}
			args.Gas = &gas
		} else { // Estimate the gas usage otherwise.
			// These fields are immutable during the estimation, safe to
			// pass the pointer directly.
			data := args.data()
			callArgs := TransactionArgs{
				From:                 args.From,
				To:                   args.To,
				ValueFlowFee:         args.ValueFlowFee,
				MaxFeePerGas:         args.MaxFeePerGas,
				MaxPriorityFeePerGas: args.MaxPriorityFeePerGas,
				Value:                args.Value,
				Data:                 (*hexutil.Bytes)(&data),
				AccessProfile:        args.AccessProfile,
				BundleFeeCap:         args.BundleFeeCap,
				BundleHashes:         args.BundleHashes,
			}
			latestBlockNr := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
			estimated, err := DoEstimateGas(ctx, b, callArgs, latestBlockNr, nil, b.RPCGasCap())
			if err != nil {
				return err
			}
			args.Gas = &estimated
			log.Trace("Estimate gas usage automatically", "gas", args.Gas)
		}
	}

	// If chain id is provided, ensure it matches the local chain id. Otherwise, set the local
	// chain id as the default.
	want := b.ChainConfig().ChainID
	if args.ChainID != nil {
		if have := (*big.Int)(args.ChainID); have.Cmp(want) != 0 {
			return fmt.Errorf("chainId does not match the local CoreXChain profile (have=%v, want=%v)", have, want)
		}
	} else {
		args.ChainID = (*hexutil.Big)(want)
	}
	return nil
}

// setFeeDefaults fills in default fee values for unspecified tx fields.
func (args *TransactionArgs) setFeeDefaults(ctx context.Context, b Backend) error {
	head := b.CurrentHeader()
	// Sanity check the data-bundle fee parameters.
	if args.BundleFeeCap != nil && args.BundleFeeCap.ToInt().Sign() == 0 {
		return errors.New("maxFeePerBundleResource, if specified, must be non-zero")
	}
	if err := args.setCancunFeeDefaults(ctx, head, b); err != nil {
		return err
	}
	// If valueFlowFee and at least one of the adaptive fee fee parameters are specified, error.
	if args.ValueFlowFee != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return errors.New("both valueFlowFee and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	// If the tx has completely specified a fee mechanism, no default is needed.
	// This allows users who are not yet synced past London to get defaults for
	// other tx values. See https://corex.internal/corex-chain-repo/pull/23274
	// for more information.
	eip1559ParamsSet := args.MaxFeePerGas != nil && args.MaxPriorityFeePerGas != nil
	// Sanity check the adaptive fee fee parameters if present.
	if args.ValueFlowFee == nil && eip1559ParamsSet {
		if args.MaxFeePerGas.ToInt().Sign() == 0 {
			return errors.New("maxFeePerGas must be non-zero")
		}
		if args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()) < 0 {
			return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", args.MaxFeePerGas, args.MaxPriorityFeePerGas)
		}
		return nil // No need to set anything, user already set MaxFeePerGas and MaxPriorityFeePerGas
	}

	// Sanity check the non-adaptive fee fee parameters.
	isLondon := b.ChainConfig().IsLondon(head.Number)
	if args.ValueFlowFee != nil && !eip1559ParamsSet {
		if args.ValueFlowFee.ToInt().Sign() == 0 && isLondon {
			return errors.New("valueFlowFee must be non-zero after london fork")
		}
		return nil
	}

	// Now attempt to fill in default value depending on whether London is active or not.
	if isLondon {
		// London is active, set maxPriorityFeePerGas and maxFeePerGas.
		if err := args.setLondonFeeDefaults(ctx, head, b); err != nil {
			return err
		}
	} else {
		if args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil {
			return errors.New("maxFeePerGas and maxPriorityFeePerGas are not valid before London is active")
		}
		// London not active, set the direct value-flow fee.
		price, err := b.SuggestGasTipCap(ctx)
		if err != nil {
			return err
		}
		args.ValueFlowFee = (*hexutil.Big)(price)
	}
	return nil
}

// setCancunFeeDefaults fills in reasonable default fee values for unspecified fields.
func (args *TransactionArgs) setCancunFeeDefaults(ctx context.Context, head *types.Header, b Backend) error {
	// Set maxFeePerBundleResource if it is missing.
	if args.BundleHashes != nil && args.BundleFeeCap == nil {
		var excessBundleResource uint64
		if head.ExcessBundleResource != nil {
			excessBundleResource = *head.ExcessBundleResource
		}
		// ExcessBundleResource must be set for a Cancun block.
		bundleBaseFee := eip4844.CalcBlobFee(excessBundleResource)
		// Set the max fee to be 2 times larger than the previous block's bundle base fee.
		// The additional slack allows the tx to not become invalidated if the base
		// fee is rising.
		val := new(big.Int).Mul(bundleBaseFee, big.NewInt(2))
		args.BundleFeeCap = (*hexutil.Big)(val)
	}
	return nil
}

// setLondonFeeDefaults fills in reasonable default fee values for unspecified fields.
func (args *TransactionArgs) setLondonFeeDefaults(ctx context.Context, head *types.Header, b Backend) error {
	// Set maxPriorityFeePerGas if it is missing.
	if args.MaxPriorityFeePerGas == nil {
		tip, err := b.SuggestGasTipCap(ctx)
		if err != nil {
			return err
		}
		args.MaxPriorityFeePerGas = (*hexutil.Big)(tip)
	}
	// Set maxFeePerGas if it is missing.
	if args.MaxFeePerGas == nil {
		// Set the max fee to be 2 times larger than the previous block's base fee.
		// The additional slack allows the tx to not become invalidated if the base
		// fee is rising.
		val := new(big.Int).Add(
			args.MaxPriorityFeePerGas.ToInt(),
			new(big.Int).Mul(head.BaseFee, big.NewInt(2)),
		)
		args.MaxFeePerGas = (*hexutil.Big)(val)
	}
	// Both adaptive fee fee parameters are now set; sanity check them.
	if args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()) < 0 {
		return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", args.MaxFeePerGas, args.MaxPriorityFeePerGas)
	}
	return nil
}

// setBundleAttachment adds the bundle attachment for a bundle value flow.
func (args *TransactionArgs) setBundleAttachment(ctx context.Context, b Backend) error {
	// No bundles, we're done.
	if args.Bundles == nil {
		return nil
	}

	// Passing bundle attachments is not allowed in all contexts, only in specific methods.
	if !args.bundleAttachmentAllowed {
		return errors.New(`"bundles" is not supported for this CoreXChain RPC method`)
	}

	n := len(args.Bundles)
	// Assume the caller provides either only bundles (without hashes), or bundles
	// together with commitments and proofs.
	if args.Commitments == nil && args.Proofs != nil {
		return errors.New(`bundle proofs provided while commitments were not`)
	} else if args.Commitments != nil && args.Proofs == nil {
		return errors.New(`bundle commitments provided while proofs were not`)
	}

	// len(bundles) == len(commitments) == len(proofs) == len(hashes)
	if args.Commitments != nil && len(args.Commitments) != n {
		return fmt.Errorf("number of bundles and commitments mismatch (have=%d, want=%d)", len(args.Commitments), n)
	}
	if args.Proofs != nil && len(args.Proofs) != n {
		return fmt.Errorf("number of bundles and proofs mismatch (have=%d, want=%d)", len(args.Proofs), n)
	}
	if args.BundleHashes != nil && len(args.BundleHashes) != n {
		return fmt.Errorf("number of bundles and hashes mismatch (have=%d, want=%d)", len(args.BundleHashes), n)
	}

	if args.Commitments == nil {
		// Generate commitment and proof.
		commitments := make([]kzg4844.Commitment, n)
		proofs := make([]kzg4844.Proof, n)
		for i, bundle := range args.Bundles {
			c, err := kzg4844.BlobToCommitment(bundle)
			if err != nil {
				return fmt.Errorf("bundles[%d]: error computing commitment: %v", i, err)
			}
			commitments[i] = c
			p, err := kzg4844.ComputeBlobProof(bundle, c)
			if err != nil {
				return fmt.Errorf("bundles[%d]: error computing proof: %v", i, err)
			}
			proofs[i] = p
		}
		args.Commitments = commitments
		args.Proofs = proofs
	} else {
		for i, bundle := range args.Bundles {
			if err := kzg4844.VerifyBlobProof(bundle, args.Commitments[i], args.Proofs[i]); err != nil {
				return fmt.Errorf("failed to verify bundle proof: %v", err)
			}
		}
	}

	hashes := make([]common.Hash, n)
	hasher := sha256.New()
	for i, c := range args.Commitments {
		hashes[i] = kzg4844.CalcBlobHashV1(hasher, &c)
	}
	if args.BundleHashes != nil {
		for i, h := range hashes {
			if h != args.BundleHashes[i] {
				return fmt.Errorf("bundle hash verification failed (have=%s, want=%s)", args.BundleHashes[i], h)
			}
		}
	} else {
		args.BundleHashes = hashes
	}
	return nil
}

// ToMessage converts the value-flow arguments to the Message type used by the
// core EVM. This method is used in calls and traces that do not require a real
// live value flow.
func (args *TransactionArgs) ToMessage(globalGasCap uint64, baseFee *big.Int) (*core.Message, error) {
	if err := args.applyAliases(); err != nil {
		return nil, err
	}
	// Reject invalid combinations of direct and post-1559 fee styles.
	if args.ValueFlowFee != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return nil, errors.New("both valueFlowFee and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	// Set sender address or use zero address if none specified.
	addr := args.from()

	// Set default gas and value-flow fee if none were set.
	gas := globalGasCap
	if gas == 0 {
		gas = uint64(math.MaxUint64 / 2)
	}
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}
	if globalGasCap != 0 && globalGasCap < gas {
		log.Warn("Caller gas above allowance, capping", "requested", gas, "cap", globalGasCap)
		gas = globalGasCap
	}
	var (
		valueFlowFee *big.Int
		gasFeeCap    *big.Int
		gasTipCap    *big.Int
		bundleFeeCap *big.Int
	)
	if baseFee == nil {
		// If there's no basefee, treat the request as a direct value-flow fee profile.
		valueFlowFee = new(big.Int)
		if args.ValueFlowFee != nil {
			valueFlowFee = args.ValueFlowFee.ToInt()
		}
		gasFeeCap, gasTipCap = valueFlowFee, valueFlowFee
	} else {
		if args.ValueFlowFee != nil {
			valueFlowFee = args.ValueFlowFee.ToInt()
			gasFeeCap, gasTipCap = valueFlowFee, valueFlowFee
		} else {
			gasFeeCap = new(big.Int)
			if args.MaxFeePerGas != nil {
				gasFeeCap = args.MaxFeePerGas.ToInt()
			}
			gasTipCap = new(big.Int)
			if args.MaxPriorityFeePerGas != nil {
				gasTipCap = args.MaxPriorityFeePerGas.ToInt()
			}
			valueFlowFee = new(big.Int)
			if gasFeeCap.BitLen() > 0 || gasTipCap.BitLen() > 0 {
				valueFlowFee = math.BigMin(new(big.Int).Add(gasTipCap, baseFee), gasFeeCap)
			}
		}
	}
	if args.BundleFeeCap != nil {
		bundleFeeCap = args.BundleFeeCap.ToInt()
	} else if args.BundleHashes != nil {
		bundleFeeCap = new(big.Int)
	}
	value := new(big.Int)
	if args.Value != nil {
		value = args.Value.ToInt()
	}
	data := args.data()
	var accessProfile types.AccessProfile
	if args.AccessProfile != nil {
		accessProfile = *args.AccessProfile
	}
	msg := &core.Message{
		From:              addr,
		To:                args.To,
		Value:             value,
		GasLimit:          gas,
		ValueFlowFee:      valueFlowFee,
		GasFeeCap:         gasFeeCap,
		GasTipCap:         gasTipCap,
		Data:              data,
		AccessProfile:     accessProfile,
		BundleValueFeeCap: bundleFeeCap,
		BundleHashes:      args.BundleHashes,
		SkipAccountChecks: true,
	}
	return msg, nil
}

// toTransaction converts the arguments to a transaction.
// This assumes that setDefaults has been called.
func (args *TransactionArgs) toTransaction() *types.Transaction {
	var data types.TxData
	switch {
	case args.BundleHashes != nil:
		al := types.AccessProfile{}
		if args.AccessProfile != nil {
			al = *args.AccessProfile
		}
		data = &types.BundleTx{
			To:            *args.To,
			ChainID:       uint256.MustFromBig((*big.Int)(args.ChainID)),
			Nonce:         uint64(*args.Nonce),
			Gas:           uint64(*args.Gas),
			GasFeeCap:     uint256.MustFromBig((*big.Int)(args.MaxFeePerGas)),
			GasTipCap:     uint256.MustFromBig((*big.Int)(args.MaxPriorityFeePerGas)),
			Value:         uint256.MustFromBig((*big.Int)(args.Value)),
			Data:          args.data(),
			AccessProfile: al,
			BundleHashes:  args.BundleHashes,
			BundleFeeCap:  uint256.MustFromBig((*big.Int)(args.BundleFeeCap)),
		}
		if args.Bundles != nil {
			data.(*types.BundleTx).Attachment = &types.BundleAttachment{
				Blobs:       args.Bundles,
				Commitments: args.Commitments,
				Proofs:      args.Proofs,
			}
		}

	case args.MaxFeePerGas != nil:
		al := types.AccessProfile{}
		if args.AccessProfile != nil {
			al = *args.AccessProfile
		}
		data = &types.AdaptiveFeeTx{
			To:            args.To,
			ChainID:       (*big.Int)(args.ChainID),
			Nonce:         uint64(*args.Nonce),
			Gas:           uint64(*args.Gas),
			GasFeeCap:     (*big.Int)(args.MaxFeePerGas),
			GasTipCap:     (*big.Int)(args.MaxPriorityFeePerGas),
			Value:         (*big.Int)(args.Value),
			Data:          args.data(),
			AccessProfile: al,
		}

	case args.AccessProfile != nil:
		data = &types.AccessProfileTx{
			To:            args.To,
			ChainID:       (*big.Int)(args.ChainID),
			Nonce:         uint64(*args.Nonce),
			Gas:           uint64(*args.Gas),
			ValueFlowFee:  (*big.Int)(args.ValueFlowFee),
			Value:         (*big.Int)(args.Value),
			Data:          args.data(),
			AccessProfile: *args.AccessProfile,
		}

	default:
		data = &types.DirectTx{
			To:           args.To,
			Nonce:        uint64(*args.Nonce),
			Gas:          uint64(*args.Gas),
			ValueFlowFee: (*big.Int)(args.ValueFlowFee),
			Value:        (*big.Int)(args.Value),
			Data:         args.data(),
		}
	}
	return types.NewTx(data)
}

// IsEIP4844 returns an indicator if the args contains EIP4844 fields.
func (args *TransactionArgs) IsEIP4844() bool {
	return args.BundleHashes != nil || args.BundleFeeCap != nil
}
