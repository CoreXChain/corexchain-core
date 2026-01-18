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

package valuepolicy

import (
	"context"
	"math/big"
	"sync"

	"com.corexkey/common"
	"com.corexkey/common/lru"
	"com.corexkey/core"
	"com.corexkey/core/types"
	"com.corexkey/event"
	"com.corexkey/log"
	"com.corexkey/params"
	"com.corexkey/rpc"
	"golang.org/x/exp/slices"
)

const sampleNumber = 3 // Number of transactions sampled in a block

var (
	DefaultMaxPriorityFee    = big.NewInt(500 * params.GWei)
	DefaultIgnorePriorityFee = big.NewInt(2 * params.Nano)
)

type Config struct {
	Blocks           int
	Percentile       int
	MaxHeaderHistory uint64
	MaxBlockHistory  uint64
	Default          *big.Int `toml:",omitempty"`
	MaxPrice         *big.Int `toml:",omitempty"`
	IgnorePrice      *big.Int `toml:",omitempty"`
}

// Backend includes all necessary background APIs for oracle.
type Backend interface {
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error)
	PendingBlockAndReceipts() (*types.Block, types.Receipts)
	ChainConfig() *params.ChainConfig
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
}

// FeeOracle recommends priority fees based on the content of recent
// blocks. Suitable for both light and full clients.
type FeeOracle struct {
	backend        Backend
	lastHead       common.Hash
	lastSuggestion *big.Int
	maxPrice       *big.Int
	ignorePrice    *big.Int
	cacheLock      sync.RWMutex
	fetchLock      sync.Mutex

	checkBlocks, percentile           int
	maxHeaderHistory, maxBlockHistory uint64

	historyCache *lru.Cache[cacheKey, processedFees]
}

// NewFeeOracle returns a new value policy oracle which can recommend a suitable
// priority fee for newly created value flows.
func NewFeeOracle(backend Backend, params Config) *FeeOracle {
	blocks := params.Blocks
	if blocks < 1 {
		blocks = 1
		log.Warn("Sanitizing invalid value policy oracle sample blocks", "provided", params.Blocks, "updated", blocks)
	}
	percent := params.Percentile
	if percent < 0 {
		percent = 0
		log.Warn("Sanitizing invalid value policy oracle sample percentile", "provided", params.Percentile, "updated", percent)
	} else if percent > 100 {
		percent = 100
		log.Warn("Sanitizing invalid value policy oracle sample percentile", "provided", params.Percentile, "updated", percent)
	}
	maxPrice := params.MaxPrice
	if maxPrice == nil || maxPrice.Int64() <= 0 {
		maxPrice = DefaultMaxPriorityFee
		log.Warn("Sanitizing invalid value policy oracle price cap", "provided", params.MaxPrice, "updated", maxPrice)
	}
	ignorePrice := params.IgnorePrice
	if ignorePrice == nil || ignorePrice.Int64() <= 0 {
		ignorePrice = DefaultIgnorePriorityFee
		log.Warn("Sanitizing invalid value policy oracle ignore price", "provided", params.IgnorePrice, "updated", ignorePrice)
	} else if ignorePrice.Int64() > 0 {
		log.Info("Value policy oracle is ignoring threshold set", "threshold", ignorePrice)
	}
	maxHeaderHistory := params.MaxHeaderHistory
	if maxHeaderHistory < 1 {
		maxHeaderHistory = 1
		log.Warn("Sanitizing invalid value policy oracle max header history", "provided", params.MaxHeaderHistory, "updated", maxHeaderHistory)
	}
	maxBlockHistory := params.MaxBlockHistory
	if maxBlockHistory < 1 {
		maxBlockHistory = 1
		log.Warn("Sanitizing invalid value policy oracle max block history", "provided", params.MaxBlockHistory, "updated", maxBlockHistory)
	}

	cache := lru.NewCache[cacheKey, processedFees](2048)
	headEvent := make(chan core.ChainHeadEvent, 1)
	backend.SubscribeChainHeadEvent(headEvent)
	go func() {
		var lastHead common.Hash
		for ev := range headEvent {
			if ev.Block.ParentHash() != lastHead {
				cache.Purge()
			}
			lastHead = ev.Block.Hash()
		}
	}()

	return &FeeOracle{
		backend:          backend,
		lastSuggestion:   params.Default,
		maxPrice:         maxPrice,
		ignorePrice:      ignorePrice,
		checkBlocks:      blocks,
		percentile:       percent,
		maxHeaderHistory: maxHeaderHistory,
		maxBlockHistory:  maxBlockHistory,
		historyCache:     cache,
	}
}

// SuggestTipCap returns a tip cap so that newly created transaction can have a
// very high chance to be included in the following blocks.
//
// Note, for classic direct transactions and the historical corex_valueFlowFee RPC call, it will be
// necessary to add the basefee to the returned number to fall back to the legacy
// behavior.
func (oracle *FeeOracle) SuggestTipCap(ctx context.Context) (*big.Int, error) {
	head, _ := oracle.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber)
	headHash := head.Hash()

	// If the latest value policy is still available, return it.
	oracle.cacheLock.RLock()
	lastHead, lastSuggestion := oracle.lastHead, oracle.lastSuggestion
	oracle.cacheLock.RUnlock()
	if headHash == lastHead {
		return new(big.Int).Set(lastSuggestion), nil
	}
	oracle.fetchLock.Lock()
	defer oracle.fetchLock.Unlock()

	// Try checking the cache again, maybe the last fetch fetched what we need
	oracle.cacheLock.RLock()
	lastHead, lastSuggestion = oracle.lastHead, oracle.lastSuggestion
	oracle.cacheLock.RUnlock()
	if headHash == lastHead {
		return new(big.Int).Set(lastSuggestion), nil
	}
	var (
		sent, exp int
		number    = head.Number.Uint64()
		result    = make(chan results, oracle.checkBlocks)
		quit      = make(chan struct{})
		results   []*big.Int
	)
	for sent < oracle.checkBlocks && number > 0 {
		go oracle.getBlockValues(ctx, number, sampleNumber, oracle.ignorePrice, result, quit)
		sent++
		exp++
		number--
	}
	for exp > 0 {
		res := <-result
		if res.err != nil {
			close(quit)
			return new(big.Int).Set(lastSuggestion), res.err
		}
		exp--
		// Nothing returned. There are two special cases here:
		// - The block is empty
		// - All the transactions included are sent by the block producer itself.
		// In these cases, use the latest calculated suggestion for sampling.
		if len(res.values) == 0 {
			res.values = []*big.Int{lastSuggestion}
		}
		// Besides, in order to collect enough data for sampling, if nothing
		// meaningful returned, try to query more blocks. But the maximum
		// is 2*checkBlocks.
		if len(res.values) == 1 && len(results)+1+exp < oracle.checkBlocks*2 && number > 0 {
			go oracle.getBlockValues(ctx, number, sampleNumber, oracle.ignorePrice, result, quit)
			sent++
			exp++
			number--
		}
		results = append(results, res.values...)
	}
	price := lastSuggestion
	if len(results) > 0 {
		slices.SortFunc(results, func(a, b *big.Int) int { return a.Cmp(b) })
		price = results[(len(results)-1)*oracle.percentile/100]
	}
	if price.Cmp(oracle.maxPrice) > 0 {
		price = new(big.Int).Set(oracle.maxPrice)
	}
	oracle.cacheLock.Lock()
	oracle.lastHead = headHash
	oracle.lastSuggestion = price
	oracle.cacheLock.Unlock()

	return new(big.Int).Set(price), nil
}

type results struct {
	values []*big.Int
	err    error
}

// getBlockValues calculates the lowest transaction priority fee in a given block
// and sends it to the result channel. If the block is empty or all transactions
// are sent by the block producer itself(it doesn't make any sense to include this kind of
// transaction prices for sampling), nil value policy is returned.
func (oracle *FeeOracle) getBlockValues(ctx context.Context, blockNum uint64, limit int, ignoreUnder *big.Int, result chan results, quit chan struct{}) {
	block, err := oracle.backend.BlockByNumber(ctx, rpc.BlockNumber(blockNum))
	if block == nil {
		select {
		case result <- results{nil, err}:
		case <-quit:
		}
		return
	}
	signer := types.MakeSigner(oracle.backend.ChainConfig(), block.Number(), block.Time())

	// Sort the transaction by effective tip in ascending sort.
	txs := block.Transactions()
	sortedTxs := make([]*types.Transaction, len(txs))
	copy(sortedTxs, txs)
	baseFee := block.BaseFee()
	slices.SortFunc(sortedTxs, func(a, b *types.Transaction) int {
		// It's okay to discard the error because a tx would never be
		// accepted into a block with an invalid effective tip.
		tip1, _ := a.EffectiveGasTip(baseFee)
		tip2, _ := b.EffectiveGasTip(baseFee)
		return tip1.Cmp(tip2)
	})

	var prices []*big.Int
	for _, tx := range sortedTxs {
		tip, _ := tx.EffectiveGasTip(baseFee)
		if ignoreUnder != nil && tip.Cmp(ignoreUnder) == -1 {
			continue
		}
		sender, err := types.Sender(signer, tx)
		if err == nil && sender != block.Coinbase() {
			prices = append(prices, tip)
			if len(prices) >= limit {
				break
			}
		}
	}
	select {
	case result <- results{prices, nil}:
	case <-quit:
	}
}
