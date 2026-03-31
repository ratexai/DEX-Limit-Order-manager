package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// SwapVMAddress is the canonical deployment address across all supported chains.
var SwapVMAddress = common.HexToAddress("0x8fdd04dbf6111437b44bbca99c28882434e0958f")

// Order represents a SwapVM order as defined in ISwapVM.sol.
type Order struct {
	Maker  common.Address
	Traits *big.Int // uint256 packed MakerTraits
	Data   []byte   // hook data + program bytecode
}

// ABIEncode returns the ABI-encoded representation of the order
// for use in contract calls (swap, quote, hash).
func (o *Order) ABIEncode() ([]byte, error) {
	return swapVMOrderArgs.Pack(o.Maker, o.Traits, o.Data)
}
