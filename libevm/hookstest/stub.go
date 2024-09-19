// Copyright 2024 the libevm authors.
//
// The libevm extensions of go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm extensions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

// Package hookstest provides test doubles and convenience wrappers for testing
// libevm hooks.
package hookstest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/libevm"
	"github.com/ethereum/go-ethereum/params"
)

// Register clears any registered [params.Extras] and then registers `extras`
// for the lifetime of the current test, clearing them via tb's
// [testing.TB.Cleanup].
func Register[C params.ChainConfigHooks, R params.RulesHooks](tb testing.TB, extras params.Extras[C, R]) {
	tb.Helper()
	params.TestOnlyClearRegisteredExtras()
	tb.Cleanup(params.TestOnlyClearRegisteredExtras)
	params.RegisterExtras(extras)
}

// A Stub is a test double for [params.ChainConfigHooks] and
// [params.RulesHooks]. Each of the fields, if non-nil, back their respective
// hook methods, which otherwise fall back to the default behaviour.
type Stub struct {
	PrecompileOverrides     map[common.Address]libevm.PrecompiledContract
	CanExecuteTransactionFn func(common.Address, *common.Address, libevm.StateReader) error
	CanCreateContractFn     func(*libevm.AddressContext, uint64, libevm.StateReader) (uint64, error)
}

// Register is a convenience wrapper for registering s as both the
// [params.ChainConfigHooks] and [params.RulesHooks] via [Register].
func (s *Stub) Register(tb testing.TB) {
	tb.Helper()
	Register(tb, params.Extras[*Stub, *Stub]{
		NewRules: func(_ *params.ChainConfig, _ *params.Rules, _ *Stub, blockNum *big.Int, isMerge bool, timestamp uint64) *Stub {
			return s
		},
	})
}

// PrecompileOverride uses the s.PrecompileOverrides map, if non-empty, as the
// canonical source of all overrides. If the map is empty then no precompiles
// are overridden.
func (s Stub) PrecompileOverride(a common.Address) (libevm.PrecompiledContract, bool) {
	if len(s.PrecompileOverrides) == 0 {
		return nil, false
	}
	p, ok := s.PrecompileOverrides[a]
	return p, ok
}

// CanExecuteTransaction proxies arguments to the s.CanExecuteTransactionFn
// function if non-nil, otherwise it acts as a noop.
func (s Stub) CanExecuteTransaction(from common.Address, to *common.Address, sr libevm.StateReader) error {
	if f := s.CanExecuteTransactionFn; f != nil {
		return f(from, to, sr)
	}
	return nil
}

// CanCreateContract proxies arguments to the s.CanCreateContractFn function if
// non-nil, otherwise it acts as a noop.
func (s Stub) CanCreateContract(cc *libevm.AddressContext, gas uint64, sr libevm.StateReader) (uint64, error) {
	if f := s.CanCreateContractFn; f != nil {
		return f(cc, gas, sr)
	}
	return gas, nil
}

var _ interface {
	params.ChainConfigHooks
	params.RulesHooks
} = Stub{}
