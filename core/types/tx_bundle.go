// Copyright 2023 CoreX Team
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

package types

import (
	"bytes"
	"crypto/sha256"
	"math/big"

	"com.corexkey/common"
	"com.corexkey/crypto/kzg4844"
	"com.corexkey/params"
	"com.corexkey/rlp"
	"github.com/holiman/uint256"
)

// BundleTx represents a data-bundle transaction.
type BundleTx struct {
	ChainID       *uint256.Int
	Nonce         uint64
	GasTipCap     *uint256.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap     *uint256.Int // a.k.a. maxFeePerGas
	Gas           uint64
	To            common.Address
	Value         *uint256.Int
	Data          []byte
	AccessProfile AccessProfile
	BundleFeeCap  *uint256.Int // a.k.a. maxFeePerBundleResource
	BundleHashes  []common.Hash

	// Bundle transactions can optionally carry an attachment during signing and RPC transport.
	Attachment *BundleAttachment `rlp:"-"`

	// Signature values
	V *uint256.Int `json:"v" gencodec:"required"`
	R *uint256.Int `json:"r" gencodec:"required"`
	S *uint256.Int `json:"s" gencodec:"required"`
}

// BundleAttachment contains the payload material of a bundle transaction.
type BundleAttachment struct {
	Blobs       []kzg4844.Blob       // Payload segments carried by the bundle
	Commitments []kzg4844.Commitment // Commitments for each payload segment
	Proofs      []kzg4844.Proof      // Proofs for each payload segment
}

// BundleHashes computes commitment hashes for the attachment payloads.
func (sc *BundleAttachment) BundleHashes() []common.Hash {
	hasher := sha256.New()
	h := make([]common.Hash, len(sc.Commitments))
	for i := range sc.Blobs {
		h[i] = kzg4844.CalcBlobHashV1(hasher, &sc.Commitments[i])
	}
	return h
}

// encodedSize computes the RLP size of the attachment elements. This does NOT return the
// encoded size of the BundleAttachment, it's just a helper for tx.Size().
func (sc *BundleAttachment) encodedSize() uint64 {
	var blobs, commitments, proofs uint64
	for i := range sc.Blobs {
		blobs += rlp.BytesSize(sc.Blobs[i][:])
	}
	for i := range sc.Commitments {
		commitments += rlp.BytesSize(sc.Commitments[i][:])
	}
	for i := range sc.Proofs {
		proofs += rlp.BytesSize(sc.Proofs[i][:])
	}
	return rlp.ListSize(blobs) + rlp.ListSize(commitments) + rlp.ListSize(proofs)
}

// bundleTxWithPayloads is used for encoding transactions with an attachment payload.
type bundleTxWithPayloads struct {
	BundleTx    *BundleTx
	Blobs       []kzg4844.Blob
	Commitments []kzg4844.Commitment
	Proofs      []kzg4844.Proof
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *BundleTx) copy() TxData {
	cpy := &BundleTx{
		Nonce: tx.Nonce,
		To:    tx.To,
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are copied below.
		AccessProfile: make(AccessProfile, len(tx.AccessProfile)),
		BundleHashes:  make([]common.Hash, len(tx.BundleHashes)),
		Value:         new(uint256.Int),
		ChainID:       new(uint256.Int),
		GasTipCap:     new(uint256.Int),
		GasFeeCap:     new(uint256.Int),
		BundleFeeCap:  new(uint256.Int),
		V:             new(uint256.Int),
		R:             new(uint256.Int),
		S:             new(uint256.Int),
	}
	copy(cpy.AccessProfile, tx.AccessProfile)
	copy(cpy.BundleHashes, tx.BundleHashes)

	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	if tx.BundleFeeCap != nil {
		cpy.BundleFeeCap.Set(tx.BundleFeeCap)
	}
	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}
	if tx.Attachment != nil {
		cpy.Attachment = &BundleAttachment{
			Blobs:       append([]kzg4844.Blob(nil), tx.Attachment.Blobs...),
			Commitments: append([]kzg4844.Commitment(nil), tx.Attachment.Commitments...),
			Proofs:      append([]kzg4844.Proof(nil), tx.Attachment.Proofs...),
		}
	}
	return cpy
}

// accessors for innerTx.
func (tx *BundleTx) txType() byte                 { return BundleTxType }
func (tx *BundleTx) chainID() *big.Int            { return tx.ChainID.ToBig() }
func (tx *BundleTx) accessProfile() AccessProfile { return tx.AccessProfile }
func (tx *BundleTx) data() []byte                 { return tx.Data }
func (tx *BundleTx) gas() uint64                  { return tx.Gas }
func (tx *BundleTx) gasFeeCap() *big.Int          { return tx.GasFeeCap.ToBig() }
func (tx *BundleTx) gasTipCap() *big.Int          { return tx.GasTipCap.ToBig() }
func (tx *BundleTx) valueFlowFee() *big.Int       { return tx.GasFeeCap.ToBig() }
func (tx *BundleTx) value() *big.Int              { return tx.Value.ToBig() }
func (tx *BundleTx) nonce() uint64                { return tx.Nonce }
func (tx *BundleTx) to() *common.Address          { tmp := tx.To; return &tmp }
func (tx *BundleTx) bundleResource() uint64 {
	return params.BundleTxBundleResourcePerBlob * uint64(len(tx.BundleHashes))
}

func (tx *BundleTx) effectiveValueFlowFee(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap.ToBig())
	}
	tip := dst.Sub(tx.GasFeeCap.ToBig(), baseFee)
	if tip.Cmp(tx.GasTipCap.ToBig()) > 0 {
		tip.Set(tx.GasTipCap.ToBig())
	}
	return tip.Add(tip, baseFee)
}

func (tx *BundleTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V.ToBig(), tx.R.ToBig(), tx.S.ToBig()
}

func (tx *BundleTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID.SetFromBig(chainID)
	tx.V.SetFromBig(v)
	tx.R.SetFromBig(r)
	tx.S.SetFromBig(s)
}

func (tx *BundleTx) withoutAttachment() *BundleTx {
	cpy := *tx
	cpy.Attachment = nil
	return &cpy
}

func (tx *BundleTx) encode(b *bytes.Buffer) error {
	if tx.Attachment == nil {
		return rlp.Encode(b, tx)
	}
	inner := &bundleTxWithPayloads{
		BundleTx:    tx,
		Blobs:       tx.Attachment.Blobs,
		Commitments: tx.Attachment.Commitments,
		Proofs:      tx.Attachment.Proofs,
	}
	return rlp.Encode(b, inner)
}

func (tx *BundleTx) decode(input []byte) error {
	// We support two formats here: a network encoding with an attachment payload, or the
	// canonical encoding without the attachment payload.
	//
	// The two encodings can be distinguished by checking whether the first element of the
	// input list is itself a list.

	outerList, _, err := rlp.SplitList(input)
	if err != nil {
		return err
	}
	firstElemKind, _, _, err := rlp.Split(outerList)
	if err != nil {
		return err
	}

	if firstElemKind != rlp.List {
		return rlp.DecodeBytes(input, tx)
	}
	// It's a transaction with an attachment payload.
	var inner bundleTxWithPayloads
	if err := rlp.DecodeBytes(input, &inner); err != nil {
		return err
	}
	*tx = *inner.BundleTx
	tx.Attachment = &BundleAttachment{
		Blobs:       inner.Blobs,
		Commitments: inner.Commitments,
		Proofs:      inner.Proofs,
	}
	return nil
}
