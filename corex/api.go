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

package corex

import (
	"com.corexkey/common"
	"com.corexkey/common/hexutil"
)

// CoreXChainAPI provides an API to access CoreXChain full node-related information.
type CoreXChainAPI struct {
	e *CoreXChain
}

// NewCoreXChainAPI creates a new CoreXChain protocol API for full nodes.
func NewCoreXChainAPI(e *CoreXChain) *CoreXChainAPI {
	return &CoreXChainAPI{e}
}

// RewardBase is the address that block settlement rewards will be sent to.
func (api *CoreXChainAPI) RewardBase() (common.Address, error) {
	return api.e.RewardBase()
}

// Coinbase is the address that block settlement rewards will be sent to (alias for RewardBase).
func (api *CoreXChainAPI) Coinbase() (common.Address, error) {
	return api.RewardBase()
}

// ProductionRate returns the local block production throughput indicator.
func (api *CoreXChainAPI) ProductionRate() hexutil.Uint64 {
	return hexutil.Uint64(api.e.Producer().ProductionRate())
}

// Producing returns an indication if this node is currently producing blocks.
func (api *CoreXChainAPI) Producing() bool {
	return api.e.IsProducing()
}
