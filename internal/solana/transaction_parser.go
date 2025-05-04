package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wnt/mercon/internal/models"
	"gorm.io/gorm"
)

// Constants for program IDs
const (
	MeteoraProgram = "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo"
)

// TransactionParser handles parsing and saving transaction data
type TransactionParser struct {
	db     *gorm.DB
	client *Client
}

// NewTransactionParser creates a new transaction parser
func NewTransactionParser(db *gorm.DB, client *Client) *TransactionParser {
	return &TransactionParser{
		db:     db,
		client: client,
	}
}

// ProcessTransaction parses a transaction and stores it in the database
func (p *TransactionParser) ProcessTransaction(ctx context.Context, tx *models.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction data is nil")
	}

	// Check if any of the instructions are a Meteora DLMM transaction.
	meteoraInstructions := []models.TransactionInstruction{}
	for _, instruction := range tx.Instructions {
		if instruction.ProgramID == MeteoraProgram {
			meteoraInstructions = append(meteoraInstructions, instruction)
		}
	}

	if len(meteoraInstructions) == 0 {
		return nil
	}

	fmt.Println("Transaction: ", tx.Signature)

	// Loop through the instructions, find out the type of the instruction, and parse it accordingly
	for _, instruction := range meteoraInstructions {
		// Unmarshal the instruction data
		var instructionData map[string]interface{}
		err := json.Unmarshal([]byte(instruction.Data), &instructionData)
		if err != nil {
			return fmt.Errorf("failed to unmarshal instruction data: %w", err)
		}

		// Extract instruction type from discriminator (first field in Anchor instruction data)
		instructionType, err := p.getInstructionType(instructionData)
		if err != nil {
			return fmt.Errorf("failed to determine instruction type: %w", err)
		}

		fmt.Printf("Instruction Type: %s\n", instructionType)

		// Parse instruction data based on the instruction type
		switch instructionType {
		case "initializeLbPair":
			err = p.parseInitializeLbPair(tx, instruction, instructionData)
		case "swap":
			err = p.parseSwap(tx, instruction, instructionData)
		case "addLiquidity":
			err = p.parseAddLiquidity(tx, instruction, instructionData)
		case "removeLiquidity":
			err = p.parseRemoveLiquidity(tx, instruction, instructionData)
		case "initializePosition":
			err = p.parseInitializePosition(tx, instruction, instructionData)
		case "claimFee":
			err = p.parseClaimFee(tx, instruction, instructionData)
		case "closePosition":
			err = p.parseClosePosition(tx, instruction, instructionData)
		case "initializeReward":
			err = p.parseInitializeReward(tx, instruction, instructionData)
		case "fundReward":
			err = p.parseFundReward(tx, instruction, instructionData)
		case "claimReward":
			err = p.parseClaimReward(tx, instruction, instructionData)
		default:
			fmt.Printf("Unhandled instruction type: %s\n", instructionType)
		}

		if err != nil {
			return fmt.Errorf("failed to parse %s instruction: %w", instructionType, err)
		}
	}

	return nil
}

// extractAccounts extracts account addresses from an instruction
func extractAccounts(instruction models.TransactionInstruction) ([]string, error) {
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal accounts: %w", err)
	}
	return accounts, nil
}

// findOrCreateWallet finds or creates a wallet and returns its ID
func (p *TransactionParser) findOrCreateWallet(address string) (uint, error) {
	var wallet models.Wallet
	if err := p.db.Where("address = ?", address).FirstOrCreate(&wallet, models.Wallet{Address: address}).Error; err != nil {
		return 0, fmt.Errorf("failed to find/create wallet: %w", err)
	}
	return wallet.ID, nil
}

// findPair finds a Meteora pair by address and returns it
func (p *TransactionParser) findPair(address string) (models.MeteoraPair, error) {
	var pair models.MeteoraPair
	if err := p.db.Where("address = ?", address).First(&pair).Error; err != nil {
		return pair, fmt.Errorf("failed to find Meteora pair: %w", err)
	}
	return pair, nil
}

// findPosition finds a Meteora position by address and returns it
func (p *TransactionParser) findPosition(address string) (models.MeteoraPosition, error) {
	var position models.MeteoraPosition
	if err := p.db.Where("address = ?", address).First(&position).Error; err != nil {
		return position, fmt.Errorf("failed to find Meteora position: %w", err)
	}
	return position, nil
}

// findReward finds a Meteora reward by pair and index
func (p *TransactionParser) findReward(pairID uint, rewardIndex uint64) (models.MeteoraReward, error) {
	var reward models.MeteoraReward
	if err := p.db.Where("pair_id = ? AND reward_index = ?", pairID, rewardIndex).First(&reward).Error; err != nil {
		return reward, fmt.Errorf("failed to find Meteora reward: %w", err)
	}
	return reward, nil
}

// extractUint64 extracts a uint64 value from data
func extractUint64(data map[string]interface{}, key string) uint64 {
	var value uint64
	if val, ok := data[key]; ok {
		if strVal, ok := val.(string); ok {
			fmt.Sscanf(strVal, "%d", &value)
		} else if floatVal, ok := val.(float64); ok {
			value = uint64(floatVal)
		}
	}
	return value
}

// extractInt32 extracts an int32 value from data
func extractInt32(data map[string]interface{}, key string) int32 {
	var value int32
	if val, ok := data[key]; ok {
		if floatVal, ok := val.(float64); ok {
			value = int32(floatVal)
		}
	}
	return value
}

// extractUint16 extracts a uint16 value from data
func extractUint16(data map[string]interface{}, key string) uint16 {
	var value uint16
	if val, ok := data[key]; ok {
		if floatVal, ok := val.(float64); ok {
			value = uint16(floatVal)
		}
	}
	return value
}

// extractBool extracts a boolean value from data
func extractBool(data map[string]interface{}, key string) bool {
	var value bool
	if val, ok := data[key]; ok {
		if boolVal, ok := val.(bool); ok {
			value = boolVal
		}
	}
	return value
}

// getInstructionType determines the Meteora DLMM instruction type
func (p *TransactionParser) getInstructionType(instructionData map[string]interface{}) (string, error) {
	// In Anchor-based programs, the instruction discriminator is the first field
	// This is a simplified approach

	// Common instruction types from Meteora DLMM
	instructionTypes := map[string]string{
		"0":  "initializeLbPair",
		"1":  "initializePermissionLbPair",
		"2":  "initializeCustomizablePermissionlessLbPair",
		"3":  "initializeBinArrayBitmapExtension",
		"4":  "initializeBinArray",
		"5":  "addLiquidity",
		"6":  "addLiquidityByWeight",
		"7":  "addLiquidityByStrategy",
		"8":  "addLiquidityByStrategyOneSide",
		"9":  "addLiquidityOneSide",
		"10": "removeLiquidity",
		"11": "initializePosition",
		"12": "initializePositionPda",
		"13": "initializePositionByOperator",
		"14": "updatePositionOperator",
		"15": "swap",
		"16": "swapExactOut",
		"17": "swapWithPriceImpact",
		"18": "withdrawProtocolFee",
		"19": "initializeReward",
		"20": "fundReward",
		"21": "updateRewardFunder",
		"22": "updateRewardDuration",
		"23": "claimReward",
		"24": "claimFee",
		"25": "closePosition",
		// More instruction types can be added here
	}

	// Check if discriminator exists in the data
	if disc, ok := instructionData["discriminator"]; ok {
		if discStr, ok := disc.(string); ok {
			if instType, ok := instructionTypes[discStr]; ok {
				return instType, nil
			}
		}
	}

	// If we can't determine the type specifically, try to infer from data fields
	if amountIn, ok := instructionData["amountIn"]; ok {
		if _, ok := instructionData["minAmountOut"]; ok {
			if _, ok := amountIn.(float64); ok {
				return "swap", nil
			}
		}
	}

	if _, ok := instructionData["liquidityParameter"]; ok {
		return "addLiquidity", nil
	}

	if _, ok := instructionData["binLiquidityRemoval"]; ok {
		return "removeLiquidity", nil
	}

	return "unknown", fmt.Errorf("could not determine instruction type from data")
}

// parseInitializeLbPair parses the initializeLbPair instruction
func (p *TransactionParser) parseInitializeLbPair(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing initializeLbPair instruction")

	// Extract account addresses
	accounts, err := extractAccounts(instruction)
	if err != nil {
		return err
	}

	if len(accounts) < 13 {
		return fmt.Errorf("not enough accounts for initializeLbPair instruction")
	}

	// Map accounts based on the documentation
	pairAccount := accounts[0]
	tokenMintX := accounts[2]
	tokenMintY := accounts[3]

	// Extract instruction data
	activeId := extractInt32(data, "activeId")
	binStep := extractUint16(data, "binStep")

	fmt.Printf("LB Pair Created: %s, TokenX: %s, TokenY: %s, ActiveID: %d, BinStep: %d\n",
		pairAccount, tokenMintX, tokenMintY, activeId, binStep)

	// Save to database
	meteoraPair := &models.MeteoraPair{
		Address:    pairAccount,
		TokenMintX: tokenMintX,
		TokenMintY: tokenMintY,
		ActiveID:   activeId,
		BinStep:    binStep,
	}

	result := p.db.Create(meteoraPair)
	if result.Error != nil {
		return fmt.Errorf("failed to save Meteora pair: %w", result.Error)
	}

	return nil
}

// parseSwap parses the swap instruction
func (p *TransactionParser) parseSwap(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing swap instruction")

	// Extract account addresses
	accounts, err := extractAccounts(instruction)
	if err != nil {
		return err
	}

	if len(accounts) < 11 {
		return fmt.Errorf("not enough accounts for swap instruction")
	}

	// Map accounts based on the documentation
	lbPair := accounts[0]
	tokenInMint := accounts[6]
	tokenOutMint := accounts[7]
	user := accounts[10]

	// Extract instruction data
	amountIn := extractUint64(data, "amountIn")
	minAmountOut := extractUint64(data, "minAmountOut")

	fmt.Printf("Swap: %s, User: %s, AmountIn: %d, MinAmountOut: %d\n",
		lbPair, user, amountIn, minAmountOut)

	// Find pair and wallet
	pair, err := p.findPair(lbPair)
	if err != nil {
		return err
	}

	walletID, err := p.findOrCreateWallet(user)
	if err != nil {
		return err
	}

	// Save to database
	meteoraSwap := &models.MeteoraSwap{
		TransactionID: tx.ID,
		PairID:        pair.ID,
		WalletID:      walletID,
		User:          user,
		TokenInMint:   tokenInMint,
		TokenOutMint:  tokenOutMint,
		AmountIn:      amountIn,
		MinAmountOut:  minAmountOut,
		SwapTime:      tx.BlockTime,
	}

	result := p.db.Create(meteoraSwap)
	if result.Error != nil {
		return fmt.Errorf("failed to save Meteora swap: %w", result.Error)
	}

	return nil
}

// parseAddLiquidity parses the addLiquidity instruction
func (p *TransactionParser) parseAddLiquidity(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing addLiquidity instruction")

	// Extract account addresses
	accounts, err := extractAccounts(instruction)
	if err != nil {
		return err
	}

	if len(accounts) < 13 {
		return fmt.Errorf("not enough accounts for addLiquidity instruction")
	}

	// Map accounts based on the documentation
	positionAddr := accounts[0]
	lbPairAddr := accounts[1]
	senderAddr := accounts[11]

	// Extract liquidity parameters
	var amountX, amountY uint64
	var activeId int32

	if liquidityParam, ok := data["liquidityParameter"].(map[string]interface{}); ok {
		if amount, ok := liquidityParam["amount"].(map[string]interface{}); ok {
			if xVal, ok := amount["x"].(string); ok {
				fmt.Sscanf(xVal, "%d", &amountX)
			}
			if yVal, ok := amount["y"].(string); ok {
				fmt.Sscanf(yVal, "%d", &amountY)
			}
		}

		if val, ok := liquidityParam["activeId"].(float64); ok {
			activeId = int32(val)
		}
	}

	fmt.Printf("Add Liquidity: Position: %s, LbPair: %s, User: %s, AmountX: %d, AmountY: %d, ActiveID: %d\n",
		positionAddr, lbPairAddr, senderAddr, amountX, amountY, activeId)

	// Find pair, position and wallet
	pair, err := p.findPair(lbPairAddr)
	if err != nil {
		return err
	}

	position, err := p.findPosition(positionAddr)
	if err != nil {
		return err
	}

	walletID, err := p.findOrCreateWallet(senderAddr)
	if err != nil {
		return err
	}

	// Save to database
	liquidityAddition := &models.MeteoraLiquidityAddition{
		TransactionID: tx.ID,
		PositionID:    position.ID,
		PairID:        pair.ID,
		WalletID:      walletID,
		User:          senderAddr,
		AmountX:       amountX,
		AmountY:       amountY,
		ActiveID:      activeId,
		AddTime:       tx.BlockTime,
	}

	result := p.db.Create(liquidityAddition)
	if result.Error != nil {
		return fmt.Errorf("failed to save Meteora liquidity addition: %w", result.Error)
	}

	return nil
}

// parseRemoveLiquidity parses the removeLiquidity instruction
func (p *TransactionParser) parseRemoveLiquidity(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing removeLiquidity instruction")

	// Extract account addresses
	accounts, err := extractAccounts(instruction)
	if err != nil {
		return err
	}

	if len(accounts) < 13 {
		return fmt.Errorf("not enough accounts for removeLiquidity instruction")
	}

	// Map accounts based on the documentation
	positionAddr := accounts[0]
	lbPairAddr := accounts[1]
	senderAddr := accounts[11]

	// Extract bin liquidity reductions
	type BinReduction struct {
		BinID  int32
		Amount uint64
	}
	var binReductions []BinReduction

	if reductions, ok := data["binLiquidityRemoval"].([]interface{}); ok {
		for _, red := range reductions {
			redMap, mapOk := red.(map[string]interface{})
			if !mapOk {
				continue
			}

			var binRed BinReduction

			if binIdVal, ok := redMap["binId"]; ok {
				if binIdFloat, ok := binIdVal.(float64); ok {
					binRed.BinID = int32(binIdFloat)
				}
			}

			if amountVal, ok := redMap["liquidityAmount"]; ok {
				if amountStr, ok := amountVal.(string); ok {
					fmt.Sscanf(amountStr, "%d", &binRed.Amount)
				}
			}

			binReductions = append(binReductions, binRed)
		}
	}

	fmt.Printf("Remove Liquidity: Position: %s, LbPair: %s, User: %s, Bins: %d\n",
		positionAddr, lbPairAddr, senderAddr, len(binReductions))

	// Find pair, position and wallet
	pair, err := p.findPair(lbPairAddr)
	if err != nil {
		return err
	}

	position, err := p.findPosition(positionAddr)
	if err != nil {
		return err
	}

	walletID, err := p.findOrCreateWallet(senderAddr)
	if err != nil {
		return err
	}

	// Convert bin reductions to JSON
	binReductionsJSON, err := json.Marshal(binReductions)
	if err != nil {
		return fmt.Errorf("failed to marshal bin reductions: %w", err)
	}

	// Save to database
	liquidityRemoval := &models.MeteoraLiquidityRemoval{
		TransactionID: tx.ID,
		PositionID:    position.ID,
		PairID:        pair.ID,
		WalletID:      walletID,
		User:          senderAddr,
		RemoveTime:    tx.BlockTime,
		BinReductions: string(binReductionsJSON),
	}

	result := p.db.Create(liquidityRemoval)
	if result.Error != nil {
		return fmt.Errorf("failed to save Meteora liquidity removal: %w", result.Error)
	}

	return nil
}

// parseInitializePosition parses the initializePosition instruction
func (p *TransactionParser) parseInitializePosition(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing initializePosition instruction")

	// Extract account addresses
	accounts, err := extractAccounts(instruction)
	if err != nil {
		return err
	}

	if len(accounts) < 8 {
		return fmt.Errorf("not enough accounts for initializePosition instruction")
	}

	// Map accounts based on the documentation
	positionAddr := accounts[1]
	lbPairAddr := accounts[2]
	ownerAddr := accounts[3]

	// Extract instruction data
	lowerBinId := extractInt32(data, "lowerBinId")
	width := extractInt32(data, "width")

	fmt.Printf("Initialize Position: %s, LbPair: %s, Owner: %s, LowerBinID: %d, Width: %d\n",
		positionAddr, lbPairAddr, ownerAddr, lowerBinId, width)

	// Find pair and wallet
	pair, err := p.findPair(lbPairAddr)
	if err != nil {
		return err
	}

	walletID, err := p.findOrCreateWallet(ownerAddr)
	if err != nil {
		return err
	}

	// Save to database
	meteoraPosition := &models.MeteoraPosition{
		Address:    positionAddr,
		PairID:     pair.ID,
		WalletID:   walletID,
		Owner:      ownerAddr,
		LowerBinID: lowerBinId,
		Width:      width,
		CreatedAt:  tx.BlockTime,
	}

	result := p.db.Create(meteoraPosition)
	if result.Error != nil {
		return fmt.Errorf("failed to save Meteora position: %w", result.Error)
	}

	return nil
}

// parseClaimFee parses the claimFee instruction
func (p *TransactionParser) parseClaimFee(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing claimFee instruction")

	// Extract account addresses
	accounts, err := extractAccounts(instruction)
	if err != nil {
		return err
	}

	if len(accounts) < 14 {
		return fmt.Errorf("not enough accounts for claimFee instruction")
	}

	// Map accounts based on the documentation
	lbPairAddr := accounts[0]
	positionAddr := accounts[1]
	senderAddr := accounts[4]

	fmt.Printf("Claim Fee: Position: %s, LbPair: %s, User: %s\n",
		positionAddr, lbPairAddr, senderAddr)

	// Find pair, position and wallet
	pair, err := p.findPair(lbPairAddr)
	if err != nil {
		return err
	}

	position, err := p.findPosition(positionAddr)
	if err != nil {
		return err
	}

	walletID, err := p.findOrCreateWallet(senderAddr)
	if err != nil {
		return err
	}

	// Save to database
	feeClaim := &models.MeteoraFeeClaim{
		TransactionID: tx.ID,
		PositionID:    position.ID,
		PairID:        pair.ID,
		WalletID:      walletID,
		User:          senderAddr,
		ClaimTime:     tx.BlockTime,
	}

	result := p.db.Create(feeClaim)
	if result.Error != nil {
		return fmt.Errorf("failed to save Meteora fee claim: %w", result.Error)
	}

	return nil
}

// parseClosePosition parses the closePosition instruction
func (p *TransactionParser) parseClosePosition(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing closePosition instruction")

	// Extract account addresses
	accounts, err := extractAccounts(instruction)
	if err != nil {
		return err
	}

	if len(accounts) < 8 {
		return fmt.Errorf("not enough accounts for closePosition instruction")
	}

	// Map accounts based on the documentation
	positionAddr := accounts[0]
	lbPairAddr := accounts[1]
	senderAddr := accounts[4]
	rentReceiver := accounts[5]

	fmt.Printf("Close Position: %s, LbPair: %s, User: %s, RentReceiver: %s\n",
		positionAddr, lbPairAddr, senderAddr, rentReceiver)

	// Look up the position in the database
	position, err := p.findPosition(positionAddr)
	if err != nil {
		return err
	}

	// Update the position's status to closed
	now := time.Now()
	position.Status = "closed"
	position.ClosedAt = &now
	if err := p.db.Save(&position).Error; err != nil {
		return fmt.Errorf("failed to update position status: %w", err)
	}

	return nil
}

// parseInitializeReward parses the initializeReward instruction
func (p *TransactionParser) parseInitializeReward(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing initializeReward instruction")

	// Extract account addresses
	accounts, err := extractAccounts(instruction)
	if err != nil {
		return err
	}

	if len(accounts) < 10 {
		return fmt.Errorf("not enough accounts for initializeReward instruction")
	}

	// Map accounts based on the documentation
	lbPairAddr := accounts[0]
	rewardVault := accounts[1]
	rewardMint := accounts[2]

	// Extract instruction data
	rewardIndex := extractUint64(data, "rewardIndex")
	rewardDuration := extractUint64(data, "rewardDuration")

	var funder string
	if val, ok := data["funder"].(string); ok {
		funder = val
	}

	fmt.Printf("Initialize Reward: LbPair: %s, RewardIndex: %d, Duration: %d, Funder: %s\n",
		lbPairAddr, rewardIndex, rewardDuration, funder)

	// Find pair
	pair, err := p.findPair(lbPairAddr)
	if err != nil {
		return err
	}

	// Calculate start and end times
	startTime := tx.BlockTime
	endTime := startTime.Add(time.Duration(rewardDuration) * time.Second)

	// Save to database
	reward := &models.MeteoraReward{
		PairID:         pair.ID,
		RewardIndex:    rewardIndex,
		RewardVault:    rewardVault,
		RewardMint:     rewardMint,
		Funder:         funder,
		RewardDuration: rewardDuration,
		StartTime:      startTime,
		EndTime:        endTime,
	}

	result := p.db.Create(reward)
	if result.Error != nil {
		return fmt.Errorf("failed to save Meteora reward: %w", result.Error)
	}

	return nil
}

// parseFundReward parses the fundReward instruction
func (p *TransactionParser) parseFundReward(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing fundReward instruction")

	// Extract account addresses
	accounts, err := extractAccounts(instruction)
	if err != nil {
		return err
	}

	if len(accounts) < 9 {
		return fmt.Errorf("not enough accounts for fundReward instruction")
	}

	// Map accounts based on the documentation
	lbPairAddr := accounts[0]
	funderAddr := accounts[4]

	// Extract instruction data
	rewardIndex := extractUint64(data, "rewardIndex")
	amount := extractUint64(data, "amount")
	carryForward := extractBool(data, "carryForward")

	fmt.Printf("Fund Reward: LbPair: %s, RewardIndex: %d, Amount: %d, CarryForward: %v\n",
		lbPairAddr, rewardIndex, amount, carryForward)

	// Find pair, reward and wallet
	pair, err := p.findPair(lbPairAddr)
	if err != nil {
		return err
	}

	reward, err := p.findReward(pair.ID, rewardIndex)
	if err != nil {
		return err
	}

	walletID, err := p.findOrCreateWallet(funderAddr)
	if err != nil {
		return err
	}

	// Save to database
	rewardFunding := &models.MeteoraRewardFunding{
		TransactionID: tx.ID,
		RewardID:      reward.ID,
		PairID:        pair.ID,
		WalletID:      walletID,
		Funder:        funderAddr,
		Amount:        amount,
		CarryForward:  carryForward,
		FundTime:      tx.BlockTime,
	}

	result := p.db.Create(rewardFunding)
	if result.Error != nil {
		return fmt.Errorf("failed to save Meteora reward funding: %w", result.Error)
	}

	return nil
}

// parseClaimReward parses the claimReward instruction
func (p *TransactionParser) parseClaimReward(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing claimReward instruction")

	// Extract account addresses
	accounts, err := extractAccounts(instruction)
	if err != nil {
		return err
	}

	if len(accounts) < 11 {
		return fmt.Errorf("not enough accounts for claimReward instruction")
	}

	// Map accounts based on the documentation
	lbPairAddr := accounts[0]
	positionAddr := accounts[1]
	senderAddr := accounts[4]

	// Extract instruction data
	rewardIndex := extractUint64(data, "rewardIndex")

	fmt.Printf("Claim Reward: Position: %s, LbPair: %s, User: %s, RewardIndex: %d\n",
		positionAddr, lbPairAddr, senderAddr, rewardIndex)

	// Find pair, position, reward and wallet
	pair, err := p.findPair(lbPairAddr)
	if err != nil {
		return err
	}

	position, err := p.findPosition(positionAddr)
	if err != nil {
		return err
	}

	reward, err := p.findReward(pair.ID, rewardIndex)
	if err != nil {
		return err
	}

	walletID, err := p.findOrCreateWallet(senderAddr)
	if err != nil {
		return err
	}

	// Save to database
	rewardClaim := &models.MeteoraRewardClaim{
		TransactionID: tx.ID,
		PositionID:    position.ID,
		RewardID:      reward.ID,
		PairID:        pair.ID,
		WalletID:      walletID,
		User:          senderAddr,
		ClaimTime:     tx.BlockTime,
	}

	result := p.db.Create(rewardClaim)
	if result.Error != nil {
		return fmt.Errorf("failed to save Meteora reward claim: %w", result.Error)
	}

	return nil
}
