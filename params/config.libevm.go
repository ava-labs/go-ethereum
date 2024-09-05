package params

import (
	"encoding/json"
	"fmt"
	"math/big"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/libevm/pseudo"
)

// Extras are arbitrary payloads to be added as extra fields in [ChainConfig]
// and [Rules] structs. See [RegisterExtras].
type Extras[C ChainConfigHooks, R RulesHooks] struct {
	// NewRules, if non-nil is called at the end of [ChainConfig.Rules] with the
	// newly created [Rules] and other context from the method call. Its
	// returned value will be the extra payload of the [Rules]. If NewRules is
	// nil then so too will the [Rules] extra payload be a nil `*R`.
	//
	// NewRules MAY modify the [Rules] but MUST NOT modify the [ChainConfig].
	NewRules func(_ *ChainConfig, _ *Rules, _ *C, blockNum *big.Int, isMerge bool, timestamp uint64) *R
}

// RegisterExtras registers the types `C` and `R` such that they are carried as
// extra payloads in [ChainConfig] and [Rules] structs, respectively. It is
// expected to be called in an `init()` function and MUST NOT be called more
// than once. Both `C` and `R` MUST be structs.
//
// After registration, JSON unmarshalling of a [ChainConfig] will create a new
// `*C` and unmarshal the JSON key "extra" into it. Conversely, JSON marshalling
// will populate the "extra" key with the contents of the `*C`. Both the
// [json.Marshaler] and [json.Unmarshaler] interfaces are honoured if
// implemented by `C` and/or `R.`
//
// Calls to [ChainConfig.Rules] will call the `NewRules` function of the
// registered [Extras] to create a new `*R`.
//
// The payloads can be accessed via the [ExtraPayloadGetter.FromChainConfig] and
// [ExtraPayloadGetter.FromRules] methods of the getter returned by
// RegisterExtras. Where stated in the interface definitions, they will also be
// used as hooks to alter Ethereum behaviour; if this isn't desired then they
// can embed [NOOPHooks] to satisfy either interface.
func RegisterExtras[C ChainConfigHooks, R RulesHooks](e Extras[C, R]) ExtraPayloadGetter[C, R] {
	if registeredExtras != nil {
		panic("re-registration of Extras")
	}
	mustBeStruct[C]()
	mustBeStruct[R]()

	getter := e.getter()
	registeredExtras = &extraConstructors{
		chainConfig: pseudo.NewConstructor[C](),
		rules:       pseudo.NewConstructor[R](),
		newForRules: e.newForRules,
		getter:      getter,
	}
	return getter
}

// TestOnlyClearRegisteredExtras clears the [Extras] previously passed to
// [RegisterExtras]. It panics if called from a non-testing call stack.
//
// In tests it SHOULD be called before every call to [RegisterExtras] and then
// defer-called afterwards, either directly or via testing.TB.Cleanup(). This is
// a workaround for the single-call limitation on [RegisterExtras].
func TestOnlyClearRegisteredExtras() {
	pc := make([]uintptr, 10)
	runtime.Callers(0, pc)
	frames := runtime.CallersFrames(pc)
	for {
		f, more := frames.Next()
		if strings.Contains(f.File, "/testing/") || strings.HasSuffix(f.File, "_test.go") {
			registeredExtras = nil
			return
		}
		if !more {
			panic("no _test.go file in call stack")
		}
	}
}

// registeredExtras holds non-generic constructors for the [Extras] types
// registered via [RegisterExtras].
var registeredExtras *extraConstructors

type extraConstructors struct {
	chainConfig, rules pseudo.Constructor
	newForRules        func(_ *ChainConfig, _ *Rules, blockNum *big.Int, isMerge bool, timestamp uint64) *pseudo.Type
	// use top-level hooksFrom<X>() functions instead of these as they handle
	// instances where no [Extras] were registered.
	getter interface {
		hooksFromChainConfig(*ChainConfig) ChainConfigHooks
		hooksFromRules(*Rules) RulesHooks
	}
}

func (e *Extras[C, R]) newForRules(c *ChainConfig, r *Rules, blockNum *big.Int, isMerge bool, timestamp uint64) *pseudo.Type {
	if e.NewRules == nil {
		return registeredExtras.rules.NilPointer()
	}
	rExtra := e.NewRules(c, r, e.getter().FromChainConfig(c), blockNum, isMerge, timestamp)
	return pseudo.From(rExtra).Type
}

func (*Extras[C, R]) getter() (g ExtraPayloadGetter[C, R]) { return }

// mustBeStruct panics if `T` isn't a struct.
func mustBeStruct[T any]() {
	// XXX: Seems this is a new go-lang feature?
	// if k := reflect.TypeFor[T]().Kind(); k != reflect.Struct {
	// 	panic(notStructMessage[T]())
	// }
}

// notStructMessage returns the message with which [mustBeStruct] might panic.
// It exists to avoid change-detector tests should the message contents change.
func notStructMessage[T any]() string {
	var x T
	return fmt.Sprintf("%T is not a struct", x)
}

// An ExtraPayloadGettter provides strongly typed access to the extra payloads
// carried by [ChainConfig] and [Rules] structs. The only valid way to construct
// a getter is by a call to [RegisterExtras].
type ExtraPayloadGetter[C ChainConfigHooks, R RulesHooks] struct {
	_ struct{} // make godoc show unexported fields so nobody tries to make their own getter ;)
}

// FromChainConfig returns the ChainConfig's extra payload.
func (ExtraPayloadGetter[C, R]) FromChainConfig(c *ChainConfig) *C {
	return pseudo.MustNewValue[*C](c.extraPayload()).Get()
}

// hooksFromChainConfig is equivalent to FromChainConfig(), but returns an
// interface instead of the concrete type implementing it; this allows it to be
// used in non-generic code. If the concrete-type value is nil (typically
// because no [Extras] were registered) a [noopHooks] is returned so it can be
// used without nil checks.
func (e ExtraPayloadGetter[C, R]) hooksFromChainConfig(c *ChainConfig) ChainConfigHooks {
	if h := e.FromChainConfig(c); h != nil {
		return *h
	}
	return NOOPHooks{}
}

// FromRules returns the Rules' extra payload.
func (ExtraPayloadGetter[C, R]) FromRules(r *Rules) *R {
	return pseudo.MustNewValue[*R](r.extraPayload()).Get()
}

// hooksFromRules is the [RulesHooks] equivalent of hooksFromChainConfig().
func (e ExtraPayloadGetter[C, R]) hooksFromRules(r *Rules) RulesHooks {
	if h := e.FromRules(r); h != nil {
		return *h
	}
	return NOOPHooks{}
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (c *ChainConfig) UnmarshalJSON(data []byte) error {
	type raw ChainConfig // doesn't inherit methods so avoids recursing back here (infinitely)
	cc := &struct {
		*raw
		Extra *pseudo.Type `json:"extra"`
	}{
		raw: (*raw)(c), // embedded to achieve regular JSON unmarshalling
	}
	if e := registeredExtras; e != nil {
		cc.Extra = e.chainConfig.NilPointer() // `c.extra` is otherwise unexported
	}

	if err := json.Unmarshal(data, cc); err != nil {
		return err
	}
	c.extra = cc.Extra
	return nil
}

// MarshalJSON implements the [json.Marshaler] interface.
func (c *ChainConfig) MarshalJSON() ([]byte, error) {
	// See UnmarshalJSON() for rationale.
	type raw ChainConfig
	cc := &struct {
		*raw
		Extra *pseudo.Type `json:"extra"`
	}{raw: (*raw)(c), Extra: c.extra}
	return json.Marshal(cc)
}

var _ interface {
	json.Marshaler
	json.Unmarshaler
} = (*ChainConfig)(nil)

// addRulesExtra is called at the end of [ChainConfig.Rules]; it exists to
// abstract the libevm-specific behaviour outside of original geth code.
func (c *ChainConfig) addRulesExtra(r *Rules, blockNum *big.Int, isMerge bool, timestamp uint64) {
	r.extra = nil
	if registeredExtras != nil {
		r.extra = registeredExtras.newForRules(c, r, blockNum, isMerge, timestamp)
	}
}

// extraPayload returns the ChainConfig's extra payload iff [RegisterExtras] has
// already been called. If the payload hasn't been populated (typically via
// unmarshalling of JSON), a nil value is constructed and returned.
func (c *ChainConfig) extraPayload() *pseudo.Type {
	if registeredExtras == nil {
		// This will only happen if someone constructs an [ExtraPayloadGetter]
		// directly, without a call to [RegisterExtras].
		//
		// See https://google.github.io/styleguide/go/best-practices#when-to-panic
		panic(fmt.Sprintf("%T.ExtraPayload() called before RegisterExtras()", c))
	}
	if c.extra == nil {
		c.extra = registeredExtras.chainConfig.NilPointer()
	}
	return c.extra
}

// extraPayload is equivalent to [ChainConfig.extraPayload].
func (r *Rules) extraPayload() *pseudo.Type {
	if registeredExtras == nil {
		// See ChainConfig.extraPayload() equivalent.
		panic(fmt.Sprintf("%T.ExtraPayload() called before RegisterExtras()", r))
	}
	if r.extra == nil {
		r.extra = registeredExtras.rules.NilPointer()
	}
	return r.extra
}
