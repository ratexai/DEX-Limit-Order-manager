package types

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// ABI argument types for Order encoding.
var swapVMOrderArgs abi.Arguments

func init() {
	addressType, _ := abi.NewType("address", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)

	swapVMOrderArgs = abi.Arguments{
		{Name: "maker", Type: addressType},
		{Name: "traits", Type: uint256Type},
		{Name: "data", Type: bytesType},
	}
}

// SwapVMABI is the minimal ABI for the SwapVM contract functions
// relevant to limit orders.
const SwapVMABIJSON = `[
	{
		"inputs": [
			{
				"components": [
					{"name": "maker", "type": "address"},
					{"name": "traits", "type": "uint256"},
					{"name": "data", "type": "bytes"}
				],
				"name": "order",
				"type": "tuple"
			},
			{"name": "tokenIn", "type": "address"},
			{"name": "tokenOut", "type": "address"},
			{"name": "amount", "type": "uint256"},
			{"name": "takerTraitsAndData", "type": "bytes"}
		],
		"name": "swap",
		"outputs": [
			{"name": "amountIn", "type": "uint256"},
			{"name": "amountOut", "type": "uint256"}
		],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"components": [
					{"name": "maker", "type": "address"},
					{"name": "traits", "type": "uint256"},
					{"name": "data", "type": "bytes"}
				],
				"name": "order",
				"type": "tuple"
			},
			{"name": "tokenIn", "type": "address"},
			{"name": "tokenOut", "type": "address"},
			{"name": "amount", "type": "uint256"},
			{"name": "takerTraitsAndData", "type": "bytes"}
		],
		"name": "quote",
		"outputs": [
			{"name": "amountIn", "type": "uint256"},
			{"name": "amountOut", "type": "uint256"}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"components": [
					{"name": "maker", "type": "address"},
					{"name": "traits", "type": "uint256"},
					{"name": "data", "type": "bytes"}
				],
				"name": "order",
				"type": "tuple"
			}
		],
		"name": "hash",
		"outputs": [
			{"name": "", "type": "bytes32"}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [{"name": "bitIndex", "type": "uint256"}],
		"name": "invalidateBit",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "orderHash", "type": "bytes32"},
			{"name": "tokenIn", "type": "address"}
		],
		"name": "invalidateTokenIn",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "orderHash", "type": "bytes32"},
			{"name": "tokenOut", "type": "address"}
		],
		"name": "invalidateTokenOut",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "eip712Domain",
		"outputs": [
			{"name": "fields", "type": "bytes1"},
			{"name": "name", "type": "string"},
			{"name": "version", "type": "string"},
			{"name": "chainId", "type": "uint256"},
			{"name": "verifyingContract", "type": "address"},
			{"name": "salt", "type": "bytes32"},
			{"name": "extensions", "type": "uint256[]"}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

// ParsedSwapVMABI returns the parsed ABI for SwapVM.
func ParsedSwapVMABI() (abi.ABI, error) {
	return abi.JSON(strings.NewReader(SwapVMABIJSON))
}
