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

// Package corexconfig contains the configuration of the CoreXChain runtime.
package corexconfig

import (
	"errors"
	"time"

	"com.corexkey/common"
	"com.corexkey/consensus"
	"com.corexkey/consensus/beacon"
	"com.corexkey/consensus/clique"
	"com.corexkey/consensus/corexhash"
	"com.corexkey/core"
	"com.corexkey/core/flowspace/bundlepool"
	"com.corexkey/core/flowspace/directpool"
	"com.corexkey/corex/downloader"
	productionengine "com.corexkey/corex/production"
	"com.corexkey/corex/valuepolicy"
	"com.corexkey/corexdb"
	"com.corexkey/params"
)

// FullNodeValuePolicy contains default fee-oracle settings for a full node.
var FullNodeValuePolicy = valuepolicy.Config{
	Blocks:           20,
	Percentile:       60,
	MaxHeaderHistory: 1024,
	MaxBlockHistory:  1024,
	MaxPrice:         valuepolicy.DefaultMaxPriorityFee,
	IgnorePrice:      valuepolicy.DefaultIgnorePriorityFee,
}

// CommunicationConfig describes the communication layer profile exposed in CoreXChain materials.
type CommunicationConfig struct {
	Protocol       string
	Transport      string
	Discovery      string
	Connector      string
	SettlementFlow string
}

// GovernanceConfig describes the governance layer profile exposed in CoreXChain materials.
type GovernanceConfig struct {
	Model        string
	Coordination string
	Committee    string
	Lifecycle    string
}

// TrustConfig describes the trust layer profile exposed in CoreXChain materials.
type TrustConfig struct {
	Model      string
	Validation string
	Proofing   string
	Finality   string
}

// AssetConfig describes the value model used by CoreXChain.
type AssetConfig struct {
	BaseAsset       string
	SettlementAsset string
	ValueFlow       string
	Anchoring       string
}

// NodeRoleConfig describes how a node is positioned inside the CoreXChain topology.
type NodeRoleConfig struct {
	Default string
	Enabled []string
}

var (
	DefaultCommunicationConfig = CommunicationConfig{
		Protocol:       "CWW",
		Transport:      "corex-mesh",
		Discovery:      "corexdisco",
		Connector:      "UTXO Connector",
		SettlementFlow: "stream-aligned value settlement",
	}
	DefaultGovernanceConfig = GovernanceConfig{
		Model:        "on-chain governance",
		Coordination: "CWW governance coordination",
		Committee:    "governance and sync committees",
		Lifecycle:    "proposal-validation-execution",
	}
	DefaultTrustConfig = TrustConfig{
		Model:      "PoV",
		Validation: "value-aware validation",
		Proofing:   "zero-knowledge assisted verification",
		Finality:   "trust-layer checkpoints",
	}
	DefaultAssetConfig = AssetConfig{
		BaseAsset:       "CXK",
		SettlementAsset: "CoreX",
		ValueFlow:       "contribution-linked circulation",
		Anchoring:       "value-centric settlement",
	}
	DefaultNodeRoleConfig = NodeRoleConfig{
		Default: "communication",
		Enabled: []string{"communication", "backbone", "governance", "trust", "sync-committee"},
	}
)

// Defaults contains default settings for use on the CoreXChain main net.
var Defaults = Config{
	SyncMode:           downloader.SnapSync,
	NetworkId:          0, // enable auto configuration of networkID == chainID
	Communication:      DefaultCommunicationConfig,
	Governance:         DefaultGovernanceConfig,
	Trust:              DefaultTrustConfig,
	Asset:              DefaultAssetConfig,
	NodeRole:           DefaultNodeRoleConfig,
	TxLookupLimit:      2350000,
	TransactionHistory: 2350000,
	StateHistory:       params.FullImmutabilityThreshold,
	LightPeers:         100,
	DatabaseCache:      512,
	TrieCleanCache:     154,
	TrieDirtyCache:     256,
	TrieTimeout:        60 * time.Minute,
	SnapshotCache:      102,
	FilterLogCacheSize: 32,
	Production:         productionengine.DefaultConfig,
	FlowSpace:          directpool.DefaultConfig,
	BundlePool:         bundlepool.DefaultConfig,
	RPCGasCap:          50000000,
	RPCEVMTimeout:      5 * time.Second,
	ValuePolicy:        FullNodeValuePolicy,
	RPCTxFeeCap:        1, // 1 corex
}

//go:generate go run github.com/fjl/gencodec -type Config -formats toml -out gen_config.go

// Config contains configuration options for CoreXChain runtime components.
type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the CoreXChain main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Network ID separates blockchains on the peer-to-peer networking level. When left
	// zero, the chain ID is used as network ID.
	NetworkId uint64
	SyncMode  downloader.SyncMode

	// This can be set to list of enrtree:// URLs which will be queried for
	// for nodes to connect to.
	CoreDiscoveryURLs []string
	SnapDiscoveryURLs []string

	// High-level CoreXChain architecture profiles.
	Communication CommunicationConfig `toml:",omitempty"`
	Governance    GovernanceConfig    `toml:",omitempty"`
	Trust         TrustConfig         `toml:",omitempty"`
	Asset         AssetConfig         `toml:",omitempty"`
	NodeRole      NodeRoleConfig      `toml:",omitempty"`

	NoPruning  bool // Whether to disable pruning and flush everything to disk
	NoPrefetch bool // Whether to disable prefetching and only load state on demand

	// Deprecated, use 'TransactionHistory' instead.
	TxLookupLimit      uint64 `toml:",omitempty"` // The maximum number of blocks from head whose tx indices are reserved.
	TransactionHistory uint64 `toml:",omitempty"` // The maximum number of blocks from head whose tx indices are reserved.
	StateHistory       uint64 `toml:",omitempty"` // The maximum number of blocks from head whose state histories are reserved.

	// State scheme represents the scheme used to store corexchain states and trie
	// nodes on top. It can be 'hash', 'path', or none which means use the scheme
	// consistent with persistent state.
	StateScheme string `toml:",omitempty"`

	// RequiredBlocks is a set of block number -> hash mappings which must be in the
	// canonical chain of all remote peers. Setting the option makes corex verify the
	// presence of these blocks for every new peer connection.
	RequiredBlocks map[uint64]common.Hash `toml:"-"`

	// Light client options
	LightServ        int  `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightIngress     int  `toml:",omitempty"` // Incoming bandwidth limit for light servers
	LightEgress      int  `toml:",omitempty"` // Outgoing bandwidth limit for light servers
	LightPeers       int  `toml:",omitempty"` // Maximum number of LES client peers
	LightNoPrune     bool `toml:",omitempty"` // Whether to disable light chain pruning
	LightNoSyncServe bool `toml:",omitempty"` // Whether to serve light clients before syncing

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	DatabaseFreezer    string

	TrieCleanCache int
	TrieDirtyCache int
	TrieTimeout    time.Duration
	SnapshotCache  int
	Preimages      bool

	// This is the number of blocks for which logs will be cached in the filter system.
	FilterLogCacheSize int

	// Block production options
	Production productionengine.Config

	// Flow-space options
	FlowSpace  directpool.Config
	BundlePool bundlepool.Config

	// Value policy / fee oracle options
	ValuePolicy valuepolicy.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot string `toml:"-"`

	// RPCGasCap is the global gas cap for corex-call variants.
	RPCGasCap uint64

	// RPCEVMTimeout is the global timeout for corex-call.
	RPCEVMTimeout time.Duration

	// RPCTxFeeCap is the global transaction fee(price * gaslimit) cap for
	// send-transaction variants. The unit is corex.
	RPCTxFeeCap float64

	// OverrideCancun (TODO: remove after the fork)
	OverrideCancun *uint64 `toml:",omitempty"`

	// OverrideVerkle (TODO: remove after the fork)
	OverrideVerkle *uint64 `toml:",omitempty"`
}

// CreateConsensusEngine creates a consensus engine for the given chain config.
// Clique remains available as a lab trust engine, while corexhash is treated as
// a legacy execution artifact.
func CreateConsensusEngine(config *params.ChainConfig, db corexdb.Database) (consensus.Engine, error) {
	// If the legacy lab authority engine is requested, set it up.
	if config.Clique != nil {
		return beacon.New(clique.New(config.Clique, db)), nil
	}
	// CoreXHash is retained only for already transitioned networks.
	if !config.TerminalTotalDifficultyPassed {
		return nil, errors.New("corexhash is retained only as a legacy execution artifact on networks that already completed trust transition")
	}
	return beacon.New(corexhash.NewFaker()), nil
}
