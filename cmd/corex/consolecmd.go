// Copyright 2016 CoreX Team
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
	"errors"
	"fmt"

	"com.corexkey/cmd/utils"
	"com.corexkey/internal/flags"
	"github.com/urfave/cli/v2"
)

var (
	consoleFlags = []cli.Flag{utils.JSpathFlag, utils.ExecFlag, utils.PreloadJSFlag}

	consoleCommand = &cli.Command{
		Action: localConsole,
		Name:   "console",
		Usage:  "Open the CoreXChain scripting shell",
		Flags:  flags.Merge(nodeFlags, rpcFlags, consoleFlags),
		Description: `
The scripting shell is not bundled in this CoreXChain distribution.
Use RPC services and configuration files for node interaction.`,
	}

	attachCommand = &cli.Command{
		Action:    remoteConsole,
		Name:      "attach",
		Usage:     "Attach to the CoreXChain scripting shell",
		ArgsUsage: "[endpoint]",
		Flags:     flags.Merge([]cli.Flag{utils.DataDirFlag, utils.HttpHeaderFlag}, consoleFlags),
		Description: `
The scripting shell is not bundled in this CoreXChain distribution.
Use RPC services and configuration files for node interaction.`,
	}

	javascriptCommand = &cli.Command{
		Action:    ephemeralConsole,
		Name:      "js",
		Usage:     "(DEPRECATED) Execute the specified JavaScript files",
		ArgsUsage: "<jsfile> [jsfile...]",
		Flags:     flags.Merge(nodeFlags, consoleFlags),
		Description: `
The embedded scripting shell is no longer included in this CoreXChain distribution.`,
	}
)

func localConsole(ctx *cli.Context) error {
	return errors.New("CoreXChain scripting shell is not included in the current CLI build")
}

func remoteConsole(ctx *cli.Context) error {
	return errors.New("CoreXChain scripting shell is not included in the current CLI build")
}

func ephemeralConsole(ctx *cli.Context) error {
	if ctx.Args().Len() > 0 {
		return fmt.Errorf("the embedded JavaScript shell is not available in the current CoreXChain CLI build")
	}
	return errors.New("the embedded JavaScript shell is not available in the current CoreXChain CLI build")
}
