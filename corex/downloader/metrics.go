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

// Contains the metrics collected by the downloader.

package downloader

import (
	"com.corexkey/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("corex/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("corex/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("corex/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("corex/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("corex/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("corex/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("corex/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("corex/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("corex/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("corex/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("corex/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("corex/downloader/receipts/timeout", nil)

	throttleCounter = metrics.NewRegisteredCounter("corex/downloader/throttle", nil)
)
