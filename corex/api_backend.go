// Copyright 2015 CoreX Team
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

package corex

import (
	"context"
	"errors"
	"math/big"
	"time"

	"com.corexkey"
	"com.corexkey/accounts"
	"com.corexkey/common"
	"com.corexkey/consensus"
	"com.corexkey/core"
	"com.corexkey/core/bloombits"
	"com.corexkey/core/flowspace"
	"com.corexkey/core/rawdb"
	"com.corexkey/core/state"
	"com.corexkey/core/types"
	"com.corexkey/core/vm"
	productionengine "com.corexkey/corex/production"
	"com.corexkey/corex/tracers"
	"com.corexkey/corex/valuepolicy"
	"com.corexkey/corexdb"
	"com.corexkey/event"
	"com.corexkey/params"
	"com.corexkey/rpc"
)

// CoreXAPIBackend implements corexapi.Backend and tracers.Backend for full nodes.
type CoreXAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	corex               *CoreXChain
	feeOracle           *valuepolicy.FeeOracle
}

// ChainConfig returns the active chain configuration.
func (b *CoreXAPIBackend) ChainConfig() *params.ChainConfig {
	return b.corex.blockchain.Config()
}

func (b *CoreXAPIBackend) CurrentBlock() *types.Header {
	return b.corex.blockchain.CurrentBlock()
}

func (b *CoreXAPIBackend) SetHead(number uint64) {
	b.corex.handler.downloader.Cancel()
	b.corex.blockchain.SetHead(number)
}

func (b *CoreXAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the block producer.
	if number == rpc.PendingBlockNumber {
		block := b.corex.production.PendingBlock()
		if block == nil {
			return nil, errors.New("pending block is not available")
		}
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.corex.blockchain.CurrentBlock(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		block := b.corex.blockchain.CurrentFinalBlock()
		if block == nil {
			return nil, errors.New("finalized block not found")
		}
		return block, nil
	}
	if number == rpc.SafeBlockNumber {
		block := b.corex.blockchain.CurrentSafeBlock()
		if block == nil {
			return nil, errors.New("safe block not found")
		}
		return block, nil
	}
	return b.corex.blockchain.GetHeaderByNumber(uint64(number)), nil
}

func (b *CoreXAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.corex.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.corex.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *CoreXAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.corex.blockchain.GetHeaderByHash(hash), nil
}

func (b *CoreXAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the block producer.
	if number == rpc.PendingBlockNumber {
		block := b.corex.production.PendingBlock()
		if block == nil {
			return nil, errors.New("pending block is not available")
		}
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		header := b.corex.blockchain.CurrentBlock()
		return b.corex.blockchain.GetBlock(header.Hash(), header.Number.Uint64()), nil
	}
	if number == rpc.FinalizedBlockNumber {
		header := b.corex.blockchain.CurrentFinalBlock()
		if header == nil {
			return nil, errors.New("finalized block not found")
		}
		return b.corex.blockchain.GetBlock(header.Hash(), header.Number.Uint64()), nil
	}
	if number == rpc.SafeBlockNumber {
		header := b.corex.blockchain.CurrentSafeBlock()
		if header == nil {
			return nil, errors.New("safe block not found")
		}
		return b.corex.blockchain.GetBlock(header.Hash(), header.Number.Uint64()), nil
	}
	return b.corex.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *CoreXAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.corex.blockchain.GetBlockByHash(hash), nil
}

// GetBody returns body of a block. It does not resolve special block numbers.
func (b *CoreXAPIBackend) GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error) {
	if number < 0 || hash == (common.Hash{}) {
		return nil, errors.New("invalid arguments; expect hash and no special block numbers")
	}
	if body := b.corex.blockchain.GetBody(hash); body != nil {
		return body, nil
	}
	return nil, errors.New("block body not found")
}

func (b *CoreXAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.corex.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.corex.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.corex.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *CoreXAPIBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	return b.corex.production.PendingBlockAndReceipts()
}

func (b *CoreXAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the block producer.
	if number == rpc.PendingBlockNumber {
		block, state := b.corex.production.Pending()
		if block == nil || state == nil {
			return nil, nil, errors.New("pending state is not available")
		}
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.corex.BlockChain().StateAt(header.Root)
	if err != nil {
		return nil, nil, err
	}
	return stateDb, header, nil
}

func (b *CoreXAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.corex.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.corex.BlockChain().StateAt(header.Root)
		if err != nil {
			return nil, nil, err
		}
		return stateDb, header, nil
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *CoreXAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.corex.blockchain.GetReceiptsByHash(hash), nil
}

func (b *CoreXAPIBackend) GetLogs(ctx context.Context, hash common.Hash, number uint64) ([][]*types.Log, error) {
	return rawdb.ReadLogs(b.corex.chainDb, hash, number), nil
}

func (b *CoreXAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	if header := b.corex.blockchain.GetHeaderByHash(hash); header != nil {
		return b.corex.blockchain.GetTd(hash, header.Number.Uint64())
	}
	return nil
}

func (b *CoreXAPIBackend) GetEVM(ctx context.Context, msg *core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config, blockCtx *vm.BlockContext) *vm.EVM {
	if vmConfig == nil {
		vmConfig = b.corex.blockchain.GetVMConfig()
	}
	txContext := core.NewEVMTxContext(msg)
	var context vm.BlockContext
	if blockCtx != nil {
		context = *blockCtx
	} else {
		context = core.NewEVMBlockContext(header, b.corex.BlockChain(), nil)
	}
	return vm.NewEVM(context, txContext, state, b.ChainConfig(), *vmConfig)
}

func (b *CoreXAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.corex.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *CoreXAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.corex.production.SubscribePendingLogs(ch)
}

func (b *CoreXAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.corex.BlockChain().SubscribeChainEvent(ch)
}

func (b *CoreXAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.corex.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *CoreXAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.corex.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *CoreXAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.corex.BlockChain().SubscribeLogsEvent(ch)
}

func (b *CoreXAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.corex.flowSpace.Add([]*types.Transaction{signedTx}, true, false)[0]
}

func (b *CoreXAPIBackend) GetFlowSpaceTransactions() (types.Transactions, error) {
	pending := b.corex.flowSpace.Pending(flowspace.PendingFilter{})
	var txs types.Transactions
	for _, batch := range pending {
		for _, lazy := range batch {
			if tx := lazy.Resolve(); tx != nil {
				txs = append(txs, tx)
			}
		}
	}
	return txs, nil
}

func (b *CoreXAPIBackend) GetFlowSpaceTransaction(hash common.Hash) *types.Transaction {
	return b.corex.flowSpace.Get(hash)
}

// GetTransaction retrieves the lookup along with the transaction itself associate
// with the given transaction hash.
//
// An error will be returned if the transaction is not found, and background
// indexing for transactions is still in progress. The error is used to indicate the
// scenario explicitly that the transaction might be reachable shortly.
//
// A null will be returned in the transaction is not found and background transaction
// indexing is already finished. The transaction is not existent from the perspective
// of node.
func (b *CoreXAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64, error) {
	lookup, tx, err := b.corex.blockchain.GetTransactionLookup(txHash)
	if err != nil {
		return false, nil, common.Hash{}, 0, 0, err
	}
	if lookup == nil || tx == nil {
		return false, nil, common.Hash{}, 0, 0, nil
	}
	return true, tx, lookup.BlockHash, lookup.BlockIndex, lookup.Index, nil
}

func (b *CoreXAPIBackend) GetFlowSpaceNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.corex.flowSpace.Nonce(addr), nil
}

func (b *CoreXAPIBackend) Stats() (runnable int, blocked int) {
	return b.corex.flowSpace.Stats()
}

func (b *CoreXAPIBackend) FlowSpaceContent() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	return b.corex.flowSpace.Content()
}

func (b *CoreXAPIBackend) FlowSpaceContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	return b.corex.flowSpace.ContentFrom(addr)
}

func (b *CoreXAPIBackend) FlowSpace() *flowspace.FlowSpace {
	return b.corex.flowSpace
}

func (b *CoreXAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.corex.flowSpace.SubscribeTransactions(ch, true)
}

func (b *CoreXAPIBackend) SyncProgress() corexchain.SyncProgress {
	prog := b.corex.Downloader().Progress()
	if txProg, err := b.corex.blockchain.TxIndexProgress(); err == nil {
		prog.TxIndexFinishedBlocks = txProg.Indexed
		prog.TxIndexRemainingBlocks = txProg.Remaining
	}
	return prog
}

func (b *CoreXAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.feeOracle.SuggestTipCap(ctx)
}

func (b *CoreXAPIBackend) FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock *big.Int, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	return b.feeOracle.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *CoreXAPIBackend) ChainDb() corexdb.Database {
	return b.corex.ChainDb()
}

func (b *CoreXAPIBackend) EventMux() *event.TypeMux {
	return b.corex.EventMux()
}

func (b *CoreXAPIBackend) AccountManager() *accounts.Manager {
	return b.corex.AccountManager()
}

func (b *CoreXAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *CoreXAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *CoreXAPIBackend) RPCGasCap() uint64 {
	return b.corex.config.RPCGasCap
}

func (b *CoreXAPIBackend) RPCEVMTimeout() time.Duration {
	return b.corex.config.RPCEVMTimeout
}

func (b *CoreXAPIBackend) RPCTxFeeCap() float64 {
	return b.corex.config.RPCTxFeeCap
}

func (b *CoreXAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.corex.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *CoreXAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.corex.bloomRequests)
	}
}

func (b *CoreXAPIBackend) Engine() consensus.Engine {
	return b.corex.engine
}

func (b *CoreXAPIBackend) CurrentHeader() *types.Header {
	return b.corex.blockchain.CurrentHeader()
}

func (b *CoreXAPIBackend) Producer() *productionengine.Producer {
	return b.corex.Producer()
}

func (b *CoreXAPIBackend) StartProduction() error {
	return b.corex.StartProduction()
}

func (b *CoreXAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, tracers.StateReleaseFunc, error) {
	return b.corex.stateAtBlock(ctx, block, reexec, base, readOnly, preferDisk)
}

func (b *CoreXAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (*core.Message, vm.BlockContext, *state.StateDB, tracers.StateReleaseFunc, error) {
	return b.corex.stateAtTransaction(ctx, block, txIndex, reexec)
}
