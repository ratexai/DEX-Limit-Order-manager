package program

// Limit order opcodes from LimitOpcodes.sol.
const (
	OpJump                        = 10
	OpJumpIfTokenIn               = 11
	OpJumpIfTokenOut              = 12
	OpDeadline                    = 13
	OpOnlyTakerTokenBalanceNonZero = 14
	OpOnlyTakerTokenBalanceGte    = 15
	OpOnlyTakerTokenSupplyShareGte = 16
	OpStaticBalancesXD            = 17
	OpInvalidateBit1D             = 18
	OpInvalidateTokenIn1D         = 19
	OpInvalidateTokenOut1D        = 20
	OpLimitSwap1D                 = 21
	OpLimitSwapOnlyFull1D         = 22
	OpRequireMinRate1D            = 23
	OpAdjustMinRate1D             = 24
	OpDutchAuctionBalanceIn1D     = 25
	OpDutchAuctionBalanceOut1D    = 26
	OpBaseFeeAdjuster1D           = 27
	OpTWAP                        = 28
	OpExtruction                  = 29
	OpSalt                        = 30
	OpFlatFeeAmountInXD           = 31
	OpFlatFeeAmountOutXD          = 32
	OpProtocolFeeAmountInXD       = 37
)
