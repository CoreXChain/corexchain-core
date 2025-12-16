// Copyright 2014 CoreX Team
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
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync/atomic"
	"time"

	"com.corexkey/common"
	"com.corexkey/common/math"
	"com.corexkey/crypto"
	"com.corexkey/rlp"
)

var (
	ErrInvalidSig           = errors.New("invalid transaction v, r, s values")
	ErrUnexpectedProtection = errors.New("transaction type does not supported chain-protected protected signatures")
	ErrInvalidTxType        = errors.New("transaction type not valid in this context")
	ErrTxTypeNotSupported   = errors.New("transaction type not supported")
	ErrGasFeeCapTooLow      = errors.New("fee cap less than base fee")
	errShortTypedTx         = errors.New("typed transaction too short")
	errInvalidYParity       = errors.New("'yParity' field must be 0 or 1")
	errVYParityMismatch     = errors.New("'v' and 'yParity' fields do not match")
	errVYParityMissing      = errors.New("missing 'yParity' or 'v' field in transaction")
)

// Transaction types.
const (
	DirectTxType        = 0x00
	AccessProfileTxType = 0x01
	AdaptiveFeeTxType   = 0x02
	BundleTxType        = 0x03
)

// Transaction is a CoreXChain transaction.
type Transaction struct {
	inner TxData    // Consensus contents of a transaction
	time  time.Time // Time first seen locally (spam avoidance)

	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

// NewTx creates a new transaction.
func NewTx(inner TxData) *Transaction {
	tx := new(Transaction)
	tx.setDecoded(inner.copy(), 0)
	return tx
}

// TxData is the underlying data of a transaction.
//
// This is implemented by AdaptiveFeeTx, DirectTx, AccessProfileTx and BundleTx.
type TxData interface {
	txType() byte // returns the type ID
	copy() TxData // creates a deep copy and initializes all fields

	chainID() *big.Int
	accessProfile() AccessProfile
	data() []byte
	gas() uint64
	valueFlowFee() *big.Int
	gasTipCap() *big.Int
	gasFeeCap() *big.Int
	value() *big.Int
	nonce() uint64
	to() *common.Address

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)

	// effectiveValueFlowFee computes the value-flow fee paid by the transaction, given
	// the inclusion block baseFee.
	//
	// Unlike other TxData methods, the returned *big.Int should be an independent
	// copy of the computed value, i.e. callers are allowed to mutate the result.
	// Method implementations can use 'dst' to store the result.
	effectiveValueFlowFee(dst *big.Int, baseFee *big.Int) *big.Int

	encode(*bytes.Buffer) error
	decode([]byte) error
}

// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	if tx.Type() == DirectTxType {
		return rlp.Encode(w, tx.inner)
	}
	// It's a typed transaction envelope.
	buf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(buf)
	buf.Reset()
	if err := tx.encodeTyped(buf); err != nil {
		return err
	}
	return rlp.Encode(w, buf.Bytes())
}

// encodeTyped writes the canonical encoding of a typed transaction to w.
func (tx *Transaction) encodeTyped(w *bytes.Buffer) error {
	w.WriteByte(tx.Type())
	return tx.inner.encode(w)
}

// MarshalBinary returns the canonical encoding of the transaction.
// For direct transactions, it returns the RLP encoding. For typed transactions,
// it returns the type byte and payload.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	if tx.Type() == DirectTxType {
		return rlp.EncodeToBytes(tx.inner)
	}
	var buf bytes.Buffer
	err := tx.encodeTyped(&buf)
	return buf.Bytes(), err
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	kind, size, err := s.Kind()
	switch {
	case err != nil:
		return err
	case kind == rlp.List:
		// It's a direct transaction.
		var inner DirectTx
		err := s.Decode(&inner)
		if err == nil {
			tx.setDecoded(&inner, rlp.ListSize(size))
		}
		return err
	case kind == rlp.Byte:
		return errShortTypedTx
	default:
		// It's a typed transaction envelope.
		// First read the tx payload bytes into a temporary buffer.
		b, buf, err := getPooledBuffer(size)
		if err != nil {
			return err
		}
		defer encodeBufferPool.Put(buf)
		if err := s.ReadBytes(b); err != nil {
			return err
		}
		// Now decode the inner transaction.
		inner, err := tx.decodeTyped(b)
		if err == nil {
			tx.setDecoded(inner, size)
		}
		return err
	}
}

// UnmarshalBinary decodes the canonical encoding of transactions.
// It supports direct RLP transactions and typed transaction envelopes.
func (tx *Transaction) UnmarshalBinary(b []byte) error {
	if len(b) > 0 && b[0] > 0x7f {
		// It's a direct transaction.
		var data DirectTx
		err := rlp.DecodeBytes(b, &data)
		if err != nil {
			return err
		}
		tx.setDecoded(&data, uint64(len(b)))
		return nil
	}
	// It's a typed transaction envelope.
	inner, err := tx.decodeTyped(b)
	if err != nil {
		return err
	}
	tx.setDecoded(inner, uint64(len(b)))
	return nil
}

// decodeTyped decodes a typed transaction from the canonical format.
func (tx *Transaction) decodeTyped(b []byte) (TxData, error) {
	if len(b) <= 1 {
		return nil, errShortTypedTx
	}
	var inner TxData
	switch b[0] {
	case AccessProfileTxType:
		inner = new(AccessProfileTx)
	case AdaptiveFeeTxType:
		inner = new(AdaptiveFeeTx)
	case BundleTxType:
		inner = new(BundleTx)
	default:
		return nil, ErrTxTypeNotSupported
	}
	err := inner.decode(b[1:])
	return inner, err
}

// setDecoded sets the inner transaction and size after decoding.
func (tx *Transaction) setDecoded(inner TxData, size uint64) {
	tx.inner = inner
	tx.time = time.Now()
	if size > 0 {
		tx.size.Store(size)
	}
}

func sanityCheckSignature(v *big.Int, r *big.Int, s *big.Int, maybeProtected bool) error {
	if isProtectedV(v) && !maybeProtected {
		return ErrUnexpectedProtection
	}

	var plainV byte
	if isProtectedV(v) {
		chainID := deriveChainId(v).Uint64()
		plainV = byte(v.Uint64() - 35 - 2*chainID)
	} else if maybeProtected {
		// Only chain-protected signatures can be optionally protected. Since
		// we determined this v value is not protected, it must be a
		// raw 27 or 28.
		plainV = byte(v.Uint64() - 27)
	} else {
		// If the signature is not optionally protected, we assume it
		// must already be equal to the recovery id.
		plainV = byte(v.Uint64())
	}
	if !crypto.ValidateSignatureValues(plainV, r, s, false) {
		return ErrInvalidSig
	}

	return nil
}

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28 && v != 1 && v != 0
	}
	// anything not 27 or 28 is considered protected
	return true
}

// Protected says whether the transaction is replay-protected.
func (tx *Transaction) Protected() bool {
	switch tx := tx.inner.(type) {
	case *DirectTx:
		return tx.V != nil && isProtectedV(tx.V)
	default:
		return true
	}
}

// Type returns the transaction type.
func (tx *Transaction) Type() uint8 {
	return tx.inner.txType()
}

// ChainId returns the EIP155 chain ID of the transaction. The return value will always be
// non-nil. For direct transactions which are not replay-protected, the return value is
// zero.
func (tx *Transaction) ChainId() *big.Int {
	return tx.inner.chainID()
}

// Data returns the input data of the transaction.
func (tx *Transaction) Data() []byte { return tx.inner.data() }

// AccessProfile returns the access list of the transaction.
func (tx *Transaction) AccessProfile() AccessProfile { return tx.inner.accessProfile() }

// Gas returns the gas limit of the transaction.
func (tx *Transaction) Gas() uint64 { return tx.inner.gas() }

// ValueFlowFee returns the direct value-flow fee of the transaction.
func (tx *Transaction) ValueFlowFee() *big.Int { return new(big.Int).Set(tx.inner.valueFlowFee()) }

// GasTipCap returns the gasTipCap per gas of the transaction.
func (tx *Transaction) GasTipCap() *big.Int { return new(big.Int).Set(tx.inner.gasTipCap()) }

// GasFeeCap returns the fee cap per gas of the transaction.
func (tx *Transaction) GasFeeCap() *big.Int { return new(big.Int).Set(tx.inner.gasFeeCap()) }

// Value returns the corex amount of the transaction.
func (tx *Transaction) Value() *big.Int { return new(big.Int).Set(tx.inner.value()) }

// Nonce returns the sender account nonce of the transaction.
func (tx *Transaction) Nonce() uint64 { return tx.inner.nonce() }

// To returns the recipient address of the transaction.
// For contract-creation transactions, To returns nil.
func (tx *Transaction) To() *common.Address {
	return copyAddressPtr(tx.inner.to())
}

// Cost returns (gas * valueFlowFee) + (bundleResource * bundleValueFee) + value.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.ValueFlowFee(), new(big.Int).SetUint64(tx.Gas()))
	if tx.Type() == BundleTxType {
		total.Add(total, new(big.Int).Mul(tx.BundleValueFeeCap(), new(big.Int).SetUint64(tx.BundleResource())))
	}
	total.Add(total, tx.Value())
	return total
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
// The return values may be nil or zero, if the transaction is unsigned.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.inner.rawSignatureValues()
}

// GasFeeCapCmp compares the fee cap of two transactions.
func (tx *Transaction) GasFeeCapCmp(other *Transaction) int {
	return tx.inner.gasFeeCap().Cmp(other.inner.gasFeeCap())
}

// GasFeeCapIntCmp compares the fee cap of the transaction against the given fee cap.
func (tx *Transaction) GasFeeCapIntCmp(other *big.Int) int {
	return tx.inner.gasFeeCap().Cmp(other)
}

// GasTipCapCmp compares the gasTipCap of two transactions.
func (tx *Transaction) GasTipCapCmp(other *Transaction) int {
	return tx.inner.gasTipCap().Cmp(other.inner.gasTipCap())
}

// GasTipCapIntCmp compares the gasTipCap of the transaction against the given gasTipCap.
func (tx *Transaction) GasTipCapIntCmp(other *big.Int) int {
	return tx.inner.gasTipCap().Cmp(other)
}

// EffectiveGasTip returns the effective miner gasTipCap for the given base fee.
// Note: if the effective gasTipCap is negative, this method returns both error
// the actual negative value, _and_ ErrGasFeeCapTooLow
func (tx *Transaction) EffectiveGasTip(baseFee *big.Int) (*big.Int, error) {
	if baseFee == nil {
		return tx.GasTipCap(), nil
	}
	var err error
	gasFeeCap := tx.GasFeeCap()
	if gasFeeCap.Cmp(baseFee) == -1 {
		err = ErrGasFeeCapTooLow
	}
	return math.BigMin(tx.GasTipCap(), gasFeeCap.Sub(gasFeeCap, baseFee)), err
}

// EffectiveGasTipValue is identical to EffectiveGasTip, but does not return an
// error in case the effective gasTipCap is negative
func (tx *Transaction) EffectiveGasTipValue(baseFee *big.Int) *big.Int {
	effectiveTip, _ := tx.EffectiveGasTip(baseFee)
	return effectiveTip
}

// EffectiveGasTipCmp compares the effective gasTipCap of two transactions assuming the given base fee.
func (tx *Transaction) EffectiveGasTipCmp(other *Transaction, baseFee *big.Int) int {
	if baseFee == nil {
		return tx.GasTipCapCmp(other)
	}
	return tx.EffectiveGasTipValue(baseFee).Cmp(other.EffectiveGasTipValue(baseFee))
}

// EffectiveGasTipIntCmp compares the effective gasTipCap of a transaction to the given gasTipCap.
func (tx *Transaction) EffectiveGasTipIntCmp(other *big.Int, baseFee *big.Int) int {
	if baseFee == nil {
		return tx.GasTipCapIntCmp(other)
	}
	return tx.EffectiveGasTipValue(baseFee).Cmp(other)
}

// BundleResource returns the bundle execution resource for bundle transactions, 0 otherwise.
func (tx *Transaction) BundleResource() uint64 {
	if bundletx, ok := tx.inner.(*BundleTx); ok {
		return bundletx.bundleResource()
	}
	return 0
}

// BundleValueFeeCap returns the bundle value-fee cap for bundle transactions, nil otherwise.
func (tx *Transaction) BundleValueFeeCap() *big.Int {
	if bundletx, ok := tx.inner.(*BundleTx); ok {
		return bundletx.BundleFeeCap.ToBig()
	}
	return nil
}

// BundleHashes returns the commitment hashes for bundle transactions, nil otherwise.
func (tx *Transaction) BundleHashes() []common.Hash {
	if bundletx, ok := tx.inner.(*BundleTx); ok {
		return bundletx.BundleHashes
	}
	return nil
}

// BundleAttachment returns the attachment of a bundle transaction, nil otherwise.
func (tx *Transaction) BundleAttachment() *BundleAttachment {
	if bundletx, ok := tx.inner.(*BundleTx); ok {
		return bundletx.Attachment
	}
	return nil
}

// BundleValueFeeCapCmp compares the bundle value-fee cap of two transactions.
func (tx *Transaction) BundleValueFeeCapCmp(other *Transaction) int {
	return tx.BundleValueFeeCap().Cmp(other.BundleValueFeeCap())
}

// BundleValueFeeCapIntCmp compares the bundle value-fee cap against the given bundle fee cap.
func (tx *Transaction) BundleValueFeeCapIntCmp(other *big.Int) int {
	return tx.BundleValueFeeCap().Cmp(other)
}

// WithoutBundleAttachment returns a copy of tx with the bundle attachment removed.
func (tx *Transaction) WithoutBundleAttachment() *Transaction {
	bundletx, ok := tx.inner.(*BundleTx)
	if !ok {
		return tx
	}
	cpy := &Transaction{
		inner: bundletx.withoutAttachment(),
		time:  tx.time,
	}
	// Note: tx.size cache not carried over because the attachment is included in size!
	if h := tx.hash.Load(); h != nil {
		cpy.hash.Store(h)
	}
	if f := tx.from.Load(); f != nil {
		cpy.from.Store(f)
	}
	return cpy
}

// SetTime sets the decoding time of a transaction. This is used by tests to set
// arbitrary times and by persistent flow spaces when loading old txs from
// disk.
func (tx *Transaction) SetTime(t time.Time) {
	tx.time = t
}

// Time returns the time when the transaction was first seen on the network. It
// is a heuristic to prefer mining older txs vs new all other things equal.
func (tx *Transaction) Time() time.Time {
	return tx.time
}

// Hash returns the transaction hash.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}

	var h common.Hash
	if tx.Type() == DirectTxType {
		h = rlpHash(tx.inner)
	} else {
		h = prefixedRlpHash(tx.Type(), tx.inner)
	}
	tx.hash.Store(h)
	return h
}

// Size returns the true encoded storage size of the transaction, either by encoding
// and returning it, or returning a previously cached value.
func (tx *Transaction) Size() uint64 {
	if size := tx.size.Load(); size != nil {
		return size.(uint64)
	}

	// Cache miss, encode and cache.
	// Note we rely on the assumption that all tx.inner values are RLP-encoded!
	c := writeCounter(0)
	rlp.Encode(&c, &tx.inner)
	size := uint64(c)

	// For bundle transactions, add the size of the attachment payload and the outer list
	// of the transaction + attachment encoding.
	if sc := tx.BundleAttachment(); sc != nil {
		size += rlp.ListSize(sc.encodedSize())
	}

	// For typed transactions, the encoding also includes the leading type byte.
	if tx.Type() != DirectTxType {
		size += 1
	}

	tx.size.Store(size)
	return size
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	if r == nil || s == nil || v == nil {
		return nil, fmt.Errorf("%w: r: %s, s: %s, v: %s", ErrInvalidSig, r, s, v)
	}
	cpy := tx.inner.copy()
	cpy.setSignatureValues(signer.ChainID(), v, r, s)
	return &Transaction{inner: cpy, time: tx.time}, nil
}

// Transactions implements DerivableList for transactions.
type Transactions []*Transaction

// Len returns the length of s.
func (s Transactions) Len() int { return len(s) }

// EncodeIndex encodes the i'th transaction to w. Note that this does not check for errors
// because we assume that *Transaction will only ever contain valid txs that were either
// constructed by decoding or via public API in this package.
func (s Transactions) EncodeIndex(i int, w *bytes.Buffer) {
	tx := s[i]
	if tx.Type() == DirectTxType {
		rlp.Encode(w, tx.inner)
	} else {
		tx.encodeTyped(w)
	}
}

// TxDifference returns a new set which is the difference between a and b.
func TxDifference(a, b Transactions) Transactions {
	keep := make(Transactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// HashDifference returns a new set which is the difference between a and b.
func HashDifference(a, b []common.Hash) []common.Hash {
	keep := make([]common.Hash, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, hash := range b {
		remove[hash] = struct{}{}
	}

	for _, hash := range a {
		if _, ok := remove[hash]; !ok {
			keep = append(keep, hash)
		}
	}

	return keep
}

// TxByNonce implements the sort interface to allow sorting a list of transactions
// by their nonces. This is usually only useful for sorting transactions from a
// single account, otherwise a nonce comparison doesn't make much sense.
type TxByNonce Transactions

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].Nonce() < s[j].Nonce() }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// copyAddressPtr copies an address.
func copyAddressPtr(a *common.Address) *common.Address {
	if a == nil {
		return nil
	}
	cpy := *a
	return &cpy
}
