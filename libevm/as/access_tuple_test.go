package as_test

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/libevm"
	"github.com/ethereum/go-ethereum/libevm/as"
	"github.com/ethereum/go-ethereum/libevm/ethtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func FuzzAccessTupleIdentity(f *testing.F) {
	{
		// Before fuzzing, demonstrate easy-to-catch differences between types.
		// If any of these fail then libevm.Access{List,Tuple} MUST be updated.
		var (
			gethVal   types.AccessTuple
			libevmVal libevm.AccessTuple
			// If the AccessTuple types are identical then so too are the
			// AccessList slices.
			_ []types.AccessTuple  = types.AccessList{}
			_ []libevm.AccessTuple = libevm.AccessList{}
		)

		gethT := reflect.TypeOf(gethVal)
		libevmT := reflect.TypeOf(libevmVal)
		require.Equal(f, gethT.NumField(), libevmT.NumField(), "number of struct fields")

		for i := 0; i < gethT.NumField(); i++ {
			gethFld := gethT.Field(i).Type
			libevmFld := libevmT.Field(i).Type
			assert.Equalf(f, gethFld, libevmFld, "struct field %d reflect.Types", i)
		}
		if f.Failed() {
			return
		}
	}

	for i := uint64(0); i < 10; i++ {
		f.Add(i)
	}
	f.Fuzz(func(t *testing.T, seed uint64) {
		rng := ethtest.NewPseudoRand(seed)
		addr := rng.Address()
		keys := make([]common.Hash, rng.Intn(50))
		for i := range keys {
			keys[i] = rng.Hash()
		}

		gethVal := types.AccessTuple{
			Address:     addr,
			StorageKeys: keys,
		}
		libevmVal := libevm.AccessTuple{
			Address:     addr,
			StorageKeys: keys,
		}

		testConversion(t, libevmVal, gethVal, as.GethAccessTuple)
		testConversion(t, gethVal, libevmVal, as.LibEVMAccessTuple)
		libList := libevm.AccessList{libevmVal, libevmVal}
		gethList := types.AccessList{gethVal, gethVal}
		testConversion(t, libList, gethList, as.GethAccessList)
		testConversion(t, gethList, libList, as.LibEVMAccessList)
	})
}

func testConversion[From any, To any](t *testing.T, from From, want To, conv func(From) To) {
	t.Helper()
	got := conv(from)
	assert.Equalf(t, want, got, "conversion from %T to %T", from, want)
}
