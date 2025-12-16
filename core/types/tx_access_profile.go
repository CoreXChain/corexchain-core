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

package types

import (
	"bytes"
	"math/big"

	"com.corexkey/common"
	"com.corexkey/rlp"
)

//go:generate go run github.com/fjl/gencodec -type AccessProfileEntry -out gen_access_profile_entry.go

// AccessProfile describes the explicit state access profile carried by a value flow.
type AccessProfile []AccessProfileEntry

// AccessProfileEntry is a single access-profile entry.
type AccessProfileEntry struct {
	Address     common.Address `json:"address"     gencodec:"required"`
	StorageKeys []common.Hash  `json:"storageKeys" gencodec:"required"`
}

// StorageKeys returns the total number of storage keys in the profile.
func (al AccessProfile) StorageKeys() int {
	sum := 0
	for _, tuple := range al {
		sum += len(tuple.StorageKeys)
	}
	return sum
}

// AccessProfileTx is the data of an access-profile transaction.
type AccessProfileTx struct {
	ChainID       *big.Int        // destination chain ID
	Nonce         uint64          // nonce of sender account
	ValueFlowFee  *big.Int        // nano per gas
	Gas           uint64          // gas limit
	To            *common.Address `rlp:"nil"` // nil means contract creation
	Value         *big.Int        // nano amount
	Data          []byte          // contract invocation input data
	AccessProfile AccessProfile   // declared access profile
	V, R, S       *big.Int        // signature values
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *AccessProfileTx) copy() TxData {
	cpy := &AccessProfileTx{
		Nonce: tx.Nonce,
		To:    copyAddressPtr(tx.To),
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are copied below.
		AccessProfile: make(AccessProfile, len(tx.AccessProfile)),
		Value:         new(big.Int),
		ChainID:       new(big.Int),
		ValueFlowFee:  new(big.Int),
		V:             new(big.Int),
		R:             new(big.Int),
		S:             new(big.Int),
	}
	copy(cpy.AccessProfile, tx.AccessProfile)
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.ValueFlowFee != nil {
		cpy.ValueFlowFee.Set(tx.ValueFlowFee)
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
	return cpy
}

// accessors for innerTx.
func (tx *AccessProfileTx) txType() byte                 { return AccessProfileTxType }
func (tx *AccessProfileTx) chainID() *big.Int            { return tx.ChainID }
func (tx *AccessProfileTx) accessProfile() AccessProfile { return tx.AccessProfile }
func (tx *AccessProfileTx) data() []byte                 { return tx.Data }
func (tx *AccessProfileTx) gas() uint64                  { return tx.Gas }
func (tx *AccessProfileTx) valueFlowFee() *big.Int       { return tx.ValueFlowFee }
func (tx *AccessProfileTx) gasTipCap() *big.Int          { return tx.ValueFlowFee }
func (tx *AccessProfileTx) gasFeeCap() *big.Int          { return tx.ValueFlowFee }
func (tx *AccessProfileTx) value() *big.Int              { return tx.Value }
func (tx *AccessProfileTx) nonce() uint64                { return tx.Nonce }
func (tx *AccessProfileTx) to() *common.Address          { return tx.To }

func (tx *AccessProfileTx) effectiveValueFlowFee(dst *big.Int, baseFee *big.Int) *big.Int {
	return dst.Set(tx.ValueFlowFee)
}

func (tx *AccessProfileTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *AccessProfileTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}

func (tx *AccessProfileTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *AccessProfileTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}
