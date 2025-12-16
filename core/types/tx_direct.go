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
)

// DirectTx is the transaction data of the original direct-transfer profile.
type DirectTx struct {
	Nonce        uint64          // nonce of sender account
	ValueFlowFee *big.Int        // nano per gas
	Gas          uint64          // gas limit
	To           *common.Address `rlp:"nil"` // nil means contract creation
	Value        *big.Int        // nano amount
	Data         []byte          // contract invocation input data
	V, R, S      *big.Int        // signature values
}

// NewTransaction creates an unsigned direct transaction.
// Deprecated: use NewTx instead.
func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return NewTx(&DirectTx{
		Nonce:        nonce,
		To:           &to,
		Value:        amount,
		Gas:          gasLimit,
		ValueFlowFee: gasPrice,
		Data:         data,
	})
}

// NewContractCreation creates an unsigned direct contract-creation transaction.
// Deprecated: use NewTx instead.
func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return NewTx(&DirectTx{
		Nonce:        nonce,
		Value:        amount,
		Gas:          gasLimit,
		ValueFlowFee: gasPrice,
		Data:         data,
	})
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *DirectTx) copy() TxData {
	cpy := &DirectTx{
		Nonce: tx.Nonce,
		To:    copyAddressPtr(tx.To),
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are initialized below.
		Value:        new(big.Int),
		ValueFlowFee: new(big.Int),
		V:            new(big.Int),
		R:            new(big.Int),
		S:            new(big.Int),
	}
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
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
func (tx *DirectTx) txType() byte           { return DirectTxType }
func (tx *DirectTx) chainID() *big.Int      { return deriveChainId(tx.V) }
func (tx *DirectTx) accessProfile() AccessProfile { return nil }
func (tx *DirectTx) data() []byte           { return tx.Data }
func (tx *DirectTx) gas() uint64            { return tx.Gas }
func (tx *DirectTx) valueFlowFee() *big.Int { return tx.ValueFlowFee }
func (tx *DirectTx) gasTipCap() *big.Int    { return tx.ValueFlowFee }
func (tx *DirectTx) gasFeeCap() *big.Int    { return tx.ValueFlowFee }
func (tx *DirectTx) value() *big.Int        { return tx.Value }
func (tx *DirectTx) nonce() uint64          { return tx.Nonce }
func (tx *DirectTx) to() *common.Address    { return tx.To }

func (tx *DirectTx) effectiveValueFlowFee(dst *big.Int, baseFee *big.Int) *big.Int {
	return dst.Set(tx.ValueFlowFee)
}

func (tx *DirectTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *DirectTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.V, tx.R, tx.S = v, r, s
}

func (tx *DirectTx) encode(*bytes.Buffer) error {
	panic("encode called on DirectTx")
}

func (tx *DirectTx) decode([]byte) error {
	panic("decode called on DirectTx)")
}
