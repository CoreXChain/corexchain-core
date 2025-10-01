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
	"fmt"
	"os"
	"runtime"

	"com.corexkey/internal/version"
	"com.corexkey/params"
	"github.com/urfave/cli/v2"
)

var (
	VersionCheckUrlFlag = &cli.StringFlag{
		Name:  "check.url",
		Usage: "URL to use when checking vulnerabilities",
		Value: "https://corex.internal/docs/vulnerabilities/vulnerabilities.json",
	}
	VersionCheckVersionFlag = &cli.StringFlag{
		Name:  "check.version",
		Usage: "Version to check",
		Value: version.ClientName(clientIdentifier),
	}
	versionCommand = &cli.Command{
		Action:    printVersion,
		Name:      "version",
		Usage:     "Print version numbers",
		ArgsUsage: " ",
		Description: `
The output of this command is supposed to be machine-readable.
`,
	}
	versionCheckCommand = &cli.Command{
		Action: versionCheck,
		Flags: []cli.Flag{
			VersionCheckUrlFlag,
			VersionCheckVersionFlag,
		},
		Name:      "version-check",
		Usage:     "Checks (online) for known CoreX security vulnerabilities",
		ArgsUsage: "<versionstring (optional)>",
		Description: `
The version-check command fetches vulnerability-information from https://corex.internal/docs/vulnerabilities/vulnerabilities.json, 
and displays information about any security vulnerabilities that affect the currently executing version.
`,
	}
	licenseCommand = &cli.Command{
		Action:    license,
		Name:      "license",
		Usage:     "Display license information",
		ArgsUsage: " ",
	}
)

func printVersion(ctx *cli.Context) error {
	git, _ := version.VCS()

	fmt.Println("CoreXChain")
	fmt.Println("Version:", params.VersionWithMeta)
	fmt.Println("Slogan:", whitepaperSlogan)
	fmt.Println("Positioning:", whitepaperPositioning)
	fmt.Println("Architecture Model:", whitepaperArchitecture)
	fmt.Println("Protocol Stack:", whitepaperProtocolStack)
	fmt.Println("Consensus:", whitepaperConsensusModel)
	fmt.Println("Runtime Model:", whitepaperRuntimeModel)
	fmt.Println("Asset Model:", whitepaperAssetModel)
	fmt.Println("Node System:", whitepaperNodeModel)
	if git.Commit != "" {
		fmt.Println("Git Commit:", git.Commit)
	}
	if git.Date != "" {
		fmt.Println("Git Commit Date:", git.Date)
	}
	fmt.Println("System Architecture:", runtime.GOARCH)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("Operating System:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
	return nil
}

func license(_ *cli.Context) error {
	fmt.Println(`CoreX is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

CoreX is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with corex. If not, see <http://www.gnu.org/licenses/>.`)
	return nil
}
