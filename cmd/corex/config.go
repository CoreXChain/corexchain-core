// Copyright 2017 CoreX Team
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

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"unicode"

	"com.corexkey/accounts"
	"com.corexkey/accounts/external"
	"com.corexkey/accounts/keystore"
	"com.corexkey/accounts/scwallet"
	"com.corexkey/accounts/usbwallet"
	"com.corexkey/cmd/utils"
	"com.corexkey/common"
	"com.corexkey/common/hexutil"
	"com.corexkey/corex/catalyst"
	"com.corexkey/corex/corexconfig"
	"com.corexkey/internal/corexapi"
	"com.corexkey/internal/flags"
	"com.corexkey/internal/version"
	"com.corexkey/log"
	"com.corexkey/node"
	"com.corexkey/params"
	"github.com/naoina/toml"
	"github.com/urfave/cli/v2"
)

var (
	dumpConfigCommand = &cli.Command{
		Action:      dumpConfig,
		Name:        "dumpconfig",
		Usage:       "Export CoreXChain configuration values in TOML format",
		ArgsUsage:   "<dumpfile (optional)>",
		Flags:       flags.Merge(nodeFlags, rpcFlags),
		Description: `Export CoreXChain configuration values in TOML format (to stdout by default).`,
	}

	configFileFlag = &cli.StringFlag{
		Name:     "config",
		Usage:    "TOML configuration file",
		Category: flags.CoreCategory,
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		id := fmt.Sprintf("%s.%s", rt.String(), field)
		if deprecated(id) {
			log.Warn("Config field is deprecated and won't have an effect", "name", id)
			return nil
		}
		var link string
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type corexConfig struct {
	CoreX corexconfig.Config
	Node  node.Config
}

func loadConfig(file string, cfg *corexConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func defaultNodeConfig() node.Config {
	git, _ := version.VCS()
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(git.Commit, git.Date)
	cfg.HTTPModules = appendUniqueModules(cfg.HTTPModules, "corex", "communication", "pov", "trust", "asset", "valueflow", "account", "control", "identity")
	cfg.WSModules = appendUniqueModules(cfg.WSModules, "corex", "communication", "pov", "trust", "asset", "valueflow", "account", "control", "identity")
	cfg.IPCPath = "corex.ipc"
	return cfg
}

func appendUniqueModules(modules []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(modules)+len(extras))
	out := make([]string, 0, len(modules)+len(extras))
	for _, module := range modules {
		if _, ok := seen[module]; ok {
			continue
		}
		seen[module] = struct{}{}
		out = append(out, module)
	}
	for _, module := range extras {
		if _, ok := seen[module]; ok {
			continue
		}
		seen[module] = struct{}{}
		out = append(out, module)
	}
	return out
}

func configProfileComment(cfg corexConfig) string {
	var b strings.Builder
	b.WriteString("# CoreXChain configuration profile\n")
	b.WriteString("# Developer = CoreX Team\n")
	b.WriteString("# Consensus = PoV\n")
	b.WriteString(fmt.Sprintf("# Communication = %s / %s / %s\n", cfg.CoreX.Communication.Protocol, cfg.CoreX.Communication.Discovery, cfg.CoreX.Communication.Connector))
	b.WriteString(fmt.Sprintf("# Governance = %s / %s\n", cfg.CoreX.Governance.Model, cfg.CoreX.Governance.Committee))
	b.WriteString(fmt.Sprintf("# Trust = %s / %s\n", cfg.CoreX.Trust.Model, cfg.CoreX.Trust.Validation))
	b.WriteString(fmt.Sprintf("# Asset = %s / %s\n", cfg.CoreX.Asset.BaseAsset, cfg.CoreX.Asset.SettlementAsset))
	b.WriteString(fmt.Sprintf("# NodeRole = %s\n", cfg.CoreX.NodeRole.Default))
	b.WriteString("# Use `corex dumpgenesis` to export the active genesis profile.\n\n")
	return b.String()
}

// loadBaseConfig loads the corexConfig based on the given command line
// parameters and config file.
func loadBaseConfig(ctx *cli.Context) corexConfig {
	// Load defaults.
	cfg := corexConfig{
		CoreX: corexconfig.Defaults,
		Node:  defaultNodeConfig(),
	}

	// Load config file.
	if file := ctx.String(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	return cfg
}

// makeConfigNode loads corex configuration and creates a blank node instance.
func makeConfigNode(ctx *cli.Context) (*node.Node, corexConfig) {
	cfg := loadBaseConfig(ctx)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	// Node doesn't by default populate account manager backends
	if err := setAccountManagerBackends(stack.Config(), stack.AccountManager(), stack.KeyStoreDir()); err != nil {
		utils.Fatalf("Failed to set account manager backends: %v", err)
	}

	utils.SetCoreConfig(ctx, stack, &cfg.CoreX)

	return stack, cfg
}

// makeFullNode loads corex configuration and creates the CoreXChain backend.
func makeFullNode(ctx *cli.Context) (*node.Node, corexapi.Backend) {
	stack, cfg := makeConfigNode(ctx)
	if ctx.IsSet(utils.OverrideCancun.Name) {
		v := ctx.Uint64(utils.OverrideCancun.Name)
		cfg.CoreX.OverrideCancun = &v
	}
	if ctx.IsSet(utils.OverrideVerkle.Name) {
		v := ctx.Uint64(utils.OverrideVerkle.Name)
		cfg.CoreX.OverrideVerkle = &v
	}
	backend, corex := utils.RegisterEthService(stack, &cfg.CoreX)

	// Configure log filter RPC API.
	filterSystem := utils.RegisterFilterAPI(stack, backend, &cfg.CoreX)
	_ = filterSystem
	// Configure full-sync tester service if requested
	if ctx.IsSet(utils.SyncTargetFlag.Name) {
		hex := hexutil.MustDecode(ctx.String(utils.SyncTargetFlag.Name))
		if len(hex) != common.HashLength {
			utils.Fatalf("invalid sync target length: have %d, want %d", len(hex), common.HashLength)
		}
		utils.RegisterFullSyncTester(stack, corex, common.BytesToHash(hex))
	}
	// Start the dev mode if requested, or launch the engine API for
	// interacting with external consensus client.
	if ctx.IsSet(utils.DeveloperFlag.Name) {
		simBeacon, err := catalyst.NewSimulatedBeacon(ctx.Uint64(utils.DeveloperPeriodFlag.Name), corex)
		if err != nil {
			utils.Fatalf("failed to register dev mode catalyst service: %v", err)
		}
		catalyst.RegisterSimulatedBeaconAPIs(stack, simBeacon)
		stack.RegisterLifecycle(simBeacon)
	} else {
		err := catalyst.Register(stack, corex)
		if err != nil {
			utils.Fatalf("failed to register catalyst service: %v", err)
		}
	}
	return stack, backend
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := configProfileComment(cfg)

	if cfg.CoreX.Genesis != nil {
		cfg.CoreX.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		dump, err = os.OpenFile(ctx.Args().Get(0), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer dump.Close()
	}
	dump.WriteString(comment)
	dump.Write(out)

	return nil
}

func deprecated(field string) bool {
	switch field {
	case "corexconfig.Config.EVMInterpreter":
		return true
	case "corexconfig.Config.EWASMInterpreter":
		return true
	case "corexconfig.Config.TrieCleanCacheJournal":
		return true
	case "corexconfig.Config.TrieCleanCacheRejournal":
		return true
	default:
		return false
	}
}

func setAccountManagerBackends(conf *node.Config, am *accounts.Manager, keydir string) error {
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if conf.UseLightweightKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}

	// Assemble the supported backends
	if len(conf.ExternalSigner) > 0 {
		log.Info("Using external signer", "url", conf.ExternalSigner)
		if extBackend, err := external.NewExternalBackend(conf.ExternalSigner); err == nil {
			am.AddBackend(extBackend)
			return nil
		} else {
			return fmt.Errorf("error connecting to external signer: %v", err)
		}
	}

	// For now, we're using EITHER external signer OR local signers.
	// If/when we implement some form of lockfile for USB and keystore wallets,
	// we can have both, but it's very confusing for the user to see the same
	// accounts in both externally and locally, plus very racey.
	am.AddBackend(keystore.NewKeyStore(keydir, scryptN, scryptP))
	if conf.USB {
		// Start a USB hub for Ledger hardware wallets
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start Ledger hub, disabling: %v", err))
		} else {
			am.AddBackend(ledgerhub)
		}
		// Start a USB hub for Trezor hardware wallets (HID version)
		if trezorhub, err := usbwallet.NewTrezorHubWithHID(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start HID Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
		// Start a USB hub for Trezor hardware wallets (WebUSB version)
		if trezorhub, err := usbwallet.NewTrezorHubWithWebUSB(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start WebUSB Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
	}
	if len(conf.SmartCardDaemonPath) > 0 {
		// Start a smart card hub
		if schub, err := scwallet.NewHub(conf.SmartCardDaemonPath, scwallet.Scheme, keydir); err != nil {
			log.Warn(fmt.Sprintf("Failed to start smart card hub, disabling: %v", err))
		} else {
			am.AddBackend(schub)
		}
	}

	return nil
}
