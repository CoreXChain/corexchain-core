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
	"fmt"
	"os"

	"com.corexkey/internal/flags"
	"github.com/urfave/cli/v2"
)

const (
	defaultKeyfileName = "keyfile.json"
)

var app *cli.App

func init() {
	app = flags.NewApp("CoreXChain key manager for CXK and CoreX assets")
	app.Description = `Key management entry for the CoreXChain PoV network.
Supports the CXK governance key model and CoreX asset workflows.`
	app.Commands = []*cli.Command{
		commandGenerate,
		commandInspect,
		commandChangePassphrase,
		commandSignMessage,
		commandVerifyMessage,
	}
}

// Commonly used command line flags.
var (
	passphraseFlag = &cli.StringFlag{
		Name:  "passwordfile",
		Usage: "the file that contains the password for the keyfile",
	}
	jsonFlag = &cli.BoolFlag{
		Name:  "json",
		Usage: "output JSON instead of human-readable format",
	}
)

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
