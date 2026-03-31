package program

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Builder constructs SwapVM bytecode programs using a fluent API.
//
// Each instruction is encoded as:
//   [1 byte: opcode] [1 byte: argsLength] [argsLength bytes: args]
type Builder struct {
	buf bytes.Buffer
}

// New creates a new program builder.
func New() *Builder {
	return &Builder{}
}

// emit writes a single instruction to the program.
func (b *Builder) emit(opcode byte, args []byte) *Builder {
	b.buf.WriteByte(opcode)
	b.buf.WriteByte(byte(len(args)))
	b.buf.Write(args)
	return b
}

// Salt adds a salt instruction for order hash uniqueness.
// The salt value is arbitrary bytes that don't affect execution.
func (b *Builder) Salt(salt []byte) *Builder {
	return b.emit(OpSalt, salt)
}

// Deadline adds a deadline check. Reverts if block.timestamp > ts.
func (b *Builder) Deadline(ts uint64) *Builder {
	args := make([]byte, 5)
	args[0] = byte(ts >> 32)
	args[1] = byte(ts >> 24)
	args[2] = byte(ts >> 16)
	args[3] = byte(ts >> 8)
	args[4] = byte(ts)
	return b.emit(OpDeadline, args)
}

// InvalidateBit adds a one-time invalidation guard using a bit index.
func (b *Builder) InvalidateBit(bitIndex uint32) *Builder {
	args := make([]byte, 4)
	binary.BigEndian.PutUint32(args, bitIndex)
	return b.emit(OpInvalidateBit1D, args)
}

// InvalidateTokenIn adds partial-fill tracking by tokenIn amount.
// Must be placed AFTER the swap instruction.
func (b *Builder) InvalidateTokenIn() *Builder {
	return b.emit(OpInvalidateTokenIn1D, nil)
}

// InvalidateTokenOut adds partial-fill tracking by tokenOut amount.
// Must be placed AFTER the swap instruction.
func (b *Builder) InvalidateTokenOut() *Builder {
	return b.emit(OpInvalidateTokenOut1D, nil)
}

// StaticBalances sets the exchange rate by defining token balances.
//
// tokens and balances must have equal length. The exchange rate is
// implicitly balanceOut/balanceIn based on which tokens match
// tokenIn/tokenOut at execution time.
func (b *Builder) StaticBalances(tokens []common.Address, balances []*big.Int) *Builder {
	if len(tokens) != len(balances) {
		panic("tokens and balances must have equal length")
	}
	n := len(tokens)
	args := make([]byte, 2+20*n+32*n)
	binary.BigEndian.PutUint16(args[0:2], uint16(n))
	offset := 2
	for i := 0; i < n; i++ {
		copy(args[offset:], tokens[i].Bytes())
		offset += 20
	}
	for i := 0; i < n; i++ {
		balances[i].FillBytes(args[offset : offset+32])
		offset += 32
	}
	return b.emit(OpStaticBalancesXD, args)
}

// LimitSwap adds the limit swap computation instruction.
//
// makerDirectionLt must be true if tokenIn < tokenOut (by address),
// false otherwise. This is validated on-chain.
func (b *Builder) LimitSwap(makerDirectionLt bool) *Builder {
	args := []byte{0}
	if makerDirectionLt {
		args[0] = 1
	}
	return b.emit(OpLimitSwap1D, args)
}

// LimitSwapOnlyFull adds a limit swap that only allows full fills.
func (b *Builder) LimitSwapOnlyFull(makerDirectionLt bool) *Builder {
	args := []byte{0}
	if makerDirectionLt {
		args[0] = 1
	}
	return b.emit(OpLimitSwapOnlyFull1D, args)
}

// RequireMinRate adds a minimum rate enforcement that reverts
// if the rate is below the specified minimum.
func (b *Builder) RequireMinRate(rateLt, rateGt uint64) *Builder {
	args := make([]byte, 16)
	binary.BigEndian.PutUint64(args[0:8], rateLt)
	binary.BigEndian.PutUint64(args[8:16], rateGt)
	return b.emit(OpRequireMinRate1D, args)
}

// FlatFeeAmountIn wraps the remaining program with a flat fee on amountIn.
// feeBps uses 1e9 = 100% scale. Must be the FIRST instruction.
func (b *Builder) FlatFeeAmountIn(feeBps uint32) *Builder {
	args := make([]byte, 4)
	binary.BigEndian.PutUint32(args, feeBps)
	return b.emit(OpFlatFeeAmountInXD, args)
}

// Build returns the final bytecode program.
func (b *Builder) Build() []byte {
	return b.buf.Bytes()
}

// --- Convenience constructors ---

// OneTimeLimitOrder creates a simple one-time limit order program.
//
//	tokenA/tokenB: the two tokens in the pair
//	balanceA/balanceB: the balances defining the exchange rate
//	bitIndex: unique bit index for invalidation
func OneTimeLimitOrder(
	tokenA, tokenB common.Address,
	balanceA, balanceB *big.Int,
	bitIndex uint32,
) []byte {
	directionLt := bytes.Compare(tokenA.Bytes(), tokenB.Bytes()) < 0
	return New().
		InvalidateBit(bitIndex).
		StaticBalances(
			[]common.Address{tokenA, tokenB},
			[]*big.Int{balanceA, balanceB},
		).
		LimitSwap(directionLt).
		Build()
}

// PartialFillLimitOrder creates a limit order that supports partial fills.
func PartialFillLimitOrder(
	tokenA, tokenB common.Address,
	balanceA, balanceB *big.Int,
) []byte {
	directionLt := bytes.Compare(tokenA.Bytes(), tokenB.Bytes()) < 0
	return New().
		StaticBalances(
			[]common.Address{tokenA, tokenB},
			[]*big.Int{balanceA, balanceB},
		).
		LimitSwap(directionLt).
		InvalidateTokenOut().
		Build()
}

// TimedLimitOrder creates a one-time limit order with a deadline.
func TimedLimitOrder(
	tokenA, tokenB common.Address,
	balanceA, balanceB *big.Int,
	bitIndex uint32,
	deadlineUnix uint64,
) []byte {
	directionLt := bytes.Compare(tokenA.Bytes(), tokenB.Bytes()) < 0
	return New().
		Deadline(deadlineUnix).
		InvalidateBit(bitIndex).
		StaticBalances(
			[]common.Address{tokenA, tokenB},
			[]*big.Int{balanceA, balanceB},
		).
		LimitSwap(directionLt).
		Build()
}
