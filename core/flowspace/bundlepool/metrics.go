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

package bundlepool

import "com.corexkey/metrics"

var (
	// datacapGauge tracks the user's configured capacity for the bundle pool. It
	// is mostly a way to expose/debug issues.
	datacapGauge = metrics.NewRegisteredGauge("bundlepool/datacap", nil)

	// The below metrics track the per-datastore metrics for the primary blob
	// store and the temporary limbo store.
	datausedGauge = metrics.NewRegisteredGauge("bundlepool/dataused", nil)
	datarealGauge = metrics.NewRegisteredGauge("bundlepool/datareal", nil)
	slotusedGauge = metrics.NewRegisteredGauge("bundlepool/slotused", nil)

	limboDatausedGauge = metrics.NewRegisteredGauge("bundlepool/limbo/dataused", nil)
	limboDatarealGauge = metrics.NewRegisteredGauge("bundlepool/limbo/datareal", nil)
	limboSlotusedGauge = metrics.NewRegisteredGauge("bundlepool/limbo/slotused", nil)

	// The below metrics track the per-shelf metrics for the primary blob store
	// and the temporary limbo store.
	shelfDatausedGaugeName = "bundlepool/shelf_%d/dataused"
	shelfDatagapsGaugeName = "bundlepool/shelf_%d/datagaps"
	shelfSlotusedGaugeName = "bundlepool/shelf_%d/slotused"
	shelfSlotgapsGaugeName = "bundlepool/shelf_%d/slotgaps"

	limboShelfDatausedGaugeName = "bundlepool/limbo/shelf_%d/dataused"
	limboShelfDatagapsGaugeName = "bundlepool/limbo/shelf_%d/datagaps"
	limboShelfSlotusedGaugeName = "bundlepool/limbo/shelf_%d/slotused"
	limboShelfSlotgapsGaugeName = "bundlepool/limbo/shelf_%d/slotgaps"

	// The oversized metrics aggregate the shelf stats above the max blob count
	// limits to track transactions that are just huge, but don't contain blobs.
	//
	// There are no oversized data in the limbo, it only contains blobs and some
	// constant metadata.
	oversizedDatausedGauge = metrics.NewRegisteredGauge("bundlepool/oversized/dataused", nil)
	oversizedDatagapsGauge = metrics.NewRegisteredGauge("bundlepool/oversized/datagaps", nil)
	oversizedSlotusedGauge = metrics.NewRegisteredGauge("bundlepool/oversized/slotused", nil)
	oversizedSlotgapsGauge = metrics.NewRegisteredGauge("bundlepool/oversized/slotgaps", nil)

	// basefeeGauge and blobfeeGauge track the current network 1559 base fee and
	// 4844 bundle fee respectively.
	basefeeGauge = metrics.NewRegisteredGauge("bundlepool/basefee", nil)
	blobfeeGauge = metrics.NewRegisteredGauge("bundlepool/blobfee", nil)

	// pooltipGauge is the configurable miner tip to permit a transaction into
	// the pool.
	pooltipGauge = metrics.NewRegisteredGauge("bundlepool/pooltip", nil)

	// addwait/time, resetwait/time and getwait/time track the rough health of
	// the pool and whether it's capable of keeping up with the load from the
	// network.
	addwaitHist   = metrics.NewRegisteredHistogram("bundlepool/addwait", nil, metrics.NewExpDecaySample(1028, 0.015))
	addtimeHist   = metrics.NewRegisteredHistogram("bundlepool/addtime", nil, metrics.NewExpDecaySample(1028, 0.015))
	getwaitHist   = metrics.NewRegisteredHistogram("bundlepool/getwait", nil, metrics.NewExpDecaySample(1028, 0.015))
	gettimeHist   = metrics.NewRegisteredHistogram("bundlepool/gettime", nil, metrics.NewExpDecaySample(1028, 0.015))
	pendwaitHist  = metrics.NewRegisteredHistogram("bundlepool/pendwait", nil, metrics.NewExpDecaySample(1028, 0.015))
	pendtimeHist  = metrics.NewRegisteredHistogram("bundlepool/pendtime", nil, metrics.NewExpDecaySample(1028, 0.015))
	resetwaitHist = metrics.NewRegisteredHistogram("bundlepool/resetwait", nil, metrics.NewExpDecaySample(1028, 0.015))
	resettimeHist = metrics.NewRegisteredHistogram("bundlepool/resettime", nil, metrics.NewExpDecaySample(1028, 0.015))

	// The below metrics track various cases where transactions are dropped out
	// of the pool. Most are exceptional, some are chain progression and some
	// threshold cappings.
	dropInvalidMeter     = metrics.NewRegisteredMeter("bundlepool/drop/invalid", nil)     // Invalid transaction, consensus change or bugfix, neutral-ish
	dropDanglingMeter    = metrics.NewRegisteredMeter("bundlepool/drop/dangling", nil)    // First nonce gapped, bad
	dropFilledMeter      = metrics.NewRegisteredMeter("bundlepool/drop/filled", nil)      // State full-overlap, chain progress, ok
	dropOverlappedMeter  = metrics.NewRegisteredMeter("bundlepool/drop/overlapped", nil)  // State partial-overlap, chain progress, ok
	dropRepeatedMeter    = metrics.NewRegisteredMeter("bundlepool/drop/repeated", nil)    // Repeated nonce, bad
	dropGappedMeter      = metrics.NewRegisteredMeter("bundlepool/drop/gapped", nil)      // Non-first nonce gapped, bad
	dropOverdraftedMeter = metrics.NewRegisteredMeter("bundlepool/drop/overdrafted", nil) // Balance exceeded, bad
	dropOvercappedMeter  = metrics.NewRegisteredMeter("bundlepool/drop/overcapped", nil)  // Per-account cap exceeded, bad
	dropOverflownMeter   = metrics.NewRegisteredMeter("bundlepool/drop/overflown", nil)   // Global disk cap exceeded, neutral-ish
	dropUnderpricedMeter = metrics.NewRegisteredMeter("bundlepool/drop/underpriced", nil) // Gas tip changed, neutral
	dropReplacedMeter    = metrics.NewRegisteredMeter("bundlepool/drop/replaced", nil)    // Transaction replaced, neutral

	// The below metrics track various outcomes of transactions being added to
	// the pool.
	addInvalidMeter      = metrics.NewRegisteredMeter("bundlepool/add/invalid", nil)      // Invalid transaction, reject, neutral
	addUnderpricedMeter  = metrics.NewRegisteredMeter("bundlepool/add/underpriced", nil)  // Gas tip too low, neutral
	addStaleMeter        = metrics.NewRegisteredMeter("bundlepool/add/stale", nil)        // Nonce already filled, reject, bad-ish
	addGappedMeter       = metrics.NewRegisteredMeter("bundlepool/add/gapped", nil)       // Nonce gapped, reject, bad-ish
	addOverdraftedMeter  = metrics.NewRegisteredMeter("bundlepool/add/overdrafted", nil)  // Balance exceeded, reject, neutral
	addOvercappedMeter   = metrics.NewRegisteredMeter("bundlepool/add/overcapped", nil)   // Per-account cap exceeded, reject, neutral
	addNoreplaceMeter    = metrics.NewRegisteredMeter("bundlepool/add/noreplace", nil)    // Replacement fees or tips too low, neutral
	addNonExclusiveMeter = metrics.NewRegisteredMeter("bundlepool/add/nonexclusive", nil) // Plain transaction from same account exists, reject, neutral
	addValidMeter        = metrics.NewRegisteredMeter("bundlepool/add/valid", nil)        // Valid transaction, add, neutral
)
