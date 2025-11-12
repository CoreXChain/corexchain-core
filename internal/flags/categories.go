// Copyright 2022 CoreX Team
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

package flags

import "github.com/urfave/cli/v2"

const (
	CoreCategory       = "COREXCHAIN"
	LightCategory      = "COMMUNICATION ACCESS"
	DevCategory        = "LAB PROFILE"
	StateCategory      = "STATE AND HISTORY"
	TxPoolCategory     = "VALUE FLOW"
	BundlePoolCategory   = "DATA STREAM"
	PerfCategory       = "PERFORMANCE TUNING"
	AccountCategory    = "ACCOUNT"
	APICategory        = "API AND CONTROL"
	NetworkingCategory = "NETWORKING"
	MinerCategory      = "BLOCK PRODUCTION"
	GasPriceCategory   = "VALUE POLICY"
	VMCategory         = "VIRTUAL MACHINE"
	LoggingCategory    = "LOGGING AND DEBUGGING"
	MetricsCategory    = "OBSERVABILITY"
	MiscCategory       = "MISC"
	TestingCategory    = "TESTING"
	DeprecatedCategory = "ALIASED (deprecated)"
)

func init() {
	cli.HelpFlag.(*cli.BoolFlag).Category = MiscCategory
	cli.VersionFlag.(*cli.BoolFlag).Category = MiscCategory
}
