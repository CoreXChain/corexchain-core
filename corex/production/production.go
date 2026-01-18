package production

import (
	"math/big"
	"time"

	"com.corexkey/common"
	"com.corexkey/consensus"
	"com.corexkey/core/state"
	"com.corexkey/core/types"
	"com.corexkey/event"
	"com.corexkey/params"
)

// Config describes the local block-production profile.
type Config struct {
	FeeFloor          *big.Int
	ResourceCeil      uint64
	RewardBase        common.Address
	ExtraData         []byte
	Recommit          time.Duration
	NewPayloadTimeout time.Duration
}

// DefaultConfig provides a minimal default production profile.
var DefaultConfig = Config{
	FeeFloor:          big.NewInt(1),
	ResourceCeil:      30_000_000,
	Recommit:          2 * time.Second,
	NewPayloadTimeout: 2 * time.Second,
}

// Producer is a lightweight placeholder for CoreXChain block production.
type Producer struct {
	config     *Config
	pending    *types.Block
	pendingDB  *state.StateDB
	producing  bool
	rewardBase common.Address
}

// New creates a new local producer placeholder.
func New(_ any, cfg *Config, _ *params.ChainConfig, _ *event.TypeMux, _ consensus.Engine, _ func(*types.Header) bool) *Producer {
	if cfg == nil {
		cfg = &DefaultConfig
	}
	return &Producer{config: cfg, rewardBase: cfg.RewardBase}
}

// PendingBlock returns the current pending block placeholder.
func (p *Producer) PendingBlock() *types.Block { return p.pending }

// PendingBlockAndReceipts returns the current pending block and receipts placeholder.
func (p *Producer) PendingBlockAndReceipts() (*types.Block, types.Receipts) { return p.pending, nil }

// Pending returns the current pending block and state placeholder.
func (p *Producer) Pending() (*types.Block, *state.StateDB) { return p.pending, p.pendingDB }

// SubscribePendingLogs creates a no-op subscription for pending logs.
func (p *Producer) SubscribePendingLogs(_ chan<- []*types.Log) event.Subscription {
	return event.NewSubscription(func(<-chan struct{}) error { return nil })
}

// SetExtra updates the metadata carried into locally produced blocks.
func (p *Producer) SetExtra(extra []byte) error {
	p.config.ExtraData = append([]byte(nil), extra...)
	return nil
}

// SetFeeFloor updates the local fee floor.
func (p *Producer) SetFeeFloor(feeFloor *big.Int) {
	if feeFloor == nil {
		p.config.FeeFloor = nil
		return
	}
	p.config.FeeFloor = new(big.Int).Set(feeFloor)
}

// SetGasTip is retained as a compatibility alias for SetFeeFloor.
func (p *Producer) SetGasTip(feeFloor *big.Int) { p.SetFeeFloor(feeFloor) }

// SetResourceCeil updates the local block resource ceiling.
func (p *Producer) SetResourceCeil(gasLimit uint64) { p.config.ResourceCeil = gasLimit }

// SetGasCeil is retained as a compatibility alias for SetResourceCeil.
func (p *Producer) SetGasCeil(gasLimit uint64) { p.SetResourceCeil(gasLimit) }

// SetRewardBase updates the local reward recipient.
func (p *Producer) SetRewardBase(rewardBase common.Address) {
	p.rewardBase = rewardBase
	p.config.RewardBase = rewardBase
}

// SetRecommitInterval updates the local block assembly refresh interval.
func (p *Producer) SetRecommitInterval(interval time.Duration) { p.config.Recommit = interval }

// Start marks the local producer as active.
func (p *Producer) Start() { p.producing = true }

// Stop marks the local producer as inactive.
func (p *Producer) Stop() { p.producing = false }

// Producing reports whether local block production is active.
func (p *Producer) Producing() bool { return p.producing }

// Mining is retained as a compatibility alias.
func (p *Producer) Mining() bool { return p.Producing() }

// ProductionRate reports a placeholder production-rate indicator.
func (p *Producer) ProductionRate() uint64 { return 0 }

// Hashrate is retained as a compatibility alias.
func (p *Producer) Hashrate() uint64 { return p.ProductionRate() }

// Close shuts down the local producer placeholder.
func (p *Producer) Close() {}
