package signer

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// OrderTypeHash is the EIP-712 typehash for the Order struct.
// keccak256("Order(address maker,uint256 traits,bytes data)")
var OrderTypeHash = crypto.Keccak256Hash(
	[]byte("Order(address maker,uint256 traits,bytes data)"),
)

// DomainSeparator computes the EIP-712 domain separator.
// The name and version should be read from the contract's eip712Domain() function.
func DomainSeparator(name, version string, chainID *big.Int, contractAddr common.Address) common.Hash {
	typeHash := crypto.Keccak256Hash(
		[]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
	)
	return crypto.Keccak256Hash(
		typeHash.Bytes(),
		crypto.Keccak256([]byte(name)),
		crypto.Keccak256([]byte(version)),
		common.LeftPadBytes(chainID.Bytes(), 32),
		common.LeftPadBytes(contractAddr.Bytes(), 32),
	)
}

// StructHash computes the EIP-712 struct hash for an Order.
func StructHash(maker common.Address, traits *big.Int, data []byte) common.Hash {
	dataHash := crypto.Keccak256Hash(data)
	return crypto.Keccak256Hash(
		OrderTypeHash.Bytes(),
		common.LeftPadBytes(maker.Bytes(), 32),
		common.LeftPadBytes(traits.Bytes(), 32),
		dataHash.Bytes(),
	)
}

// OrderHash computes the full EIP-712 order hash ready for signing.
func OrderHash(domainSep common.Hash, maker common.Address, traits *big.Int, data []byte) common.Hash {
	structHash := StructHash(maker, traits, data)
	return crypto.Keccak256Hash(
		[]byte("\x19\x01"),
		domainSep.Bytes(),
		structHash.Bytes(),
	)
}

// SignOrder signs an order with the maker's private key and returns
// a 65-byte signature (r || s || v).
func SignOrder(
	key *ecdsa.PrivateKey,
	domainSep common.Hash,
	maker common.Address,
	traits *big.Int,
	data []byte,
) ([]byte, error) {
	hash := OrderHash(domainSep, maker, traits, data)
	sig, err := crypto.Sign(hash.Bytes(), key)
	if err != nil {
		return nil, fmt.Errorf("signing order: %w", err)
	}
	// go-ethereum returns [R || S || V] where V is 0 or 1.
	// EIP-712 expects V = 27 or 28.
	sig[64] += 27
	return sig, nil
}
