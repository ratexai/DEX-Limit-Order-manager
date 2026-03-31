package swapvm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	svmtypes "github.com/yura4gus/dexwallettrust/pkg/swapvm/types"
	"github.com/yura4gus/dexwallettrust/pkg/swapvm/program"
	"github.com/yura4gus/dexwallettrust/pkg/swapvm/signer"
)

// Client provides high-level operations for SwapVM limit orders.
type Client struct {
	eth          *ethclient.Client
	abi          abi.ABI
	contractAddr common.Address
	chainID      *big.Int
	domainSep    common.Hash
	domainLoaded bool
}

// NewClient creates a SwapVM client connected to the given RPC endpoint.
func NewClient(ctx context.Context, rpcURL string) (*Client, error) {
	eth, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to RPC: %w", err)
	}

	chainID, err := eth.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting chain ID: %w", err)
	}

	parsed, err := abi.JSON(strings.NewReader(svmtypes.SwapVMABIJSON))
	if err != nil {
		return nil, fmt.Errorf("parsing ABI: %w", err)
	}

	return &Client{
		eth:          eth,
		abi:          parsed,
		contractAddr: svmtypes.SwapVMAddress,
		chainID:      chainID,
	}, nil
}

// loadDomain reads the EIP-712 domain separator from the contract.
func (c *Client) loadDomain(ctx context.Context) error {
	if c.domainLoaded {
		return nil
	}

	callData, err := c.abi.Pack("eip712Domain")
	if err != nil {
		return fmt.Errorf("packing eip712Domain call: %w", err)
	}

	result, err := c.eth.CallContract(ctx, ethereum.CallMsg{
		To:   &c.contractAddr,
		Data: callData,
	}, nil)
	if err != nil {
		return fmt.Errorf("calling eip712Domain: %w", err)
	}

	outputs, err := c.abi.Unpack("eip712Domain", result)
	if err != nil {
		return fmt.Errorf("unpacking eip712Domain: %w", err)
	}

	// outputs: fields, name, version, chainId, verifyingContract, salt, extensions
	name := outputs[1].(string)
	version := outputs[2].(string)

	c.domainSep = signer.DomainSeparator(name, version, c.chainID, c.contractAddr)
	c.domainLoaded = true
	return nil
}

// LimitOrderParams defines parameters for creating a limit order.
type LimitOrderParams struct {
	TokenA   common.Address // first token
	TokenB   common.Address // second token
	AmountA  *big.Int       // balance of tokenA (defines rate)
	AmountB  *big.Int       // balance of tokenB (defines rate)
	BitIndex uint32         // unique bit index for one-time orders (0 for partial fill)
	Deadline uint64         // unix timestamp, 0 for no deadline
	Partial  bool           // true for partial-fill support
}

// CreateLimitOrder creates a signed limit order ready for submission.
func (c *Client) CreateLimitOrder(
	ctx context.Context,
	key *ecdsa.PrivateKey,
	params LimitOrderParams,
) (*svmtypes.Order, []byte, error) {
	if err := c.loadDomain(ctx); err != nil {
		return nil, nil, err
	}

	// Build program bytecode
	var prog []byte
	if params.Partial {
		prog = program.PartialFillLimitOrder(
			params.TokenA, params.TokenB,
			params.AmountA, params.AmountB,
		)
	} else if params.Deadline > 0 {
		prog = program.TimedLimitOrder(
			params.TokenA, params.TokenB,
			params.AmountA, params.AmountB,
			params.BitIndex, params.Deadline,
		)
	} else {
		prog = program.OneTimeLimitOrder(
			params.TokenA, params.TokenB,
			params.AmountA, params.AmountB,
			params.BitIndex,
		)
	}

	// Build order
	maker := ethcrypto.PubkeyToAddress(key.PublicKey)
	traits := svmtypes.SimpleLimitTraits()
	order := &svmtypes.Order{
		Maker:  maker,
		Traits: traits,
		Data:   prog,
	}

	// Sign
	sig, err := signer.SignOrder(key, c.domainSep, maker, traits, prog)
	if err != nil {
		return nil, nil, fmt.Errorf("signing order: %w", err)
	}

	return order, sig, nil
}

// Quote previews swap amounts without executing.
func (c *Client) Quote(
	ctx context.Context,
	order *svmtypes.Order,
	tokenIn, tokenOut common.Address,
	amount *big.Int,
	exactIn bool,
	sig []byte,
) (amountIn, amountOut *big.Int, err error) {
	takerData := svmtypes.NewTakerTraits().
		SetExactIn(exactIn).
		SetSignature(sig).
		Build()

	callData, err := c.abi.Pack("quote",
		struct {
			Maker  common.Address
			Traits *big.Int
			Data   []byte
		}{order.Maker, order.Traits, order.Data},
		tokenIn, tokenOut, amount, takerData,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("packing quote call: %w", err)
	}

	result, err := c.eth.CallContract(ctx, ethereum.CallMsg{
		To:   &c.contractAddr,
		Data: callData,
	}, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("calling quote: %w", err)
	}

	outputs, err := c.abi.Unpack("quote", result)
	if err != nil {
		return nil, nil, fmt.Errorf("unpacking quote: %w", err)
	}

	return outputs[0].(*big.Int), outputs[1].(*big.Int), nil
}

// Swap executes a limit order on-chain.
func (c *Client) Swap(
	ctx context.Context,
	takerKey *ecdsa.PrivateKey,
	order *svmtypes.Order,
	tokenIn, tokenOut common.Address,
	amount *big.Int,
	exactIn bool,
	threshold *big.Int,
	makerSig []byte,
) (*types.Transaction, error) {
	takerBuilder := svmtypes.NewTakerTraits().
		SetExactIn(exactIn).
		SetSignature(makerSig)

	if threshold != nil {
		takerBuilder.SetStrictThreshold(threshold)
	}

	takerData := takerBuilder.Build()

	callData, err := c.abi.Pack("swap",
		struct {
			Maker  common.Address
			Traits *big.Int
			Data   []byte
		}{order.Maker, order.Traits, order.Data},
		tokenIn, tokenOut, amount, takerData,
	)
	if err != nil {
		return nil, fmt.Errorf("packing swap call: %w", err)
	}

	takerAddr := ethcrypto.PubkeyToAddress(takerKey.PublicKey)
	nonce, err := c.eth.PendingNonceAt(ctx, takerAddr)
	if err != nil {
		return nil, fmt.Errorf("getting nonce: %w", err)
	}

	gasPrice, err := c.eth.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting gas price: %w", err)
	}

	tx := types.NewTransaction(nonce, c.contractAddr, big.NewInt(0), 500_000, gasPrice, callData)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(c.chainID), takerKey)
	if err != nil {
		return nil, fmt.Errorf("signing tx: %w", err)
	}

	err = c.eth.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("sending tx: %w", err)
	}

	return signedTx, nil
}

// CancelBitOrder cancels a one-time order by invalidating its bit.
func (c *Client) CancelBitOrder(
	ctx context.Context,
	makerKey *ecdsa.PrivateKey,
	bitIndex uint32,
) (*types.Transaction, error) {
	callData, err := c.abi.Pack("invalidateBit", new(big.Int).SetUint64(uint64(bitIndex)))
	if err != nil {
		return nil, fmt.Errorf("packing invalidateBit: %w", err)
	}

	return c.sendTx(ctx, makerKey, callData)
}

// CancelPartialOrder cancels a partial-fill order.
func (c *Client) CancelPartialOrder(
	ctx context.Context,
	makerKey *ecdsa.PrivateKey,
	orderHash common.Hash,
	token common.Address,
	byTokenIn bool,
) (*types.Transaction, error) {
	var method string
	if byTokenIn {
		method = "invalidateTokenIn"
	} else {
		method = "invalidateTokenOut"
	}

	callData, err := c.abi.Pack(method, orderHash, token)
	if err != nil {
		return nil, fmt.Errorf("packing %s: %w", method, err)
	}

	return c.sendTx(ctx, makerKey, callData)
}

func (c *Client) sendTx(ctx context.Context, key *ecdsa.PrivateKey, data []byte) (*types.Transaction, error) {
	addr := ethcrypto.PubkeyToAddress(key.PublicKey)
	nonce, err := c.eth.PendingNonceAt(ctx, addr)
	if err != nil {
		return nil, err
	}

	gasPrice, err := c.eth.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(nonce, c.contractAddr, big.NewInt(0), 200_000, gasPrice, data)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(c.chainID), key)
	if err != nil {
		return nil, err
	}

	return signed, c.eth.SendTransaction(ctx, signed)
}
