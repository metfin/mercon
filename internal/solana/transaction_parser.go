package solana

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/wnt/mercon/internal/constants"
	"github.com/wnt/mercon/internal/models"
)

// NewTransactionParser creates a new parser for processing transactions
func NewTransactionParser(client *Client) *TransactionParser {
	return &TransactionParser{
		Client: client,
	}
}

// TransactionParser processes Solana transactions
type TransactionParser struct {
	Client *Client
}

// MeteoraTxType represents the type of Meteora transaction
type MeteoraTxType uint8

const (
	MeteoraTxUnknown MeteoraTxType = iota
	MeteoraTxSwap
	MeteoraTxAddLiquidity
	MeteoraTxRemoveLiquidity
	MeteoraTxClaimFee
	MeteoraTxClaimReward
	MeteoraTxFundReward
	MeteoraTxInitializePosition
	MeteoraTxClosePosition
	MeteoraTxInitializePair
)

// ProcessTransaction processes a transaction and extracts relevant Meteora data
func (p *TransactionParser) ProcessTransaction(ctx context.Context, tx Transaction) (*models.Transaction, error) {
	// First we filter all the non-meteora tx instructions
	meteoraInstructions := make([]Instruction, 0)
	for _, instruction := range tx.Instructions {
		if instruction.ProgramId == constants.MeteoraDLMM {
			meteoraInstructions = append(meteoraInstructions, instruction)
		}
	}

	if len(meteoraInstructions) == 0 {
		return nil, fmt.Errorf("no meteora instructions found in transaction %s", tx.Signature)
	}

	// Create the base transaction model
	txModel := &models.Transaction{
		Signature:   tx.Signature,
		BlockTime:   UnixTimeToTime(tx.Timestamp),
		Slot:        tx.Slot,
		Description: tx.Description,
		Type:        tx.Type,
		Source:      tx.Source,
		Fee:         tx.Fee,
		FeePayer:    tx.FeePayer,
	}

	if tx.TransactionError != nil {
		txModel.Error = tx.TransactionError.Error
	}

	// Process each Meteora instruction
	for i, instruction := range meteoraInstructions {
		err := p.processMeteoraTxInstruction(ctx, instruction, txModel, i)
		if err != nil {
			return nil, fmt.Errorf("error processing meteora instruction: %v", err)
		}
	}

	return txModel, nil
}

// processMeteoraTxInstruction processes a single Meteora instruction and updates the transaction model
func (p *TransactionParser) processMeteoraTxInstruction(ctx context.Context, instruction Instruction, txModel *models.Transaction, index int) error {
	// Decode instruction type from the first byte of data
	data, err := base64.StdEncoding.DecodeString(instruction.Data)
	if err != nil {
		return fmt.Errorf("error decoding instruction data: %v", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("empty instruction data")
	}

	// The first byte is the instruction discriminator
	instructionType := data[0]

	// Process based on instruction type
	switch instructionType {
	case 16: // swap
		return p.parseSwap(ctx, instruction, data, txModel)
	case 17: // swapExactOut
		return p.parseSwapExactOut(ctx, instruction, data, txModel)
	case 6: // addLiquidity
		return p.parseAddLiquidity(ctx, instruction, data, txModel)
	case 7: // addLiquidityByWeight
		return p.parseAddLiquidityByWeight(ctx, instruction, data, txModel)
	case 8: // addLiquidityByStrategy
		return p.parseAddLiquidityByStrategy(ctx, instruction, data, txModel)
	case 9: // addLiquidityByStrategyOneSide
		return p.parseAddLiquidityByStrategyOneSide(ctx, instruction, data, txModel)
	case 11: // removeLiquidity
		return p.parseRemoveLiquidity(ctx, instruction, data, txModel)
	case 25: // claimFee
		return p.parseClaimFee(ctx, instruction, data, txModel)
	case 24: // claimReward
		return p.parseClaimReward(ctx, instruction, data, txModel)
	case 21: // fundReward
		return p.parseFundReward(ctx, instruction, data, txModel)
	case 12, 13, 14: // initializePosition variants
		return p.parseInitializePosition(ctx, instruction, data, txModel)
	case 26: // closePosition
		return p.parseClosePosition(ctx, instruction, data, txModel)
	case 1, 2, 3: // initialize pair variants
		return p.parseInitializePair(ctx, instruction, data, txModel)
	default:
		// Other instruction types not explicitly handled
		return nil
	}
}

// parseSwap parses a swap instruction
func (p *TransactionParser) parseSwap(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	if len(instruction.Accounts) < 9 {
		return fmt.Errorf("insufficient accounts for swap operation")
	}

	if len(data) < 17 { // 1 byte discriminator + 8 bytes amountIn + 8 bytes minAmountOut
		return fmt.Errorf("insufficient data for swap operation")
	}

	// Extract parameters from data
	amountIn := binary.LittleEndian.Uint64(data[1:9])
	minAmountOut := binary.LittleEndian.Uint64(data[9:17])

	// Extract accounts
	lbPair := instruction.Accounts[0]
	// reserveX and reserveY used for reference but not directly
	// userTokenIn and userTokenOut used for determining swap direction
	userTokenIn := instruction.Accounts[4]
	// tokenXMint and tokenYMint are used
	tokenXMint := instruction.Accounts[6]
	tokenYMint := instruction.Accounts[7]
	// oracle not directly used
	user := instruction.Accounts[10] // Index 10 for user account

	// Get pair information
	pairID, err := p.getPairID(ctx, lbPair)
	if err != nil {
		return err
	}

	// Determine wallet ID
	walletID, err := p.getWalletID(ctx, user)
	if err != nil {
		return err
	}

	// Check if token in is X or Y
	swapForY := false
	tokenInMint := ""
	tokenOutMint := ""

	// Check if userTokenIn is associated with tokenXMint
	isXToY, err := p.isTokenXToY(ctx, userTokenIn, tokenXMint)
	if err != nil {
		return err
	}

	swapForY = isXToY
	if swapForY {
		tokenInMint = tokenXMint
		tokenOutMint = tokenYMint
	} else {
		tokenInMint = tokenYMint
		tokenOutMint = tokenXMint
	}

	// Extract additional event data if available
	amountOut := minAmountOut
	fee := uint64(0)
	feeBps := uint16(0)
	protocolFee := uint64(0)
	startBinID := int32(0)
	endBinID := int32(0)

	// Extract swap event from tx events if available - note: this is a placeholder
	// In a complete implementation, we would use the transaction events
	if txModel.Source == "meteora" {
		// Try to parse the swap event for more details
		// This would be specific to the format of the Solana program's events
		// Placeholder for actual event parsing logic
	}

	// Create the swap model
	swap := models.MeteoraSwap{
		TransactionID: txModel.ID,
		PairID:        pairID,
		WalletID:      walletID,
		User:          user,
		TokenInMint:   tokenInMint,
		TokenOutMint:  tokenOutMint,
		AmountIn:      amountIn,
		AmountOut:     amountOut,
		MinAmountOut:  minAmountOut,
		Fee:           fee,
		ProtocolFee:   protocolFee,
		FeeBps:        feeBps,
		SwapTime:      txModel.BlockTime,
		StartBinID:    startBinID,
		EndBinID:      endBinID,
		SwapForY:      swapForY,
	}

	// Add swap to transaction
	txModel.Swaps = append(txModel.Swaps, swap)

	return nil
}

// parseSwapExactOut parses a swapExactOut instruction
func (p *TransactionParser) parseSwapExactOut(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	if len(instruction.Accounts) < 9 {
		return fmt.Errorf("insufficient accounts for swap exact out operation")
	}

	if len(data) < 17 { // 1 byte discriminator + 8 bytes maxAmountIn + 8 bytes exactAmountOut
		return fmt.Errorf("insufficient data for swap exact out operation")
	}

	// Extract parameters from data
	maxAmountIn := binary.LittleEndian.Uint64(data[1:9])
	exactAmountOut := binary.LittleEndian.Uint64(data[9:17])

	// Extract accounts (same as swap)
	lbPair := instruction.Accounts[0]
	// reserveX and reserveY used for reference but not directly
	// userTokenIn and userTokenOut used for determining swap direction
	userTokenIn := instruction.Accounts[4]
	// tokenXMint and tokenYMint are used
	tokenXMint := instruction.Accounts[6]
	tokenYMint := instruction.Accounts[7]
	// oracle not directly used
	user := instruction.Accounts[10] // Index 10 for user account

	// Get pair information
	pairID, err := p.getPairID(ctx, lbPair)
	if err != nil {
		return err
	}

	// Determine wallet ID
	walletID, err := p.getWalletID(ctx, user)
	if err != nil {
		return err
	}

	// Check if token in is X or Y
	swapForY := false
	tokenInMint := ""
	tokenOutMint := ""

	// Check if userTokenIn is associated with tokenXMint
	isXToY, err := p.isTokenXToY(ctx, userTokenIn, tokenXMint)
	if err != nil {
		return err
	}

	swapForY = isXToY
	if swapForY {
		tokenInMint = tokenXMint
		tokenOutMint = tokenYMint
	} else {
		tokenInMint = tokenYMint
		tokenOutMint = tokenXMint
	}

	// Create the swap model
	swap := models.MeteoraSwap{
		TransactionID: txModel.ID,
		PairID:        pairID,
		WalletID:      walletID,
		User:          user,
		TokenInMint:   tokenInMint,
		TokenOutMint:  tokenOutMint,
		AmountIn:      maxAmountIn, // This is max, actual amount may be different
		AmountOut:     exactAmountOut,
		MinAmountOut:  exactAmountOut, // In exact out, this is the exact amount
		SwapTime:      txModel.BlockTime,
		SwapForY:      swapForY,
	}

	// Add swap to transaction
	txModel.Swaps = append(txModel.Swaps, swap)

	return nil
}

// parseAddLiquidity parses an addLiquidity instruction
func (p *TransactionParser) parseAddLiquidity(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	if len(instruction.Accounts) < 12 {
		return fmt.Errorf("insufficient accounts for add liquidity operation")
	}

	// Extract accounts
	position := instruction.Accounts[0]
	lbPair := instruction.Accounts[1]
	// User token accounts used for reference when tracking transfers
	// userTokenX := instruction.Accounts[3]
	// userTokenY := instruction.Accounts[4]
	reserveX := instruction.Accounts[5]
	reserveY := instruction.Accounts[6]
	tokenXMint := instruction.Accounts[7]
	tokenYMint := instruction.Accounts[8]
	user := instruction.Accounts[12] // Index 12 for sender account

	// Get position, pair, and wallet information
	positionID, err := p.getPositionID(ctx, position)
	if err != nil {
		return err
	}

	pairID, err := p.getPairID(ctx, lbPair)
	if err != nil {
		return err
	}

	walletID, err := p.getWalletID(ctx, user)
	if err != nil {
		return err
	}

	// Parse amount X and Y from token transfers
	// This is simplified and would need to be replaced with actual token transfer tracking
	amountX := uint64(0)
	amountY := uint64(0)

	for _, transfer := range txModel.TokenTransfers {
		if transfer.ToTokenAccount == reserveX && transfer.Mint == tokenXMint {
			amountX = uint64(transfer.TokenAmount)
		} else if transfer.ToTokenAccount == reserveY && transfer.Mint == tokenYMint {
			amountY = uint64(transfer.TokenAmount)
		}
	}

	// Get active bin ID (default to 0 if not available)
	activeID := int32(0)

	// Create the liquidity addition model
	addition := models.MeteoraLiquidityAddition{
		TransactionID: txModel.ID,
		PositionID:    positionID,
		PairID:        pairID,
		WalletID:      walletID,
		User:          user,
		AmountX:       amountX,
		AmountY:       amountY,
		ActiveID:      activeID,
		AddTime:       txModel.BlockTime,
	}

	// Add liquidity addition to transaction
	txModel.LiquidityAdditions = append(txModel.LiquidityAdditions, addition)

	return nil
}

// Other liquidity addition variants can be added with similar implementations
func (p *TransactionParser) parseAddLiquidityByWeight(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	// Similar to parseAddLiquidity but with weight distribution
	return p.parseAddLiquidity(ctx, instruction, data, txModel)
}

func (p *TransactionParser) parseAddLiquidityByStrategy(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	// Similar to parseAddLiquidity but with strategy
	return p.parseAddLiquidity(ctx, instruction, data, txModel)
}

// parseAddLiquidityByStrategyOneSide parses an addLiquidityByStrategyOneSide instruction
func (p *TransactionParser) parseAddLiquidityByStrategyOneSide(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	// Similar to parseAddLiquidity but one-sided
	if len(instruction.Accounts) < 10 {
		return fmt.Errorf("insufficient accounts for add liquidity one side operation")
	}

	// Extract accounts
	position := instruction.Accounts[0]
	lbPair := instruction.Accounts[1]
	// User token account used for reference when tracking transfers
	// userToken := instruction.Accounts[3]
	reserve := instruction.Accounts[4]
	tokenMint := instruction.Accounts[5]
	user := instruction.Accounts[8] // Index 8 for sender account

	// Get position, pair, and wallet information
	positionID, err := p.getPositionID(ctx, position)
	if err != nil {
		return err
	}

	pairID, err := p.getPairID(ctx, lbPair)
	if err != nil {
		return err
	}

	walletID, err := p.getWalletID(ctx, user)
	if err != nil {
		return err
	}

	// Parse amount from token transfers
	amount := uint64(0)

	for _, transfer := range txModel.TokenTransfers {
		if transfer.ToTokenAccount == reserve && transfer.Mint == tokenMint {
			amount = uint64(transfer.TokenAmount)
		}
	}

	// Determine if this is X or Y token
	isXToken, err := p.isXToken(ctx, tokenMint, lbPair)
	if err != nil {
		return err
	}

	// Create the liquidity addition model
	addition := models.MeteoraLiquidityAddition{
		TransactionID: txModel.ID,
		PositionID:    positionID,
		PairID:        pairID,
		WalletID:      walletID,
		User:          user,
		AddTime:       txModel.BlockTime,
	}

	if isXToken {
		addition.AmountX = amount
	} else {
		addition.AmountY = amount
	}

	// Add liquidity addition to transaction
	txModel.LiquidityAdditions = append(txModel.LiquidityAdditions, addition)

	return nil
}

// parseRemoveLiquidity parses a removeLiquidity instruction
func (p *TransactionParser) parseRemoveLiquidity(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	if len(instruction.Accounts) < 12 {
		return fmt.Errorf("insufficient accounts for remove liquidity operation")
	}

	// Extract accounts
	position := instruction.Accounts[0]
	lbPair := instruction.Accounts[1]
	// User token accounts used for reference when tracking transfers
	// userTokenX := instruction.Accounts[3]
	// userTokenY := instruction.Accounts[4]
	reserveX := instruction.Accounts[5]
	reserveY := instruction.Accounts[6]
	tokenXMint := instruction.Accounts[7]
	tokenYMint := instruction.Accounts[8]
	user := instruction.Accounts[12] // Index 12 for sender account

	// Get position, pair, and wallet information
	positionID, err := p.getPositionID(ctx, position)
	if err != nil {
		return err
	}

	pairID, err := p.getPairID(ctx, lbPair)
	if err != nil {
		return err
	}

	walletID, err := p.getWalletID(ctx, user)
	if err != nil {
		return err
	}

	// Parse amount X and Y from token transfers
	// This is simplified and would need to be replaced with actual token transfer tracking
	amountX := uint64(0)
	amountY := uint64(0)

	for _, transfer := range txModel.TokenTransfers {
		if transfer.FromTokenAccount == reserveX && transfer.Mint == tokenXMint {
			amountX = uint64(transfer.TokenAmount)
		} else if transfer.FromTokenAccount == reserveY && transfer.Mint == tokenYMint {
			amountY = uint64(transfer.TokenAmount)
		}
	}

	// Create the liquidity removal model
	removal := models.MeteoraLiquidityRemoval{
		TransactionID:  txModel.ID,
		PositionID:     positionID,
		PairID:         pairID,
		WalletID:       walletID,
		User:           user,
		AmountXRemoved: amountX,
		AmountYRemoved: amountY,
		RemoveTime:     txModel.BlockTime,
	}

	// Add liquidity removal to transaction
	txModel.LiquidityRemovals = append(txModel.LiquidityRemovals, removal)

	return nil
}

// parseClaimFee parses a claimFee instruction
func (p *TransactionParser) parseClaimFee(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	if len(instruction.Accounts) < 12 {
		return fmt.Errorf("insufficient accounts for claim fee operation")
	}

	// Extract accounts
	lbPair := instruction.Accounts[0]
	position := instruction.Accounts[1]
	user := instruction.Accounts[4] // Index 4 for sender account
	reserveX := instruction.Accounts[5]
	reserveY := instruction.Accounts[6]
	userTokenX := instruction.Accounts[7]
	userTokenY := instruction.Accounts[8]
	tokenXMint := instruction.Accounts[9]
	tokenYMint := instruction.Accounts[10]

	// Get position, pair, and wallet information
	positionID, err := p.getPositionID(ctx, position)
	if err != nil {
		return err
	}

	pairID, err := p.getPairID(ctx, lbPair)
	if err != nil {
		return err
	}

	walletID, err := p.getWalletID(ctx, user)
	if err != nil {
		return err
	}

	// Parse amount X and Y from token transfers
	amountX := uint64(0)
	amountY := uint64(0)

	for _, transfer := range txModel.TokenTransfers {
		if transfer.FromTokenAccount == reserveX && transfer.ToTokenAccount == userTokenX && transfer.Mint == tokenXMint {
			amountX = uint64(transfer.TokenAmount)
		} else if transfer.FromTokenAccount == reserveY && transfer.ToTokenAccount == userTokenY && transfer.Mint == tokenYMint {
			amountY = uint64(transfer.TokenAmount)
		}
	}

	// Create the fee claim model
	claim := models.MeteoraFeeClaim{
		TransactionID: txModel.ID,
		PositionID:    positionID,
		PairID:        pairID,
		WalletID:      walletID,
		User:          user,
		AmountX:       amountX,
		AmountY:       amountY,
		ClaimTime:     txModel.BlockTime,
	}

	// Add fee claim to transaction
	txModel.FeeClaims = append(txModel.FeeClaims, claim)

	return nil
}

// parseClaimReward parses a claimReward instruction
func (p *TransactionParser) parseClaimReward(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	if len(instruction.Accounts) < 10 {
		return fmt.Errorf("insufficient accounts for claim reward operation")
	}

	if len(data) < 9 { // 1 byte discriminator + 8 bytes reward index
		return fmt.Errorf("insufficient data for claim reward operation")
	}

	// Extract parameters from data
	rewardIndex := binary.LittleEndian.Uint64(data[1:9])

	// Extract accounts
	lbPair := instruction.Accounts[0]
	position := instruction.Accounts[1]
	user := instruction.Accounts[4] // Index 4 for sender account
	rewardVault := instruction.Accounts[5]
	rewardMint := instruction.Accounts[6]
	userTokenAccount := instruction.Accounts[7]

	// Get position, pair, reward, and wallet information
	positionID, err := p.getPositionID(ctx, position)
	if err != nil {
		return err
	}

	pairID, err := p.getPairID(ctx, lbPair)
	if err != nil {
		return err
	}

	rewardID, err := p.getRewardID(ctx, lbPair, rewardIndex)
	if err != nil {
		return err
	}

	walletID, err := p.getWalletID(ctx, user)
	if err != nil {
		return err
	}

	// Parse amount from token transfers
	amount := uint64(0)

	for _, transfer := range txModel.TokenTransfers {
		if transfer.FromTokenAccount == rewardVault && transfer.ToTokenAccount == userTokenAccount && transfer.Mint == rewardMint {
			amount = uint64(transfer.TokenAmount)
		}
	}

	// Create the reward claim model
	claim := models.MeteoraRewardClaim{
		TransactionID: txModel.ID,
		PositionID:    positionID,
		RewardID:      rewardID,
		PairID:        pairID,
		WalletID:      walletID,
		User:          user,
		Amount:        amount,
		ClaimTime:     txModel.BlockTime,
	}

	// Add reward claim to transaction
	txModel.RewardClaims = append(txModel.RewardClaims, claim)

	return nil
}

// parseFundReward parses a fundReward instruction
func (p *TransactionParser) parseFundReward(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	if len(instruction.Accounts) < 8 {
		return fmt.Errorf("insufficient accounts for fund reward operation")
	}

	if len(data) < 18 { // 1 byte discriminator + 8 bytes reward index + 8 bytes amount + 1 byte carryForward
		return fmt.Errorf("insufficient data for fund reward operation")
	}

	// Extract parameters from data
	rewardIndex := binary.LittleEndian.Uint64(data[1:9])
	amount := binary.LittleEndian.Uint64(data[9:17])
	carryForward := data[17] != 0

	// Extract accounts
	lbPair := instruction.Accounts[0]
	// These accounts used for reference when tracking transfers
	// rewardVault := instruction.Accounts[1]
	// rewardMint := instruction.Accounts[2]
	// funderTokenAccount := instruction.Accounts[3]
	funder := instruction.Accounts[4]

	// Get pair, reward, and wallet information
	pairID, err := p.getPairID(ctx, lbPair)
	if err != nil {
		return err
	}

	rewardID, err := p.getRewardID(ctx, lbPair, rewardIndex)
	if err != nil {
		return err
	}

	walletID, err := p.getWalletID(ctx, funder)
	if err != nil {
		return err
	}

	// Create the reward funding model
	funding := models.MeteoraRewardFunding{
		TransactionID: txModel.ID,
		RewardID:      rewardID,
		PairID:        pairID,
		WalletID:      walletID,
		Funder:        funder,
		Amount:        amount,
		CarryForward:  carryForward,
		FundTime:      txModel.BlockTime,
	}

	// Add reward funding to transaction
	txModel.RewardFundings = append(txModel.RewardFundings, funding)

	return nil
}

// parseInitializePosition parses an initializePosition instruction
func (p *TransactionParser) parseInitializePosition(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	// This is a placeholder for actually creating position records
	// In a complete implementation, this would create the position in the database
	// and return without adding anything to the transaction model
	return nil
}

// parseClosePosition parses a closePosition instruction
func (p *TransactionParser) parseClosePosition(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	// This is a placeholder for actually closing position records
	// In a complete implementation, this would update the position status in the database
	// and return without adding anything to the transaction model
	return nil
}

// parseInitializePair parses an initializePair instruction
func (p *TransactionParser) parseInitializePair(ctx context.Context, instruction Instruction, data []byte, txModel *models.Transaction) error {
	// This is a placeholder for actually creating pair records
	// In a complete implementation, this would create the pair in the database
	// and return without adding anything to the transaction model
	return nil
}

// Helper functions for database lookups

// getPairID looks up or creates a pair record
func (p *TransactionParser) getPairID(ctx context.Context, pairAddress string) (uint, error) {
	// Placeholder for database lookup
	// In a complete implementation, this would look up the pair by address
	// or create it if it doesn't exist
	return 1, nil
}

// getPositionID looks up or creates a position record
func (p *TransactionParser) getPositionID(ctx context.Context, positionAddress string) (uint, error) {
	// Placeholder for database lookup
	// In a complete implementation, this would look up the position by address
	// or create it if it doesn't exist
	return 1, nil
}

// getWalletID looks up or creates a wallet record
func (p *TransactionParser) getWalletID(ctx context.Context, walletAddress string) (uint, error) {
	// Placeholder for database lookup
	// In a complete implementation, this would look up the wallet by address
	// or create it if it doesn't exist
	return 1, nil
}

// getRewardID looks up or creates a reward record
func (p *TransactionParser) getRewardID(ctx context.Context, pairAddress string, rewardIndex uint64) (uint, error) {
	// Placeholder for database lookup
	// In a complete implementation, this would look up the reward by pair and index
	// or create it if it doesn't exist
	return 1, nil
}

// isTokenXToY determines if the swap is from token X to token Y
func (p *TransactionParser) isTokenXToY(ctx context.Context, tokenAccount string, tokenXMint string) (bool, error) {
	// Placeholder for token account lookup
	// In a complete implementation, this would determine if the token account is for token X
	return true, nil
}

// isXToken determines if a token mint is token X for a pair
func (p *TransactionParser) isXToken(ctx context.Context, tokenMint string, pairAddress string) (bool, error) {
	// Placeholder for pair token lookup
	// In a complete implementation, this would determine if the token is X or Y for the pair
	return true, nil
}

// UnixTimeToTime converts a Unix timestamp to a Time
func UnixTimeToTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}
