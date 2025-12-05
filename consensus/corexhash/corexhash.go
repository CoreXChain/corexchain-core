// Copyright 2017 CoreX Team
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

// Package corexhash implements the corexhash proof-of-work consensus engine.
package corexhash

import (
	"time"

	"com.corexkey/consensus"
	"com.corexkey/core/types"
	"com.corexkey/rpc"
)

// CoreXHash is a consensus engine based on proof-of-work implementing the corexhash
// algorithm.
type CoreXHash struct {
	fakeFail  *uint64        // Block number which fails PoW check even in fake mode
	fakeDelay *time.Duration // Time delay to sleep for before returning from verify
	fakeFull  bool           // Accepts everything as valid
}

// NewFaker creates an corexhash consensus engine with a fake PoW scheme that accepts
// all blocks' seal as valid, though they still have to conform to the CoreXChain
// consensus rules.
func NewFaker() *CoreXHash {
	return new(CoreXHash)
}

// NewFakeFailer creates a corexhash consensus engine with a fake PoW scheme that
// accepts all blocks as valid apart from the single one specified, though they
// still have to conform to the CoreXChain consensus rules.
func NewFakeFailer(fail uint64) *CoreXHash {
	return &CoreXHash{
		fakeFail: &fail,
	}
}

// NewFakeDelayer creates a corexhash consensus engine with a fake PoW scheme that
// accepts all blocks as valid, but delays verifications by some time, though
// they still have to conform to the CoreXChain consensus rules.
func NewFakeDelayer(delay time.Duration) *CoreXHash {
	return &CoreXHash{
		fakeDelay: &delay,
	}
}

// NewFullFaker creates an corexhash consensus engine with a full fake scheme that
// accepts all blocks as valid, without checking any consensus rules whatsoever.
func NewFullFaker() *CoreXHash {
	return &CoreXHash{
		fakeFull: true,
	}
}

// Close closes the exit channel to notify all backend threads exiting.
func (corexhash *CoreXHash) Close() error {
	return nil
}

// APIs implements consensus.Engine, returning no APIs as corexhash is an empty
// shell in the post-merge world.
func (corexhash *CoreXHash) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{}
}

// Seal generates a new sealing request for the given input block and pushes
// the result into the given channel. For the corexhash engine, this method will
// just panic as sealing is not supported anymore.
func (corexhash *CoreXHash) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	panic("corexhash (pow) sealing not supported any more")
}
