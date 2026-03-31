package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// MakerTraits bit positions (from high bit).
const (
	BitShouldUnwrapWETH              = 255
	BitUseAquaInsteadOfSignature     = 254
	BitAllowZeroAmountIn             = 253
	BitHasPreTransferInHook          = 252
	BitHasPostTransferInHook         = 251
	BitHasPreTransferOutHook         = 250
	BitHasPostTransferOutHook        = 249
	BitPreTransferInHookHasTarget    = 248
	BitPostTransferInHookHasTarget   = 247
	BitPreTransferOutHookHasTarget   = 246
	BitPostTransferOutHookHasTarget  = 245
	OrderDataSlicesIndexesBitOffset  = 160
)

// MakerTraitsBuilder constructs the uint256 MakerTraits bitfield.
type MakerTraitsBuilder struct {
	value *big.Int
}

// NewMakerTraits creates a new builder with all bits zeroed.
func NewMakerTraits() *MakerTraitsBuilder {
	return &MakerTraitsBuilder{value: new(big.Int)}
}

// SetBit sets a specific bit in the traits.
func (b *MakerTraitsBuilder) SetBit(bit int) *MakerTraitsBuilder {
	b.value.SetBit(b.value, bit, 1)
	return b
}

// SetReceiver sets the receiver address (bits 159-0).
// address(0) means tokens go to the maker.
func (b *MakerTraitsBuilder) SetReceiver(addr common.Address) *MakerTraitsBuilder {
	receiver := new(big.Int).SetBytes(addr.Bytes())
	// Clear bits 159-0
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1))
	b.value.AndNot(b.value, mask)
	// Set receiver
	b.value.Or(b.value, receiver)
	return b
}

// SetUnwrapWETH sets the SHOULD_UNWRAP_WETH flag.
func (b *MakerTraitsBuilder) SetUnwrapWETH() *MakerTraitsBuilder {
	return b.SetBit(BitShouldUnwrapWETH)
}

// Build returns the final uint256 value.
func (b *MakerTraitsBuilder) Build() *big.Int {
	return new(big.Int).Set(b.value)
}

// SimpleLimitTraits returns MakerTraits for a simple limit order
// with no hooks, no unwrap, receiver = maker (default).
func SimpleLimitTraits() *big.Int {
	return NewMakerTraits().Build()
}

// SimpleLimitTraitsWithReceiver returns MakerTraits for a simple limit order
// with a custom receiver address.
func SimpleLimitTraitsWithReceiver(receiver common.Address) *big.Int {
	return NewMakerTraits().SetReceiver(receiver).Build()
}
