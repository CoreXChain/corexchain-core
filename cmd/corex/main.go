// Copyright 2014 CoreX Team
// This file is part of corex-chain.
//
// corex-chain is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// corex-chain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with corex-chain. If not, see <http://www.gnu.org/licenses/>.

// corex is a command-line client for CoreXChain.
package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"com.corexkey/accounts"
	"com.corexkey/accounts/keystore"
	"com.corexkey/cmd/utils"
	"com.corexkey/common"
	"com.corexkey/corex"
	"com.corexkey/corex/downloader"
	"com.corexkey/internal/corexapi"
	"com.corexkey/internal/debug"
	"com.corexkey/internal/flags"
	"com.corexkey/log"
	"com.corexkey/node"
	"go.uber.org/automaxprocs/maxprocs"

	// Force-load the tracer engines to trigger registration
	_ "com.corexkey/corex/tracers/js"
	_ "com.corexkey/corex/tracers/native"

	"github.com/urfave/cli/v2"
)

const (
	clientIdentifier         = "corex" // Client identifier to advertise over the network
	whitepaperSlogan         = "Contribution and value, this is the core."
	whitepaperPositioning    = "Decentralized communication and on-chain governance"
	whitepaperArchitecture   = "Communication / Governance / Trust"
	whitepaperProtocolStack  = "CWW protocol + UTXO Connector"
	whitepaperConsensusModel = "PoV + Zero-Knowledge Proof"
	whitepaperRuntimeModel   = "Stream-based scheduling with staged governance and trust-layer settlement"
	whitepaperAssetModel     = "CXK governance key + CoreX mainnet asset with a dedicated conversion path"
	whitepaperNodeModel      = "communication / backbone / governance / trust / sync committee"
)

var (
	// flags that configure the node
	nodeFlags = flags.Merge([]cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.BootnodesFlag,
		utils.MinFreeDiskSpaceFlag,
		utils.KeyStoreDirFlag,
		utils.ExternalSignerFlag,
		utils.NoUSBFlag, // deprecated
		utils.USBFlag,
		utils.SmartCardDaemonPathFlag,
		utils.OverrideCancun,
		utils.OverrideVerkle,
		utils.EnableLegacyAccountFlag,
		utils.TxPoolLocalsFlag,
		utils.TxPoolNoLocalsFlag,
		utils.TxPoolJournalFlag,
		utils.TxPoolRejournalFlag,
		utils.TxPoolPriceLimitFlag,
		utils.TxPoolPriceBumpFlag,
		utils.TxPoolAccountSlotsFlag,
		utils.TxPoolGlobalSlotsFlag,
		utils.TxPoolAccountQueueFlag,
		utils.TxPoolGlobalQueueFlag,
		utils.TxPoolLifetimeFlag,
		utils.BundlePoolDataDirFlag,
		utils.BundlePoolDataCapFlag,
		utils.BundlePoolPriceBumpFlag,
		utils.SyncModeFlag,
		utils.SyncTargetFlag,
		utils.ExitWhenSyncedFlag,
		utils.GCModeFlag,
		utils.SnapshotFlag,
		utils.TxLookupLimitFlag, // deprecated
		utils.TransactionHistoryFlag,
		utils.StateHistoryFlag,
		utils.LightServeFlag,    // deprecated
		utils.LightIngressFlag,  // deprecated
		utils.LightEgressFlag,   // deprecated
		utils.LightMaxPeersFlag, // deprecated
		utils.LightNoPruneFlag,  // deprecated
		utils.LightKDFFlag,
		utils.LightNoSyncServeFlag, // deprecated
		utils.CoreRequiredBlocksFlag,
		utils.LegacyWhitelistFlag, // deprecated
		utils.BloomFilterSizeFlag,
		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheTrieFlag,
		utils.CacheTrieJournalFlag,   // deprecated
		utils.CacheTrieRejournalFlag, // deprecated
		utils.CacheGCFlag,
		utils.CacheSnapshotFlag,
		utils.CacheNoPrefetchFlag,
		utils.CachePreimagesFlag,
		utils.CacheLogSizeFlag,
		utils.FDLimitFlag,
		utils.CryptoKZGFlag,
		utils.ListenPortFlag,
		utils.DiscoveryPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.ProductionEnabledFlag,
		utils.ProductionGasLimitFlag,
		utils.ProductionFeeFloorFlag,
		utils.ProductionRewardBaseFlag,
		utils.ProductionMetadataFlag,
		utils.ProductionRecommitIntervalFlag,
		utils.ProductionNewPayloadTimeoutFlag,
		utils.NATFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV4Flag,
		utils.DiscoveryV5Flag,
		utils.LegacyDiscoveryV5Flag, // deprecated
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.DNSDiscoveryFlag,
		utils.DeveloperFlag,
		utils.DeveloperGasLimitFlag,
		utils.DeveloperPeriodFlag,
		utils.VMEnableDebugFlag,
		utils.NetworkIdFlag,
		utils.NoCompactionFlag,
		utils.ValuePolicyBlocksFlag,
		utils.ValuePolicyPercentileFlag,
		utils.ValuePolicyMaxPriceFlag,
		utils.ValuePolicyIgnorePriceFlag,
		configFileFlag,
		utils.LogDebugFlag,
		utils.LogBacktraceAtFlag,
	}, utils.NetworkFlags, utils.DatabaseFlags)

	rpcFlags = []cli.Flag{
		utils.HTTPEnabledFlag,
		utils.HTTPListenAddrFlag,
		utils.HTTPPortFlag,
		utils.HTTPCORSDomainFlag,
		utils.AuthListenFlag,
		utils.AuthPortFlag,
		utils.AuthVirtualHostsFlag,
		utils.JWTSecretFlag,
		utils.HTTPVirtualHostsFlag,
		utils.HTTPApiFlag,
		utils.HTTPPathPrefixFlag,
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
		utils.WSPathPrefixFlag,
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
		utils.InsecureUnlockAllowedFlag,
		utils.RPCGlobalGasCapFlag,
		utils.RPCGlobalEVMTimeoutFlag,
		utils.RPCGlobalTxFeeCapFlag,
		utils.AllowUnprotectedTxs,
		utils.BatchRequestLimit,
		utils.BatchResponseMaxSize,
	}
)

var app = flags.NewApp("CoreXChain PoV client for decentralized communication and on-chain governance")

func init() {
	// Initialize the CLI app and start CoreX
	app.Description = `CoreXChain integrates decentralized communication, governance, and trust coordination.
Slogan: Contribution and value, this is the core.
Architecture: Communication / Governance / Trust
Protocol stack: CWW protocol + UTXO Connector
Consensus: PoV + Zero-Knowledge Proof`
	app.Action = corex
	app.Commands = []*cli.Command{
		// See chaincmd.go:
		initCommand,
		importCommand,
		exportCommand,
		importHistoryCommand,
		exportHistoryCommand,
		importPreimagesCommand,
		removedbCommand,
		dumpCommand,
		dumpGenesisCommand,
		// See accountcmd.go:
		accountCommand,
		walletCommand,
		// See consolecmd.go:
		consoleCommand,
		attachCommand,
		javascriptCommand,
		// See misccmd.go:
		versionCommand,
		versionCheckCommand,
		licenseCommand,
		// See config.go
		dumpConfigCommand,
		// see dbcmd.go
		dbCommand,
		// See cmd/utils/flags_history.go
		utils.ShowDeprecated,
		// See snapshot.go
		snapshotCommand,
		// See verkle.go
		verkleCommand,
	}
	if logTestCommand != nil {
		app.Commands = append(app.Commands, logTestCommand)
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = flags.Merge(
		nodeFlags,
		rpcFlags,
		consoleFlags,
		debug.Flags,
	)
	flags.AutoEnvVars(app.Flags, "COREX")

	app.Before = func(ctx *cli.Context) error {
		maxprocs.Set() // Automatically set GOMAXPROCS to match Linux container CPU quota.
		flags.MigrateGlobalFlags(ctx)
		if err := debug.Setup(ctx); err != nil {
			return err
		}
		flags.CheckEnvVars(ctx, app.Flags, "COREX")
		return nil
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// prepare manipulates memory cache allowance and setups metric system.
// This function should be called before launching devp2p stack.
func prepare(ctx *cli.Context) {
	// If we're running a known preset, log it for convenience.
	switch {
	case ctx.IsSet(utils.LabNetFlag.Name):
		log.Info("Starting CoreX on LabNet communication preset...")

	case ctx.IsSet(utils.TestNetFlag.Name):
		log.Info("Starting CoreX on TestNet governance preset...")

	case ctx.IsSet(utils.StagingNetFlag.Name):
		log.Info("Starting CoreX on StagingNet trust preset...")

	case ctx.IsSet(utils.DeveloperFlag.Name):
		log.Info("Starting CoreX in communication sandbox mode...")
		log.Warn(`You are running CoreX in --dev mode. Please note the following:

  1. This mode is intended for fast iteration on communication, governance and trust flows.
  2. The database is created in memory unless specified otherwise. Therefore, shutting down
     your computer or losing power will wipe your entire block data and chain state for
     your dev environment.
  3. A random, pre-allocated developer account will be available and unlocked as
     corex.coinbase, which can be used for testing. The random dev account is temporary,
     stored on a ramdisk, and will be lost if your machine is restarted.
  4. Local block production is enabled by default and only progresses when transactions
     are pending in the mempool.
  5. Networking is disabled; there is no listen-address, the maximum number of peers is set
     to 0, and discovery is disabled.
`)

	case !ctx.IsSet(utils.NetworkIdFlag.Name):
		log.Info("Starting CoreX on CoreXChain corenet...")
	}
	// If we're a full node on corenet without --cache specified, bump default cache allowance
	if !ctx.IsSet(utils.CacheFlag.Name) && !ctx.IsSet(utils.NetworkIdFlag.Name) {
		// Make sure we're not on any supported preconfigured testnet either
		if !ctx.IsSet(utils.StagingNetFlag.Name) &&
			!ctx.IsSet(utils.TestNetFlag.Name) &&
			!ctx.IsSet(utils.LabNetFlag.Name) &&
			!ctx.IsSet(utils.DeveloperFlag.Name) {
			// Nope, we're really on corenet. Bump that cache up!
			log.Info("Bumping default cache on corenet", "provided", ctx.Int(utils.CacheFlag.Name), "updated", 4096)
			ctx.Set(utils.CacheFlag.Name, strconv.Itoa(4096))
		}
	}
}

// corex is the main entry point into the system if no special subcommand is run.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func corex(ctx *cli.Context) error {
	if args := ctx.Args().Slice(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	prepare(ctx)
	stack, backend := makeFullNode(ctx)
	defer stack.Close()

	startNode(ctx, stack, backend, false)
	stack.Wait()
	return nil
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and local
// block production.
func startNode(ctx *cli.Context, stack *node.Node, backend corexapi.Backend, isConsole bool) {
	debug.Memsize.Add("node", stack)

	// Start up the node itself
	utils.StartNode(ctx, stack, isConsole)

	// Unlock any account specifically requested
	unlockAccounts(ctx, stack)

	// Register wallet event handlers to open and auto-derive wallets
	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	go func() {
		// Open any wallets already attached
		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			}
		}
		// Listen for wallet event till termination
		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}
			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()
				log.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

				var derivationPaths []accounts.DerivationPath
				if event.Wallet.URL().Scheme == "ledger" {
					derivationPaths = append(derivationPaths, accounts.LegacyLedgerBaseDerivationPath)
				}
				derivationPaths = append(derivationPaths, accounts.DefaultBaseDerivationPath)

				event.Wallet.SelfDerive(derivationPaths, nil)

			case accounts.WalletDropped:
				log.Info("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()

	// Spawn a standalone goroutine for status synchronization monitoring,
	// close the node when synchronization is complete if user required.
	if ctx.Bool(utils.ExitWhenSyncedFlag.Name) {
		go func() {
			sub := stack.EventMux().Subscribe(downloader.DoneEvent{})
			defer sub.Unsubscribe()
			for {
				event := <-sub.Chan()
				if event == nil {
					continue
				}
				done, ok := event.Data.(downloader.DoneEvent)
				if !ok {
					continue
				}
				if timestamp := time.Unix(int64(done.Latest.Time), 0); time.Since(timestamp) < 10*time.Minute {
					log.Info("Synchronisation completed", "latestnum", done.Latest.Number, "latesthash", done.Latest.Hash(),
						"age", common.PrettyAge(timestamp))
					stack.Close()
				}
			}
		}()
	}

	// Start auxiliary services if enabled
	if ctx.Bool(utils.ProductionEnabledFlag.Name) {
		// Block production only makes sense if a full CoreXChain node is running.
		if ctx.String(utils.SyncModeFlag.Name) == "light" {
			utils.Fatalf("Light clients do not support block production")
		}
		corexBackend, ok := backend.(*corex.CoreXAPIBackend)
		if !ok {
			utils.Fatalf("CoreXChain service not running")
		}
		// Set the fee floor from the CLI and start block production.
		feeFloor := flags.GlobalBig(ctx, utils.ProductionFeeFloorFlag.Name)
		corexBackend.FlowSpace().SetGasTip(feeFloor)
		if err := corexBackend.StartProduction(); err != nil {
			utils.Fatalf("Failed to start block production: %v", err)
		}
	}
}

// unlockAccounts unlocks any account specifically requested.
func unlockAccounts(ctx *cli.Context, stack *node.Node) {
	var unlocks []string
	inputs := strings.Split(ctx.String(utils.UnlockedAccountFlag.Name), ",")
	for _, input := range inputs {
		if trimmed := strings.TrimSpace(input); trimmed != "" {
			unlocks = append(unlocks, trimmed)
		}
	}
	// Short circuit if there is no account to unlock.
	if len(unlocks) == 0 {
		return
	}
	// If insecure account unlocking is not allowed if node's APIs are exposed to external.
	// Print warning log to user and skip unlocking.
	if !stack.Config().InsecureUnlockAllowed && stack.Config().ExtRPCEnabled() {
		utils.Fatalf("Account unlock with HTTP access is forbidden!")
	}
	backends := stack.AccountManager().Backends(keystore.KeyStoreType)
	if len(backends) == 0 {
		log.Warn("Failed to unlock accounts, keystore is not available")
		return
	}
	ks := backends[0].(*keystore.KeyStore)
	passwords := utils.MakePasswordList(ctx)
	for i, account := range unlocks {
		unlockAccount(ks, account, i, passwords)
	}
}
