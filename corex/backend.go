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

// Package corex implements the CoreXChain protocol.
package corex

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"

	"com.corexkey/accounts"
	"com.corexkey/common"
	"com.corexkey/common/hexutil"
	"com.corexkey/consensus"
	"com.corexkey/consensus/beacon"
	"com.corexkey/consensus/clique"
	"com.corexkey/core"
	"com.corexkey/core/bloombits"
	"com.corexkey/core/flowspace"
	"com.corexkey/core/flowspace/bundlepool"
	"com.corexkey/core/flowspace/directpool"
	"com.corexkey/core/rawdb"
	"com.corexkey/core/state/pruner"
	"com.corexkey/core/types"
	"com.corexkey/core/vm"
	"com.corexkey/corex/corexconfig"
	"com.corexkey/corex/downloader"
	productionengine "com.corexkey/corex/production"
	"com.corexkey/corex/protocols/corex"
	"com.corexkey/corex/protocols/snap"
	"com.corexkey/corex/valuepolicy"
	"com.corexkey/corexdb"
	"com.corexkey/event"
	"com.corexkey/internal/corexapi"
	"com.corexkey/internal/shutdowncheck"
	"com.corexkey/log"
	"com.corexkey/node"
	"com.corexkey/p2p"
	"com.corexkey/p2p/dnsdisc"
	"com.corexkey/p2p/enode"
	"com.corexkey/params"
	"com.corexkey/rlp"
	"com.corexkey/rpc"
)

// Config contains the configuration options of the CRX protocol.
// Deprecated: use corexconfig.Config instead.
type Config = corexconfig.Config

// CoreXChain implements the CoreXChain full node service.
type CoreXChain struct {
	config *corexconfig.Config

	// Handlers
	flowSpace *flowspace.FlowSpace

	blockchain         *core.BlockChain
	handler            *handler
	coreDialCandidates enode.Iterator
	snapDialCandidates enode.Iterator
	merger             *consensus.Merger

	// DB interfaces
	chainDb corexdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests     chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer      *core.ChainIndexer             // Bloom indexer operating during block imports
	closeBloomHandler chan struct{}

	APIBackend *CoreXAPIBackend

	production *productionengine.Producer
	feeFloor   *big.Int
	rewardbase common.Address

	networkID     uint64
	netRPCService *corexapi.NetAPI

	p2pServer *p2p.Server

	lock sync.RWMutex // Protects the variadic fields (e.g. fee floor and rewardbase)

	shutdownTracker *shutdowncheck.ShutdownTracker // Tracks if and when the node has shutdown ungracefully
}

// New creates a new CoreXChain object (including the
// initialisation of the common CoreXChain object)
func New(stack *node.Node, config *corexconfig.Config) (*CoreXChain, error) {
	// Ensure configuration values are compatible and sane
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run corex.CoreXChain in light sync mode, light mode has been deprecated")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if config.Production.FeeFloor == nil || config.Production.FeeFloor.Cmp(common.Big0) <= 0 {
		log.Warn("Sanitizing invalid block-production fee floor", "provided", config.Production.FeeFloor, "updated", corexconfig.Defaults.Production.FeeFloor)
		config.Production.FeeFloor = new(big.Int).Set(corexconfig.Defaults.Production.FeeFloor)
	}
	if config.NoPruning && config.TrieDirtyCache > 0 {
		if config.SnapshotCache > 0 {
			config.TrieCleanCache += config.TrieDirtyCache * 3 / 5
			config.SnapshotCache += config.TrieDirtyCache * 2 / 5
		} else {
			config.TrieCleanCache += config.TrieDirtyCache
		}
		config.TrieDirtyCache = 0
	}
	log.Info("Allocated trie memory caches", "clean", common.StorageSize(config.TrieCleanCache)*1024*1024, "dirty", common.StorageSize(config.TrieDirtyCache)*1024*1024)

	// Assemble the CoreXChain object
	chainDb, err := stack.OpenDatabaseWithFreezer("chaindata", config.DatabaseCache, config.DatabaseHandles, config.DatabaseFreezer, "corex/db/chaindata/", false)
	if err != nil {
		return nil, err
	}
	scheme, err := rawdb.ParseStateScheme(config.StateScheme, chainDb)
	if err != nil {
		return nil, err
	}
	// Try to recover offline state pruning only in hash-based.
	if scheme == rawdb.HashScheme {
		if err := pruner.RecoverPruning(stack.ResolvePath(""), chainDb); err != nil {
			log.Error("Failed to recover state", "error", err)
		}
	}
	// Transfer block-production-related config to the execution engine config.
	chainConfig, err := core.LoadChainConfig(chainDb, config.Genesis)
	if err != nil {
		return nil, err
	}
	engine, err := corexconfig.CreateConsensusEngine(chainConfig, chainDb)
	if err != nil {
		return nil, err
	}
	networkID := config.NetworkId
	if networkID == 0 {
		networkID = chainConfig.ChainID.Uint64()
	}
	corex := &CoreXChain{
		config:            config,
		merger:            consensus.NewMerger(chainDb),
		chainDb:           chainDb,
		eventMux:          stack.EventMux(),
		accountManager:    stack.AccountManager(),
		engine:            engine,
		closeBloomHandler: make(chan struct{}),
		networkID:         networkID,
		feeFloor:          config.Production.FeeFloor,
		rewardbase:        config.Production.RewardBase,
		bloomRequests:     make(chan chan *bloombits.Retrieval),
		bloomIndexer:      core.NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
		p2pServer:         stack.Server(),
		shutdownTracker:   shutdowncheck.NewShutdownTracker(chainDb),
	}
	bcVersion := rawdb.ReadDatabaseVersion(chainDb)
	var dbVer = "<nil>"
	if bcVersion != nil {
		dbVer = fmt.Sprintf("%d", *bcVersion)
	}
	log.Info("Initialising CoreXChain protocol", "network", networkID, "dbversion", dbVer)

	if !config.SkipBcVersionCheck {
		if bcVersion != nil && *bcVersion > core.BlockChainVersion {
			return nil, fmt.Errorf("database version is v%d, CoreX %s only supports v%d", *bcVersion, params.VersionWithMeta, core.BlockChainVersion)
		} else if bcVersion == nil || *bcVersion < core.BlockChainVersion {
			if bcVersion != nil { // only print warning on upgrade, not on init
				log.Warn("Upgrade blockchain database version", "from", dbVer, "to", core.BlockChainVersion)
			}
			rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
		}
	}
	var (
		vmConfig = vm.Config{
			EnablePreimageRecording: config.EnablePreimageRecording,
		}
		cacheConfig = &core.CacheConfig{
			TrieCleanLimit:      config.TrieCleanCache,
			TrieCleanNoPrefetch: config.NoPrefetch,
			TrieDirtyLimit:      config.TrieDirtyCache,
			TrieDirtyDisabled:   config.NoPruning,
			TrieTimeLimit:       config.TrieTimeout,
			SnapshotLimit:       config.SnapshotCache,
			Preimages:           config.Preimages,
			StateHistory:        config.StateHistory,
			StateScheme:         scheme,
		}
	)
	// Override the chain config with provided settings.
	var overrides core.ChainOverrides
	if config.OverrideCancun != nil {
		overrides.OverrideCancun = config.OverrideCancun
	}
	if config.OverrideVerkle != nil {
		overrides.OverrideVerkle = config.OverrideVerkle
	}
	corex.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, config.Genesis, &overrides, corex.engine, vmConfig, corex.shouldPreserve, &config.TransactionHistory)
	if err != nil {
		return nil, err
	}
	corex.bloomIndexer.Start(corex.blockchain)

	if config.BundlePool.Datadir != "" {
		config.BundlePool.Datadir = stack.ResolvePath(config.BundlePool.Datadir)
	}
	bundlePool := bundlepool.New(config.BundlePool, corex.blockchain)

	if config.FlowSpace.Journal != "" {
		config.FlowSpace.Journal = stack.ResolvePath(config.FlowSpace.Journal)
	}
	directPool := directpool.New(config.FlowSpace, corex.blockchain)

	corex.flowSpace, err = flowspace.New(config.FlowSpace.PriceLimit, corex.blockchain, []flowspace.SubPool{directPool, bundlePool})
	if err != nil {
		return nil, err
	}
	// Permit the downloader to use the trie cache allowance during fast sync
	cacheLimit := cacheConfig.TrieCleanLimit + cacheConfig.TrieDirtyLimit + cacheConfig.SnapshotLimit
	if corex.handler, err = newHandler(&handlerConfig{
		Database:       chainDb,
		Chain:          corex.blockchain,
		FlowSpace:      corex.flowSpace,
		Merger:         corex.merger,
		Network:        networkID,
		Sync:           config.SyncMode,
		BloomCache:     uint64(cacheLimit),
		EventMux:       corex.eventMux,
		RequiredBlocks: config.RequiredBlocks,
	}); err != nil {
		return nil, err
	}

	corex.production = productionengine.New(corex, &config.Production, corex.blockchain.Config(), corex.EventMux(), corex.engine, corex.isLocalBlock)
	corex.production.SetExtra(makeExtraData(config.Production.ExtraData))

	corex.APIBackend = &CoreXAPIBackend{stack.Config().ExtRPCEnabled(), stack.Config().AllowUnprotectedTxs, corex, nil}
	if corex.APIBackend.allowUnprotectedTxs {
		log.Info("Unprotected value flows allowed")
	}
	valuePolicyParams := config.ValuePolicy
	if valuePolicyParams.Default == nil {
		valuePolicyParams.Default = config.Production.FeeFloor
	}
	corex.APIBackend.feeOracle = valuepolicy.NewFeeOracle(corex.APIBackend, valuePolicyParams)

	// Setup DNS discovery iterators.
	dnsclient := dnsdisc.NewClient(dnsdisc.Config{})
	corex.coreDialCandidates, err = dnsclient.NewIterator(corex.config.CoreDiscoveryURLs...)
	if err != nil {
		return nil, err
	}
	corex.snapDialCandidates, err = dnsclient.NewIterator(corex.config.SnapDiscoveryURLs...)
	if err != nil {
		return nil, err
	}

	// Start the RPC service
	corex.netRPCService = corexapi.NewNetAPI(corex.p2pServer, networkID)

	// Register the backend on the node
	stack.RegisterAPIs(corex.APIs())
	stack.RegisterProtocols(corex.Protocols())
	stack.RegisterLifecycle(corex)

	// Successful startup; push a marker and check previous unclean shutdowns.
	corex.shutdownTracker.MarkStartup()

	return corex, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"corex",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Production metadata exceeds limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// APIs return the collection of RPC services the corexchain package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *CoreXChain) APIs() []rpc.API {
	apis := corexapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "corex",
			Service:   NewCoreXChainAPI(s),
		}, {
			Namespace: "pov",
			Service:   NewProfileAPI(s),
		}, {
			Namespace: "production",
			Service:   NewProductionAPI(s),
		}, {
			Namespace: "corex",
			Service:   downloader.NewDownloaderAPI(s.handler.downloader, s.blockchain, s.eventMux),
		}, {
			Namespace: "admin",
			Service:   NewAdminAPI(s),
		}, {
			Namespace: "control",
			Service:   NewAdminAPI(s),
		}, {
			Namespace: "debug",
			Service:   NewDebugAPI(s),
		}, {
			Namespace: "communication",
			Service:   s.netRPCService,
		}, {
			Namespace: "net",
			Service:   s.netRPCService,
		},
	}...)
}

func (s *CoreXChain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *CoreXChain) RewardBase() (eb common.Address, err error) {
	s.lock.RLock()
	rewardbase := s.rewardbase
	s.lock.RUnlock()

	if rewardbase != (common.Address{}) {
		return rewardbase, nil
	}
	return common.Address{}, errors.New("rewardbase must be explicitly specified")
}

// isLocalBlock checks whether the specified block is produced
// by local block-settlement accounts.
//
// We regard two types of accounts as local block-settlement accounts: rewardbase
// and accounts specified via `flowspace.locals` flag.
func (s *CoreXChain) isLocalBlock(header *types.Header) bool {
	author, err := s.engine.Author(header)
	if err != nil {
		log.Warn("Failed to retrieve block author", "number", header.Number.Uint64(), "hash", header.Hash(), "err", err)
		return false
	}
	// Check whether the given address is rewardbase.
	s.lock.RLock()
	rewardbase := s.rewardbase
	s.lock.RUnlock()
	if author == rewardbase {
		return true
	}
	// Check whether the given address is specified by `flowspace.local`
	// CLI flag.
	for _, account := range s.config.FlowSpace.Locals {
		if account == author {
			return true
		}
	}
	return false
}

// shouldPreserve checks whether we should preserve the given block
// during the chain reorg depending on whether the author of block
// is a local account.
func (s *CoreXChain) shouldPreserve(header *types.Header) bool {
	// The reason we need to disable the self-reorg preserving for clique
	// is it can be probable to introduce a deadlock.
	//
	// e.g. If there are 7 available signers
	//
	// r1   A
	// r2     B
	// r3       C
	// r4         D
	// r5   A      [X] F G
	// r6    [X]
	//
	// In the round5, the in-turn signer E is offline, so the worst case
	// is A, F and G sign the block of round5 and reject the block of opponents
	// and in the round6, the last available signer B is offline, the whole
	// network is stuck.
	if _, ok := s.engine.(*clique.Clique); ok {
		return false
	}
	return s.isLocalBlock(header)
}

// SetRewardBase sets the block-settlement reward address.
func (s *CoreXChain) SetRewardBase(rewardbase common.Address) {
	s.lock.Lock()
	s.rewardbase = rewardbase
	s.lock.Unlock()

	s.production.SetRewardBase(rewardbase)
}

// StartProduction starts local block production. If block production is already
// running, this method adjusts the number of threads allowed to use and
// updates the minimum fee floor required by the value-flow pool.
func (s *CoreXChain) StartProduction() error {
	// If block production was not running, initialize it.
	if !s.IsProducing() {
		// Propagate the initial fee floor to the value-flow pool.
		s.lock.RLock()
		price := s.feeFloor
		s.lock.RUnlock()
		s.flowSpace.SetGasTip(price)

		// Configure the local block-settlement address.
		eb, err := s.RewardBase()
		if err != nil {
			log.Error("Cannot start block production without rewardbase", "err", err)
			return fmt.Errorf("rewardbase missing: %v", err)
		}
		var cli *clique.Clique
		if c, ok := s.engine.(*clique.Clique); ok {
			cli = c
		} else if cl, ok := s.engine.(*beacon.Beacon); ok {
			if c, ok := cl.InnerEngine().(*clique.Clique); ok {
				cli = c
			}
		}
		if cli != nil {
			wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
			if wallet == nil || err != nil {
				log.Error("RewardBase account unavailable locally", "err", err)
				return fmt.Errorf("signer missing: %v", err)
			}
			cli.Authorize(eb, wallet.SignData)
		}
		// If block production is started, we can disable the value-flow rejection
		// mechanism introduced to speed sync times.
		s.handler.enableSyncedFeatures()

		go s.production.Start()
	}
	return nil
}

// StopProduction terminates local block production, both at the consensus engine
// level as well as at the block creation level.
func (s *CoreXChain) StopProduction() {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		th.SetThreads(-1)
	}
	// Stop the block creating itself
	s.production.Stop()
}

func (s *CoreXChain) IsProducing() bool                    { return s.production.Producing() }
func (s *CoreXChain) Producer() *productionengine.Producer { return s.production }

func (s *CoreXChain) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *CoreXChain) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *CoreXChain) FlowSpace() *flowspace.FlowSpace    { return s.flowSpace }
func (s *CoreXChain) EventMux() *event.TypeMux           { return s.eventMux }
func (s *CoreXChain) Engine() consensus.Engine           { return s.engine }
func (s *CoreXChain) ChainDb() corexdb.Database          { return s.chainDb }
func (s *CoreXChain) IsListening() bool                  { return true } // Always listening
func (s *CoreXChain) Downloader() *downloader.Downloader { return s.handler.downloader }
func (s *CoreXChain) Synced() bool                       { return s.handler.synced.Load() }
func (s *CoreXChain) SetSynced()                         { s.handler.enableSyncedFeatures() }
func (s *CoreXChain) ArchiveMode() bool                  { return s.config.NoPruning }
func (s *CoreXChain) BloomIndexer() *core.ChainIndexer   { return s.bloomIndexer }
func (s *CoreXChain) Merger() *consensus.Merger          { return s.merger }
func (s *CoreXChain) SyncMode() downloader.SyncMode {
	mode, _ := s.handler.chainSync.modeAndLocalHead()
	return mode
}

// Protocols returns all the currently configured
// network protocols to start.
func (s *CoreXChain) Protocols() []p2p.Protocol {
	protos := corex.MakeProtocols((*ethHandler)(s.handler), s.networkID, s.coreDialCandidates)
	if s.config.SnapshotCache > 0 {
		protos = append(protos, snap.MakeProtocols((*snapHandler)(s.handler), s.snapDialCandidates)...)
	}
	return protos
}

// Start implements node.Lifecycle, starting all internal goroutines needed by the
// CoreXChain protocol implementation.
func (s *CoreXChain) Start() error {
	corex.StartENRUpdater(s.blockchain, s.p2pServer.LocalNode())

	// Start the bloom bits servicing goroutines
	s.startBloomHandlers(params.BloomBitsBlocks)

	// Regularly update shutdown marker
	s.shutdownTracker.Start()

	// Figure out a max peers count based on the server limits
	maxPeers := s.p2pServer.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= s.p2pServer.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, s.p2pServer.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.handler.Start(maxPeers)
	return nil
}

// Stop implements node.Lifecycle, terminating all internal goroutines used by the
// CoreXChain protocol.
func (s *CoreXChain) Stop() error {
	// Stop all the peer-related stuff first.
	s.coreDialCandidates.Close()
	s.snapDialCandidates.Close()
	s.handler.Stop()

	// Then stop everything else.
	s.bloomIndexer.Close()
	close(s.closeBloomHandler)
	s.flowSpace.Close()
	s.production.Close()
	s.blockchain.Stop()
	s.engine.Close()

	// Clean shutdown marker as the last thing before closing db
	s.shutdownTracker.Stop()

	s.chainDb.Close()
	s.eventMux.Stop()

	return nil
}
