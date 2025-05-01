# Meteora DLMM Transaction Parser Documentation

This documentation provides a comprehensive guide to parsing transactions from the Meteora DLMM (Dynamic Liquidity Market Maker) program on the Solana blockchain. This documentation is intended for developers building transaction monitoring, analytics, or historical data services.

## Program Overview

**Program ID**: Varies by network (mainnet-beta, devnet, testnet)  
**IDL Version**: 0.9.0  
**Name**: lb_clmm (Liquidity Bins Concentrated Liquidity Market Maker)

## Transaction Types

This section lists all transaction types (instructions) available in the Meteora DLMM program, along with their account indices and data structure.

### 1. initializeLbPair

Creates a new liquidity bin pair.

**Accounts:**
1. `lbPair` (mut) - The LB pair account to be initialized
2. `binArrayBitmapExtension` (mut, optional) - The bitmap extension account
3. `tokenMintX` - The mint address of token X
4. `tokenMintY` - The mint address of token Y
5. `reserveX` (mut) - The reserve account for token X
6. `reserveY` (mut) - The reserve account for token Y
7. `oracle` (mut) - The oracle account
8. `presetParameter` - The preset parameter account
9. `funder` (mut, signer) - The account paying for the creation
10. `tokenProgram` - The SPL Token program
11. `systemProgram` - The System program
12. `rent` - The Rent sysvar
13. `eventAuthority` - The event authority account
14. `program` - The program ID

**Data:**
- `activeId` (i32) - The initial active bin ID
- `binStep` (u16) - The bin step size

### 2. initializePermissionLbPair

Creates a new permission-based liquidity bin pair.

**Accounts:**
1. `base` (signer) - The base key account
2. `lbPair` (mut) - The LB pair account to be initialized
3. `binArrayBitmapExtension` (mut, optional) - The bitmap extension account
4. `tokenMintX` - The mint address of token X
5. `tokenMintY` - The mint address of token Y
6. `reserveX` (mut) - The reserve account for token X
7. `reserveY` (mut) - The reserve account for token Y
8. `oracle` (mut) - The oracle account
9. `admin` (mut, signer) - The admin account
10. `tokenBadgeX` (optional) - The token badge for token X
11. `tokenBadgeY` (optional) - The token badge for token Y
12. `tokenProgramX` - Token program for X
13. `tokenProgramY` - Token program for Y
14. `systemProgram` - The System program
15. `rent` - The Rent sysvar
16. `eventAuthority` - The event authority account
17. `program` - The program ID

**Data:**
- `ixData` (InitPermissionPairIx) - Initialization data

### 3. initializeCustomizablePermissionlessLbPair

Creates a new customizable permissionless liquidity bin pair.

**Accounts:**
1. `lbPair` (mut) - The LB pair account to be initialized
2. `binArrayBitmapExtension` (mut, optional) - The bitmap extension account
3. `tokenMintX` - The mint address of token X
4. `tokenMintY` - The mint address of token Y
5. `reserveX` (mut) - The reserve account for token X
6. `reserveY` (mut) - The reserve account for token Y
7. `oracle` (mut) - The oracle account
8. `userTokenX` - The user's token account for token X
9. `funder` (mut, signer) - The account paying for the creation
10. `tokenProgram` - The SPL Token program
11. `systemProgram` - The System program
12. `userTokenY` - The user's token account for token Y
13. `eventAuthority` - The event authority account
14. `program` - The program ID

**Data:**
- `params` (CustomizableParams) - Customizable parameters

### 4. initializeBinArrayBitmapExtension

Initializes a bitmap extension account for a bin array.

**Accounts:**
1. `lbPair` - The LB pair account
2. `binArrayBitmapExtension` (mut) - The bitmap extension account to initialize
3. `funder` (mut, signer) - The account paying for the creation
4. `systemProgram` - The System program
5. `rent` - The Rent sysvar

**Data:** None

### 5. initializeBinArray

Initializes a bin array for a specific index.

**Accounts:**
1. `lbPair` - The LB pair account
2. `binArray` (mut) - The bin array account to initialize
3. `funder` (mut, signer) - The account paying for the creation
4. `systemProgram` - The System program

**Data:**
- `index` (i64) - The bin array index

### 6. addLiquidity

Adds liquidity to a position.

**Accounts:**
1. `position` (mut) - The position account
2. `lbPair` (mut) - The LB pair account
3. `binArrayBitmapExtension` (mut, optional) - The bitmap extension account
4. `userTokenX` (mut) - The user's token account for token X
5. `userTokenY` (mut) - The user's token account for token Y
6. `reserveX` (mut) - The reserve account for token X
7. `reserveY` (mut) - The reserve account for token Y
8. `tokenXMint` - The mint address of token X
9. `tokenYMint` - The mint address of token Y
10. `binArrayLower` (mut) - The lower bin array account
11. `binArrayUpper` (mut) - The upper bin array account
12. `sender` (signer) - The sender account
13. `tokenXProgram` - Token program for X
14. `tokenYProgram` - Token program for Y
15. `eventAuthority` - The event authority account
16. `program` - The program ID

**Data:**
- `liquidityParameter` (LiquidityParameter) - Liquidity parameters

### 7. addLiquidityByWeight

Adds liquidity to a position by weight distribution.

**Accounts:**
(Same as addLiquidity)

**Data:**
- `liquidityParameter` (LiquidityParameterByWeight) - Liquidity parameters with weights

### 8. addLiquidityByStrategy

Adds liquidity to a position using a predefined strategy.

**Accounts:**
(Same as addLiquidity)

**Data:**
- `liquidityParameter` (LiquidityParameterByStrategy) - Strategy-based liquidity parameters

### 9. addLiquidityByStrategyOneSide

Adds one-sided liquidity to a position using a predefined strategy.

**Accounts:**
1. `position` (mut) - The position account
2. `lbPair` (mut) - The LB pair account
3. `binArrayBitmapExtension` (mut, optional) - The bitmap extension account
4. `userToken` (mut) - The user's token account
5. `reserve` (mut) - The reserve account
6. `tokenMint` - The token mint address
7. `binArrayLower` (mut) - The lower bin array account
8. `binArrayUpper` (mut) - The upper bin array account
9. `sender` (signer) - The sender account
10. `tokenProgram` - The SPL Token program
11. `eventAuthority` - The event authority account
12. `program` - The program ID

**Data:**
- `liquidityParameter` (LiquidityParameterByStrategyOneSide) - One-sided strategy parameters

### 10. addLiquidityOneSide

Adds one-sided liquidity to a position.

**Accounts:**
(Same as addLiquidityByStrategyOneSide)

**Data:**
- `liquidityParameter` (LiquidityOneSideParameter) - One-sided liquidity parameters

### 11. removeLiquidity

Removes liquidity from a position.

**Accounts:**
1. `position` (mut) - The position account
2. `lbPair` (mut) - The LB pair account
3. `binArrayBitmapExtension` (mut, optional) - The bitmap extension account
4. `userTokenX` (mut) - The user's token account for token X
5. `userTokenY` (mut) - The user's token account for token Y
6. `reserveX` (mut) - The reserve account for token X
7. `reserveY` (mut) - The reserve account for token Y
8. `tokenXMint` - The mint address of token X
9. `tokenYMint` - The mint address of token Y
10. `binArrayLower` (mut) - The lower bin array account
11. `binArrayUpper` (mut) - The upper bin array account
12. `sender` (signer) - The sender account
13. `tokenXProgram` - Token program for X
14. `tokenYProgram` - Token program for Y
15. `eventAuthority` - The event authority account
16. `program` - The program ID

**Data:**
- `binLiquidityRemoval` (vec<BinLiquidityReduction>) - Array of bin liquidity reductions

### 12. initializePosition

Initializes a new position.

**Accounts:**
1. `payer` (mut, signer) - The account paying for the creation
2. `position` (mut, signer) - The position account to be created
3. `lbPair` - The LB pair account
4. `owner` (signer) - The position owner
5. `systemProgram` - The System program
6. `rent` - The Rent sysvar
7. `eventAuthority` - The event authority account
8. `program` - The program ID

**Data:**
- `lowerBinId` (i32) - The lower bin ID bound
- `width` (i32) - The width of the position in bins

### 13. initializePositionPda

Initializes a new position using a PDA.

**Accounts:**
1. `payer` (mut, signer) - The account paying for the creation
2. `base` (signer) - The base key account
3. `position` (mut) - The position account to be created
4. `lbPair` - The LB pair account
5. `owner` (signer) - The position owner
6. `systemProgram` - The System program
7. `rent` - The Rent sysvar
8. `eventAuthority` - The event authority account
9. `program` - The program ID

**Data:**
- `lowerBinId` (i32) - The lower bin ID bound
- `width` (i32) - The width of the position in bins

### 14. initializePositionByOperator

Initializes a new position by an operator.

**Accounts:**
1. `payer` (mut, signer) - The account paying for the creation
2. `base` (signer) - The base key account
3. `position` (mut) - The position account to be created
4. `lbPair` - The LB pair account
5. `owner` - The position owner
6. `operator` (signer) - The operator account
7. `operatorTokenX` - The operator's token account for token X
8. `ownerTokenX` - The owner's token account for token X
9. `systemProgram` - The System program
10. `eventAuthority` - The event authority account
11. `program` - The program ID

**Data:**
- `lowerBinId` (i32) - The lower bin ID bound
- `width` (i32) - The width of the position in bins
- `feeOwner` (publicKey) - The fee recipient account
- `lockReleasePoint` (u64) - The lock release point

### 15. updatePositionOperator

Updates the operator of a position.

**Accounts:**
1. `position` (mut) - The position account
2. `owner` (signer) - The position owner
3. `eventAuthority` - The event authority account
4. `program` - The program ID

**Data:**
- `operator` (publicKey) - The new operator account

### 16. swap

Performs a token swap.

**Accounts:**
1. `lbPair` (mut) - The LB pair account
2. `binArrayBitmapExtension` (optional) - The bitmap extension account
3. `reserveX` (mut) - The reserve account for token X
4. `reserveY` (mut) - The reserve account for token Y
5. `userTokenIn` (mut) - The user's token account for input
6. `userTokenOut` (mut) - The user's token account for output
7. `tokenXMint` - The mint address of token X
8. `tokenYMint` - The mint address of token Y
9. `oracle` (mut) - The oracle account
10. `hostFeeIn` (mut, optional) - The host fee account
11. `user` (signer) - The user account
12. `tokenXProgram` - Token program for X
13. `tokenYProgram` - Token program for Y
14. `eventAuthority` - The event authority account
15. `program` - The program ID

**Data:**
- `amountIn` (u64) - The input amount
- `minAmountOut` (u64) - The minimum output amount

### 17. swapExactOut

Performs a token swap with exact output amount.

**Accounts:**
(Same as swap)

**Data:**
- `maxInAmount` (u64) - The maximum input amount
- `outAmount` (u64) - The exact output amount

### 18. swapWithPriceImpact

Performs a token swap with a specified maximum price impact.

**Accounts:**
(Same as swap)

**Data:**
- `amountIn` (u64) - The input amount
- `activeId` (option<i32>) - The optional active bin ID
- `maxPriceImpactBps` (u16) - The maximum price impact in basis points

### 19. withdrawProtocolFee

Withdraws protocol fees.

**Accounts:**
1. `lbPair` (mut) - The LB pair account
2. `reserveX` (mut) - The reserve account for token X
3. `reserveY` (mut) - The reserve account for token Y
4. `tokenXMint` - The mint address of token X
5. `tokenYMint` - The mint address of token Y
6. `receiverTokenX` (mut) - The receiver's token account for token X
7. `receiverTokenY` (mut) - The receiver's token account for token Y
8. `claimFeeOperator` - The claim fee operator account
9. `operator` (signer) - The operator account
10. `tokenXProgram` - Token program for X
11. `tokenYProgram` - Token program for Y
12. `memoProgram` - The Memo program

**Data:**
- `amountX` (u64) - The amount of token X to withdraw
- `amountY` (u64) - The amount of token Y to withdraw
- `remainingAccountsInfo` (RemainingAccountsInfo) - Additional accounts information

### 20. initializeReward

Initializes a reward for liquidity providers.

**Accounts:**
1. `lbPair` (mut) - The LB pair account
2. `rewardVault` (mut) - The reward vault account
3. `rewardMint` - The reward mint address
4. `tokenBadge` (optional) - The token badge
5. `admin` (mut, signer) - The admin account
6. `tokenProgram` - The SPL Token program
7. `systemProgram` - The System program
8. `rent` - The Rent sysvar
9. `eventAuthority` - The event authority account
10. `program` - The program ID

**Data:**
- `rewardIndex` (u64) - The reward index
- `rewardDuration` (u64) - The reward duration
- `funder` (publicKey) - The funder account

### 21. fundReward

Funds an existing reward.

**Accounts:**
1. `lbPair` (mut) - The LB pair account
2. `rewardVault` (mut) - The reward vault account
3. `rewardMint` - The reward mint address
4. `funderTokenAccount` (mut) - The funder's token account
5. `funder` (signer) - The funder account
6. `binArray` (mut) - The bin array account
7. `tokenProgram` - The SPL Token program
8. `eventAuthority` - The event authority account
9. `program` - The program ID

**Data:**
- `rewardIndex` (u64) - The reward index
- `amount` (u64) - The amount to fund
- `carryForward` (bool) - Whether to carry forward any existing rewards
- `remainingAccountsInfo` (RemainingAccountsInfo) - Additional accounts information

### 22. updateRewardFunder

Updates the funder of a reward.

**Accounts:**
1. `lbPair` (mut) - The LB pair account
2. `admin` (signer) - The admin account
3. `eventAuthority` - The event authority account
4. `program` - The program ID

**Data:**
- `rewardIndex` (u64) - The reward index
- `newFunder` (publicKey) - The new funder account

### 23. updateRewardDuration

Updates the duration of a reward.

**Accounts:**
1. `lbPair` (mut) - The LB pair account
2. `admin` (signer) - The admin account
3. `binArray` (mut) - The bin array account
4. `eventAuthority` - The event authority account
5. `program` - The program ID

**Data:**
- `rewardIndex` (u64) - The reward index
- `newDuration` (u64) - The new duration

### 24. claimReward

Claims accumulated rewards.

**Accounts:**
1. `lbPair` (mut) - The LB pair account
2. `position` (mut) - The position account
3. `binArrayLower` (mut) - The lower bin array account
4. `binArrayUpper` (mut) - The upper bin array account
5. `sender` (signer) - The sender account
6. `rewardVault` (mut) - The reward vault account
7. `rewardMint` - The reward mint address
8. `userTokenAccount` (mut) - The user's token account
9. `tokenProgram` - The SPL Token program
10. `eventAuthority` - The event authority account
11. `program` - The program ID

**Data:**
- `rewardIndex` (u64) - The reward index

### 25. claimFee

Claims accumulated swap fees.

**Accounts:**
1. `lbPair` (mut) - The LB pair account
2. `position` (mut) - The position account
3. `binArrayLower` (mut) - The lower bin array account
4. `binArrayUpper` (mut) - The upper bin array account
5. `sender` (signer) - The sender account
6. `reserveX` (mut) - The reserve account for token X
7. `reserveY` (mut) - The reserve account for token Y
8. `userTokenX` (mut) - The user's token account for token X
9. `userTokenY` (mut) - The user's token account for token Y
10. `tokenXMint` - The mint address of token X
11. `tokenYMint` - The mint address of token Y
12. `tokenProgram` - The SPL Token program
13. `eventAuthority` - The event authority account
14. `program` - The program ID

**Data:** None

### 26. closePosition

Closes a position.

**Accounts:**
1. `position` (mut) - The position account
2. `lbPair` (mut) - The LB pair account
3. `binArrayLower` (mut) - The lower bin array account
4. `binArrayUpper` (mut) - The upper bin array account
5. `sender` (signer) - The sender account
6. `rentReceiver` (mut) - The rent receiver account
7. `eventAuthority` - The event authority account
8. `program` - The program ID

**Data:** None

### 27-87. Additional Operations

The DLMM program contains many more instructions for specialized operations including:

- updateBaseFeeParameters
- updateDynamicFeeParameters
- increaseOracleLength
- initializePresetParameter
- closePresetParameter
- closePresetParameter2
- removeAllLiquidity
- setPairStatus
- migratePosition
- migrateBinArray
- updateFeesAndRewards
- withdrawIneligibleReward
- setActivationPoint
- removeLiquidityByRange
- addLiquidityOneSidePrecise
- goToABin
- setPreActivationDuration
- setPreActivationSwapAddress
- setPairStatusPermissionless
- initializeTokenBadge
- createClaimProtocolFeeOperator
- closeClaimProtocolFeeOperator
- Various V2 versions of functions (addLiquidity2, claimFee2, etc.)

## Event Types

The DLMM program emits various events that can be parsed from transaction logs:

1. `CompositionFee`
2. `AddLiquidity`
3. `RemoveLiquidity`
4. `Swap`
5. `ClaimReward`
6. `FundReward`
7. `InitializeReward`
8. `UpdateRewardDuration`
9. `UpdateRewardFunder`
10. `PositionClose`
11. `ClaimFee`
12. `LbPairCreate`
13. `PositionCreate`
14. `IncreasePositionLength`
15. `DecreasePositionLength`
16. `FeeParameterUpdate`
17. `DynamicFeeParameterUpdate`
18. `IncreaseObservation`
19. `WithdrawIneligibleReward`
20. `UpdatePositionOperator`
21. `UpdatePositionLockReleasePoint`
22. `GoToABin`

## Common Data Structures

### BinLiquidityReduction
Used for removing liquidity from specific bins.
```typescript
{
  binId: i32,
  liquidityAmount: BN
}
```

### LiquidityParameter
Used for adding liquidity to a position.
```typescript
{
  amount: {
    x: BN,
    y: BN
  },
  activeId: i32,
  maxActiveBinSlippage: number,
  binLiquidityDist: BinLiquidityDistribution[]
}
```

### StrategyParameters
Used for strategy-based liquidity provisioning.
```typescript
{
  maxBinId: number,
  minBinId: number,
  strategyType: StrategyType
}
```

### BinLiquidity
Represents a single bin with its liquidity information.
```typescript
{
  binId: number,
  xAmount: BN,
  yAmount: BN,
  supply: BN,
  price: string,
  version: number,
  pricePerToken: string
}
```

## Error Codes

The DLMM program defines various error codes (6000-6082) for different error conditions, such as:
- 6000: InvalidStartBinIndex
- 6001: InvalidBinId
- 6002: InvalidInput
- 6003: ExceededAmountSlippageTolerance
- 6004: ExceededBinSlippageTolerance
- ...
- 6082: NotSupportAtTheMoment

## Implementing a Transaction Parser

When implementing a transaction parser for Meteora DLMM:

1. Identify the instruction by program ID and instruction discriminator
2. Parse the accounts based on the instruction type
3. Parse the instruction data using the appropriate data structure
4. Process any emitted events from transaction logs
5. Handle potential errors using the error code mapping

## Tips for Efficient Parsing

1. Cache account lookups to reduce RPC calls
2. Watch for IDL updates as new features are released
3. Handle version differences in account structures
4. Consider using the SDK types for proper data handling
5. Implement robust error handling for all potential error codes

## Example Transaction Format

Here's an example of what a parsed swap transaction might look like:

```json
{
  "type": "swap",
  "accounts": {
    "lbPair": "ARwi1S4DaiTG5DX7S4M4ZsrXqpMD1MrTmbu9ue2tpmEq",
    "reserveX": "BVDkb7jQM4GzZVpQ9YsbxpZH8bBVHgvxuaUZH9FdP8Ft",
    "reserveY": "GBzwdxAXEpE6iYGKhVp9PQJnXsmGPmQJiZdALMZoMQwr",
    "userTokenIn": "8ZrEcJHv5q9MqzgFkCCifZLmuJpHTv5GhJuUaGjEGXbE",
    "userTokenOut": "3FCEfHj8Jb9J1pssxe1b3HVYbGK6Z8ioQcFKYb2KhGN1",
    "tokenXMint": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
    "tokenYMint": "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB",
    "oracle": "AyKxHMBxr9qZBQ28jNdX2YgYPijAGuKUyXELJnPfqGQT",
    "user": "EFLbqVnHgAZ8GvddKnpdWQTV3vLLKgpZqB8UhET8ymUP"
  },
  "data": {
    "amountIn": "100000000",
    "minAmountOut": "72614843"
  },
  "events": {
    "swap": {
      "lbPair": "ARwi1S4DaiTG5DX7S4M4ZsrXqpMD1MrTmbu9ue2tpmEq",
      "from": "EFLbqVnHgAZ8GvddKnpdWQTV3vLLKgpZqB8UhET8ymUP",
      "startBinId": 8191,
      "endBinId": 8191,
      "amountIn": "100000000",
      "amountOut": "72764321",
      "swapForY": true,
      "fee": "300000",
      "protocolFee": "0",
      "feeBps": "3000",
      "hostFee": "0"
    }
  }
}
```

## Conclusion

This documentation provides a comprehensive reference for parsing Meteora DLMM transactions. By understanding the account structures, data formats, and event types, developers can build robust transaction parsers for monitoring and analyzing Meteora DLMM activity on the Solana blockchain. 