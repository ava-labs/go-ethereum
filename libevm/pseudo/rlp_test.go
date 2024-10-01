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

package pseudo_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/libevm/ethtest"
	"github.com/ethereum/go-ethereum/libevm/pseudo"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestRLPEquivalence(t *testing.T) {
	t.Parallel()

	for seed := uint64(0); seed < 20; seed++ {
		rng := ethtest.NewPseudoRand(seed)

		t.Run("fuzz", func(t *testing.T) {
			t.Parallel()

			hdr := &types.Header{
				ParentHash:  rng.Hash(),
				UncleHash:   rng.Hash(),
				Coinbase:    rng.Address(),
				Root:        rng.Hash(),
				TxHash:      rng.Hash(),
				ReceiptHash: rng.Hash(),
				Difficulty:  big.NewInt(rng.Int63()),
				Number:      big.NewInt(rng.Int63()),
				GasLimit:    rng.Uint64(),
				GasUsed:     rng.Uint64(),
				Time:        rng.Uint64(),
				Extra:       rng.Bytes(uint(rng.Uint64n(128))),
				MixDigest:   rng.Hash(),
			}
			rng.Read(hdr.Bloom[:])
			rng.Read(hdr.Nonce[:])

			want, err := rlp.EncodeToBytes(hdr)
			require.NoErrorf(t, err, "rlp.EncodeToBytes(%T)", hdr)

			typ := pseudo.From(hdr).Type
			got, err := rlp.EncodeToBytes(typ)
			require.NoErrorf(t, err, "rlp.EncodeToBytes(%T)", typ)

			require.Equalf(t, want, got, "RLP encoding of %T (canonical) vs %T (under test)", hdr, typ)
		})
	}
}
