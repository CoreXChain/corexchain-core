// Copyright 2023 CoreX Team
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
	"math/big"
	"time"

	"com.corexkey/common"
	"com.corexkey/common/hexutil"
)

// ProductionAPI provides an API to control local block production.
type ProductionAPI struct {
	e *CoreXChain
}

// NewProductionAPI creates a new ProductionAPI instance.
func NewProductionAPI(e *CoreXChain) *ProductionAPI {
	return &ProductionAPI{e}
}

// Start starts local block production. If threads is nil,
// the number of workers started is equal to the number of logical CPUs that are
// usable by this process. If block production is already running, this method adjust the
// number of threads allowed to use and updates the minimum price required by the
// flow space.
func (api *ProductionAPI) Start() error {
	return api.e.StartProduction()
}

// Stop terminates local block production, both at the consensus engine level as well as at
// the block creation level.
func (api *ProductionAPI) Stop() {
	api.e.StopProduction()
}

// GetProductionRate returns the local block production throughput indicator.
func (api *ProductionAPI) GetProductionRate() hexutil.Uint64 {
	return hexutil.Uint64(api.e.Producer().ProductionRate())
}

// SetMetadata sets the metadata string that is included when a block is produced.
func (api *ProductionAPI) SetMetadata(metadata string) (bool, error) {
	if err := api.e.Producer().SetExtra([]byte(metadata)); err != nil {
		return false, err
	}
	return true, nil
}

// SetFeeFloor sets the minimum accepted priority fee for block production.
func (api *ProductionAPI) SetFeeFloor(feeFloor hexutil.Big) bool {
	api.e.lock.Lock()
	api.e.feeFloor = (*big.Int)(&feeFloor)
	api.e.lock.Unlock()

	api.e.flowSpace.SetGasTip((*big.Int)(&feeFloor))
	api.e.Producer().SetFeeFloor((*big.Int)(&feeFloor))
	return true
}

// SetBlockGasLimit sets the gaslimit target used during block production.
func (api *ProductionAPI) SetBlockGasLimit(gasLimit hexutil.Uint64) bool {
	api.e.Producer().SetResourceCeil(uint64(gasLimit))
	return true
}

// SetRewardBase sets the rewardbase for local block production.
func (api *ProductionAPI) SetRewardBase(rewardbase common.Address) bool {
	api.e.SetRewardBase(rewardbase)
	return true
}

// SetRecommitInterval updates the interval for block assembly recommitting.
func (api *ProductionAPI) SetRecommitInterval(interval int) {
	api.e.Producer().SetRecommitInterval(time.Duration(interval) * time.Millisecond)
}
