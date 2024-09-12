// Package as converts between native Ethereum types and their libevm
// equivalents.
//
// All functions are named for their returned type, not their input, to improve
// readability at the call site:
//
//	list := as.GethAccessList(...) // list is a geth-native AccessList
//	list := as.LibEVMAccessList(...) // list is a libevm AccessList mirror
//
// Unless stated otherwise, all conversions make deep copies of their inputs.
package as

import (
	"slices"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/libevm"
)

// GethAccessList converts the libevm AccessList to its geth equivalent.
func GethAccessList(l libevm.AccessList) types.AccessList {
	return convertSlice(l, GethAccessTuple)
}

// LibEVMAccessList converts the geth AccessList to its libevm equivalent.
func LibEVMAccessList(l types.AccessList) libevm.AccessList {
	return convertSlice(l, LibEVMAccessTuple)
}

func convertSlice[SIn ~[]EIn, EIn any, EOut any](in SIn, conv func(EIn) EOut) []EOut {
	out := make([]EOut, len(in))
	for i, e := range in {
		out[i] = conv(e)
	}
	return out
}

// NOTE
// ****
// Struct conversions deliberately do NOT use field keys i.f.f. the field types
// are all distinct. This will make them break (by design) if the number of
// fields changes.

// GethAccessTuple converts the libevm AccessTuple to its geth equivalent.
func GethAccessTuple(t libevm.AccessTuple) types.AccessTuple {
	return types.AccessTuple{
		t.Address,
		slices.Clone(t.StorageKeys),
	}
}

// LibEVMAccessTuple converts the geth AccessTuple to its libevm equivalent.
func LibEVMAccessTuple(t types.AccessTuple) libevm.AccessTuple {
	return libevm.AccessTuple{
		t.Address,
		slices.Clone(t.StorageKeys),
	}
}
