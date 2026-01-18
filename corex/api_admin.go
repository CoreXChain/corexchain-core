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
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"com.corexkey/core"
	"com.corexkey/core/types"
	"com.corexkey/rlp"
)

// AdminAPI is the collection of CoreXChain full node related APIs for node
// administration.
type AdminAPI struct {
	corex *CoreXChain
}

// NewAdminAPI creates a new instance of AdminAPI.
func NewAdminAPI(corex *CoreXChain) *AdminAPI {
	return &AdminAPI{corex: corex}
}

// ExportChain exports the current blockchain into a local file,
// or a range of blocks if first and last are non-nil.
func (api *AdminAPI) ExportChain(file string, first *uint64, last *uint64) (bool, error) {
	if first == nil && last != nil {
		return false, errors.New("last cannot be specified without first")
	}
	if first != nil && last == nil {
		head := api.corex.BlockChain().CurrentHeader().Number.Uint64()
		last = &head
	}
	if _, err := os.Stat(file); err == nil {
		// File already exists. Allowing overwrite could be a DoS vector,
		// since the 'file' may point to arbitrary paths on the drive.
		return false, errors.New("location would overwrite an existing file")
	}
	// Make sure we can create the file to export into
	out, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return false, err
	}
	defer out.Close()

	var writer io.Writer = out
	if strings.HasSuffix(file, ".gz") {
		writer = gzip.NewWriter(writer)
		defer writer.(*gzip.Writer).Close()
	}

	// Export the blockchain
	if first != nil {
		if err := api.corex.BlockChain().ExportN(writer, *first, *last); err != nil {
			return false, err
		}
	} else if err := api.corex.BlockChain().Export(writer); err != nil {
		return false, err
	}
	return true, nil
}

// ExportLedger exports the current CoreXChain ledger archive into a local file.
func (api *AdminAPI) ExportLedger(file string, first *uint64, last *uint64) (bool, error) {
	return api.ExportChain(file, first, last)
}

func hasAllBlocks(chain *core.BlockChain, bs []*types.Block) bool {
	for _, b := range bs {
		if !chain.HasBlock(b.Hash(), b.NumberU64()) {
			return false
		}
	}

	return true
}

// ImportChain imports a blockchain from a local file.
func (api *AdminAPI) ImportChain(file string) (bool, error) {
	// Make sure the can access the file to import
	in, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer in.Close()

	var reader io.Reader = in
	if strings.HasSuffix(file, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return false, err
		}
	}

	// Run actual the import in pre-configured batches
	stream := rlp.NewStream(reader, 0)

	blocks, index := make([]*types.Block, 0, 2500), 0
	for batch := 0; ; batch++ {
		// Load a batch of blocks from the input file
		for len(blocks) < cap(blocks) {
			block := new(types.Block)
			if err := stream.Decode(block); err == io.EOF {
				break
			} else if err != nil {
				return false, fmt.Errorf("block %d: failed to parse: %v", index, err)
			}
			// ignore the genesis block when importing blocks
			if block.NumberU64() == 0 {
				continue
			}
			blocks = append(blocks, block)
			index++
		}
		if len(blocks) == 0 {
			break
		}

		if hasAllBlocks(api.corex.BlockChain(), blocks) {
			blocks = blocks[:0]
			continue
		}
		// Import the batch and reset the buffer
		if _, err := api.corex.BlockChain().InsertChain(blocks); err != nil {
			return false, fmt.Errorf("batch %d: failed to insert: %v", batch, err)
		}
		blocks = blocks[:0]
	}
	return true, nil
}

// ImportLedger imports a CoreXChain ledger archive from a local file.
func (api *AdminAPI) ImportLedger(file string) (bool, error) {
	return api.ImportChain(file)
}
