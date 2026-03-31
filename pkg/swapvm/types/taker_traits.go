package types

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// TakerTraits flag bits (within the 2-byte flags field).
const (
	FlagIsExactIn                      = 0x0001
	FlagShouldUnwrapWETH               = 0x0002
	FlagHasPreTransferInCallback       = 0x0004
	FlagHasPreTransferOutCallback      = 0x0008
	FlagIsStrictThreshold              = 0x0010
	FlagIsFirstTransferFromTaker       = 0x0020
	FlagUseTransferFromAndAquaPush     = 0x0040
)

// TakerTraitsBuilder constructs the takerTraitsAndData bytes.
type TakerTraitsBuilder struct {
	flags     uint16
	threshold *big.Int        // 0 or 32 bytes
	to        *common.Address // 0 or 20 bytes
	deadline  uint64          // 0 or 5 bytes (uint40)
	signature []byte          // 65 bytes ECDSA
}

// NewTakerTraits creates a builder with sensible defaults for limit orders.
func NewTakerTraits() *TakerTraitsBuilder {
	return &TakerTraitsBuilder{
		flags: FlagIsExactIn | FlagIsFirstTransferFromTaker,
	}
}

// SetExactIn sets IS_EXACT_IN flag.
func (b *TakerTraitsBuilder) SetExactIn(v bool) *TakerTraitsBuilder {
	if v {
		b.flags |= FlagIsExactIn
	} else {
		b.flags &^= FlagIsExactIn
	}
	return b
}

// SetStrictThreshold sets IS_STRICT_THRESHOLD and the threshold value.
func (b *TakerTraitsBuilder) SetStrictThreshold(threshold *big.Int) *TakerTraitsBuilder {
	b.flags |= FlagIsStrictThreshold
	b.threshold = threshold
	return b
}

// SetTo sets the recipient address for tokenOut.
func (b *TakerTraitsBuilder) SetTo(addr common.Address) *TakerTraitsBuilder {
	b.to = &addr
	return b
}

// SetDeadline sets the execution deadline (unix timestamp, uint40).
func (b *TakerTraitsBuilder) SetDeadline(ts uint64) *TakerTraitsBuilder {
	b.deadline = ts
	return b
}

// SetSignature sets the ECDSA signature bytes.
func (b *TakerTraitsBuilder) SetSignature(sig []byte) *TakerTraitsBuilder {
	b.signature = sig
	return b
}

// Build constructs the final takerTraitsAndData bytes.
//
// Wire format:
//   [20 bytes: sliceIndices] [2 bytes: flags] [takerData...]
//
// takerData layout:
//   [Threshold: 0|32 bytes] [To: 0|20 bytes] [Deadline: 0|5 bytes]
//   [HookData sections...] [Signature: remaining]
func (b *TakerTraitsBuilder) Build() []byte {
	var takerData []byte

	// Slice indices track cumulative offsets.
	var indices [10]uint16
	offset := uint16(0)

	// Index 0: Threshold
	if b.threshold != nil {
		threshBytes := make([]byte, 32)
		b.threshold.FillBytes(threshBytes)
		takerData = append(takerData, threshBytes...)
		offset += 32
	}
	indices[0] = offset

	// Index 1: To address
	if b.to != nil {
		takerData = append(takerData, b.to.Bytes()...)
		offset += 20
	}
	indices[1] = offset

	// Index 2: Deadline
	if b.deadline > 0 {
		dlBytes := make([]byte, 5)
		// uint40 big-endian
		dlBytes[0] = byte(b.deadline >> 32)
		dlBytes[1] = byte(b.deadline >> 24)
		dlBytes[2] = byte(b.deadline >> 16)
		dlBytes[3] = byte(b.deadline >> 8)
		dlBytes[4] = byte(b.deadline)
		takerData = append(takerData, dlBytes...)
		offset += 5
	}
	indices[2] = offset

	// Indices 3-8: Hook/callback data (none for simple limit orders)
	for i := 3; i <= 8; i++ {
		indices[i] = offset
	}

	// Index 9: InstructionsArgs (none for simple limit orders)
	indices[9] = offset

	// Signature
	takerData = append(takerData, b.signature...)

	// Encode slice indices as uint160 (10 x uint16, big-endian)
	sliceBytes := make([]byte, 20)
	for i := 0; i < 10; i++ {
		binary.BigEndian.PutUint16(sliceBytes[i*2:], indices[i])
	}

	// Encode flags as uint16 big-endian
	flagBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(flagBytes, b.flags)

	// Final: [sliceIndices (20)] [flags (2)] [takerData]
	result := make([]byte, 0, 22+len(takerData))
	result = append(result, sliceBytes...)
	result = append(result, flagBytes...)
	result = append(result, takerData...)

	return result
}
