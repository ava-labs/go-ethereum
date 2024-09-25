// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package vm

import (
	"fmt"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/libevm"
)

// evmCallArgs mirrors the parameters of the [EVM] methods Call(), CallCode(),
// DelegateCall() and StaticCall(). Its fields are identical to those of the
// parameters, prepended with the receiver name and appended with additional
// values. As {Delegate,Static}Call don't accept a value, they MUST set the
// respective field to nil.
//
// Instantiation can be achieved by merely copying the parameter names, in
// order, which is trivially achieved with AST manipulation:
//
//	func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *uint256.Int) ... {
//	    ...
//	    args := &evmCallArgs{evm, caller, addr, input, gas, value, false}
type evmCallArgs struct {
	evm *EVM
	// args:start
	caller ContractRef
	addr   common.Address
	input  []byte
	gas    uint64
	value  *uint256.Int
	// args:end

	// evm.interpreter.readOnly is only set to true via a call to
	// EVMInterpreter.Run() so, if a precompile is called directly with
	// StaticCall(), then readOnly might not be set yet. StaticCall() MUST set
	// this to forceReadOnly and all other methods MUST set it to
	// inheritReadOnly; i.e. equivalent to the boolean they each pass to
	// EVMInterpreter.Run().
	readWrite rwInheritance
}

type rwInheritance uint8

const (
	inheritReadOnly rwInheritance = iota + 1
	forceReadOnly
)

// run runs the [PrecompiledContract], differentiating between stateful and
// regular types.
func (args *evmCallArgs) run(p PrecompiledContract, input []byte, suppliedGas uint64) (ret []byte, remainingGas uint64, err error) {
	if p, ok := p.(statefulPrecompile); ok {
		return p(args.env(), input, suppliedGas)
	}
	// Gas consumption for regular precompiles was already handled by the native
	// RunPrecompiledContract(), which called this method.
	ret, err = p.Run(input)
	return ret, suppliedGas, err
}

// PrecompiledStatefulContract is the stateful equivalent of a
// [PrecompiledContract].
type PrecompiledStatefulContract func(env Environment, input []byte, suppliedGas uint64) (ret []byte, remainingGas uint64, err error)

// NewStatefulPrecompile constructs a new PrecompiledContract that can be used
// via an [EVM] instance but MUST NOT be called directly; a direct call to Run()
// reserves the right to panic. See other requirements defined in the comments
// on [PrecompiledContract].
func NewStatefulPrecompile(run PrecompiledStatefulContract) PrecompiledContract {
	return statefulPrecompile(run)
}

// statefulPrecompile implements the [PrecompiledContract] interface to allow a
// [PrecompiledStatefulContract] to be carried with regular geth plumbing. The
// methods are defined on this unexported type instead of directly on
// [PrecompiledStatefulContract] to hide implementation details.
type statefulPrecompile PrecompiledStatefulContract

// RequiredGas always returns zero as this gas is consumed by native geth code
// before the contract is run.
func (statefulPrecompile) RequiredGas([]byte) uint64 { return 0 }

func (p statefulPrecompile) Run([]byte) ([]byte, error) {
	// https://google.github.io/styleguide/go/best-practices.html#when-to-panic
	// This would indicate an API misuse and would occur in tests, not in
	// production.
	panic(fmt.Sprintf("BUG: call to %T.Run(); MUST call %T itself", p, p))
}

func (args *evmCallArgs) env() *environment {
	return &environment{
		evm:      args.evm,
		readOnly: args.readOnly(),
		addrs: libevm.AddressContext{
			Origin: args.evm.Origin,
			Caller: args.caller.Address(),
			Self:   args.addr,
		},
	}
}

func (args *evmCallArgs) readOnly() bool {
	if args.readWrite == inheritReadOnly {
		if args.evm.interpreter.readOnly { //nolint:gosimple // Clearer code coverage for difficult-to-test branch
			return true
		}
		return false
	}
	// Even though args.readWrite may be some value other than forceReadOnly,
	// that would be an invalid use of the API so we default to read-only as the
	// safest failure mode.
	return true
}

var (
	// These lock in the assumptions made when implementing [evmCallArgs]. If
	// these break then the struct fields SHOULD be changed to match these
	// signatures.
	_ = [](func(ContractRef, common.Address, []byte, uint64, *uint256.Int) ([]byte, uint64, error)){
		(*EVM)(nil).Call,
		(*EVM)(nil).CallCode,
	}
	_ = [](func(ContractRef, common.Address, []byte, uint64) ([]byte, uint64, error)){
		(*EVM)(nil).DelegateCall,
		(*EVM)(nil).StaticCall,
	}
)
