// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

/*
This test file is part of the blockchain package rather than than the
blockchain_test package so it can bridge access to the internals to properly
test cases which are either not possible or can't reliably be tested via the
public interface.  The functions are only exported while the tests are being
run.
*/

package blockchain

import (
	"sort"
	"time"
)

// TstSetCoinbaseMaturity makes the ability to set the coinbase maturity
// available to the test package.
func TstSetCoinbaseMaturity(maturity int32) {
	coinbaseMaturity = maturity
}

// TstTimeSorter makes the internal timeSorter type available to the test
// package.
func TstTimeSorter(times []time.Time) sort.Interface {
	return timeSorter(times)
}

// TstCheckSerializedHeight makes the internal checkSerializedHeight function
// available to the test package.
var TstCheckSerializedHeight = checkSerializedHeight

// TstSetMaxMedianTimeEntries makes the ability to set the maximum number of
// median tiem entries available to the test package.
func TstSetMaxMedianTimeEntries(val int) {
	maxMedianTimeEntries = val
}

// TstCheckBlockScripts makes the internal checkBlockScripts function available
// to the test package.
var TstCheckBlockScripts = checkBlockScripts
