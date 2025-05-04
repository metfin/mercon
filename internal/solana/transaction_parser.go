package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/wnt/mercon/internal/models"
	"gorm.io/gorm"
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
		if instruction.ProgramID == "LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo" {
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
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	if len(accounts) < 13 {
		return fmt.Errorf("not enough accounts for initializeLbPair instruction")
	}

	// Map accounts based on the documentation
	pairAccount := accounts[0]
	tokenMintX := accounts[2]
	tokenMintY := accounts[3]
	_ = accounts[4] // reserveX
	_ = accounts[5] // reserveY
	_ = accounts[6] // oracle

	// Extract instruction data
	var activeId int32
	var binStep uint16

	if val, ok := data["activeId"]; ok {
		if floatVal, ok := val.(float64); ok {
			activeId = int32(floatVal)
		}
	}

	if val, ok := data["binStep"]; ok {
		if floatVal, ok := val.(float64); ok {
			binStep = uint16(floatVal)
		}
	}

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
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	if len(accounts) < 11 {
		return fmt.Errorf("not enough accounts for swap instruction")
	}

	// Map accounts based on the documentation
	lbPair := accounts[0]
	_ = accounts[2] // reserveX
	_ = accounts[3] // reserveY
	_ = accounts[4] // userTokenIn
	_ = accounts[5] // userTokenOut
	tokenInMint := accounts[6]
	tokenOutMint := accounts[7]
	_ = accounts[8] // oracle
	user := accounts[10]

	// Extract instruction data
	var amountIn uint64
	var minAmountOut uint64

	if val, ok := data["amountIn"]; ok {
		if strVal, ok := val.(string); ok {
			fmt.Sscanf(strVal, "%d", &amountIn)
		} else if floatVal, ok := val.(float64); ok {
			amountIn = uint64(floatVal)
		}
	}

	if val, ok := data["minAmountOut"]; ok {
		if strVal, ok := val.(string); ok {
			fmt.Sscanf(strVal, "%d", &minAmountOut)
		} else if floatVal, ok := val.(float64); ok {
			minAmountOut = uint64(floatVal)
		}
	}

	fmt.Printf("Swap: %s, User: %s, AmountIn: %d, MinAmountOut: %d\n",
		lbPair, user, amountIn, minAmountOut)

	// Find or create the pair ID
	var pairID uint
	var pair models.MeteoraPair
	if err := p.db.Where("address = ?", lbPair).First(&pair).Error; err != nil {
		return fmt.Errorf("failed to find Meteora pair: %w", err)
	}
	pairID = pair.ID

	// Find or create the wallet ID
	var walletID uint
	var wallet models.Wallet
	if err := p.db.Where("address = ?", user).FirstOrCreate(&wallet, models.Wallet{Address: user}).Error; err != nil {
		return fmt.Errorf("failed to find/create wallet: %w", err)
	}
	walletID = wallet.ID

	// Save to database
	meteoraSwap := &models.MeteoraSwap{
		TransactionID: tx.ID,
		PairID:        pairID,
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
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	if len(accounts) < 13 {
		return fmt.Errorf("not enough accounts for addLiquidity instruction")
	}

	// Map accounts based on the documentation
	positionAddr := accounts[0]
	lbPairAddr := accounts[1]
	_ = accounts[3] // userTokenX
	_ = accounts[4] // userTokenY
	_ = accounts[5] // reserveX
	_ = accounts[6] // reserveY
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

	// Look up the pair and position in the database
	var pair models.MeteoraPair
	if err := p.db.Where("address = ?", lbPairAddr).First(&pair).Error; err != nil {
		return fmt.Errorf("failed to find Meteora pair: %w", err)
	}

	var position models.MeteoraPosition
	if err := p.db.Where("address = ?", positionAddr).First(&position).Error; err != nil {
		return fmt.Errorf("failed to find Meteora position: %w", err)
	}

	// Find or create the wallet
	var wallet models.Wallet
	if err := p.db.Where("address = ?", senderAddr).FirstOrCreate(&wallet, models.Wallet{Address: senderAddr}).Error; err != nil {
		return fmt.Errorf("failed to find/create wallet: %w", err)
	}

	// Save to database
	liquidityAddition := &models.MeteoraLiquidityAddition{
		TransactionID: tx.ID,
		PositionID:    position.ID,
		PairID:        pair.ID,
		WalletID:      wallet.ID,
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
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	if len(accounts) < 13 {
		return fmt.Errorf("not enough accounts for removeLiquidity instruction")
	}

	// Map accounts based on the documentation
	positionAddr := accounts[0]
	lbPairAddr := accounts[1]
	_ = accounts[3] // userTokenX
	_ = accounts[4] // userTokenY
	_ = accounts[5] // reserveX
	_ = accounts[6] // reserveY
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

	// Look up the pair and position in the database
	var pair models.MeteoraPair
	if err := p.db.Where("address = ?", lbPairAddr).First(&pair).Error; err != nil {
		return fmt.Errorf("failed to find Meteora pair: %w", err)
	}

	var position models.MeteoraPosition
	if err := p.db.Where("address = ?", positionAddr).First(&position).Error; err != nil {
		return fmt.Errorf("failed to find Meteora position: %w", err)
	}

	// Find or create the wallet
	var wallet models.Wallet
	if err := p.db.Where("address = ?", senderAddr).FirstOrCreate(&wallet, models.Wallet{Address: senderAddr}).Error; err != nil {
		return fmt.Errorf("failed to find/create wallet: %w", err)
	}

	// Save to database
	// Convert bin reductions to JSON
	binReductionsJSON, err := json.Marshal(binReductions)
	if err != nil {
		return fmt.Errorf("failed to marshal bin reductions: %w", err)
	}

	liquidityRemoval := &models.MeteoraLiquidityRemoval{
		TransactionID: tx.ID,
		PositionID:    position.ID,
		PairID:        pair.ID,
		WalletID:      wallet.ID,
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
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	if len(accounts) < 8 {
		return fmt.Errorf("not enough accounts for initializePosition instruction")
	}

	// Map accounts based on the documentation
	_ = accounts[0] // payer
	positionAddr := accounts[1]
	lbPairAddr := accounts[2]
	ownerAddr := accounts[3]

	// Extract instruction data
	var lowerBinId int32
	var width int32

	if val, ok := data["lowerBinId"].(float64); ok {
		lowerBinId = int32(val)
	}

	if val, ok := data["width"].(float64); ok {
		width = int32(val)
	}

	fmt.Printf("Initialize Position: %s, LbPair: %s, Owner: %s, LowerBinID: %d, Width: %d\n",
		positionAddr, lbPairAddr, ownerAddr, lowerBinId, width)

	// Look up the pair in the database
	var pair models.MeteoraPair
	if err := p.db.Where("address = ?", lbPairAddr).First(&pair).Error; err != nil {
		return fmt.Errorf("failed to find Meteora pair: %w", err)
	}

	// Find or create the wallet
	var wallet models.Wallet
	if err := p.db.Where("address = ?", ownerAddr).FirstOrCreate(&wallet, models.Wallet{Address: ownerAddr}).Error; err != nil {
		return fmt.Errorf("failed to find/create wallet: %w", err)
	}

	// Save to database
	meteoraPosition := &models.MeteoraPosition{
		Address:    positionAddr,
		PairID:     pair.ID,
		WalletID:   wallet.ID,
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
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	if len(accounts) < 14 {
		return fmt.Errorf("not enough accounts for claimFee instruction")
	}

	// Map accounts based on the documentation
	lbPairAddr := accounts[0]
	positionAddr := accounts[1]
	senderAddr := accounts[4]
	_ = accounts[5] // reserveX
	_ = accounts[6] // reserveY
	_ = accounts[7] // userTokenX
	_ = accounts[8] // userTokenY

	fmt.Printf("Claim Fee: Position: %s, LbPair: %s, User: %s\n",
		positionAddr, lbPairAddr, senderAddr)

	// Look up the pair and position in the database
	var pair models.MeteoraPair
	if err := p.db.Where("address = ?", lbPairAddr).First(&pair).Error; err != nil {
		return fmt.Errorf("failed to find Meteora pair: %w", err)
	}

	var position models.MeteoraPosition
	if err := p.db.Where("address = ?", positionAddr).First(&position).Error; err != nil {
		return fmt.Errorf("failed to find Meteora position: %w", err)
	}

	// Find or create the wallet
	var wallet models.Wallet
	if err := p.db.Where("address = ?", senderAddr).FirstOrCreate(&wallet, models.Wallet{Address: senderAddr}).Error; err != nil {
		return fmt.Errorf("failed to find/create wallet: %w", err)
	}

	// Save to database
	feeClaim := &models.MeteoraFeeClaim{
		TransactionID: tx.ID,
		PositionID:    position.ID,
		PairID:        pair.ID,
		WalletID:      wallet.ID,
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
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
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
	var position models.MeteoraPosition
	if err := p.db.Where("address = ?", positionAddr).First(&position).Error; err != nil {
		return fmt.Errorf("failed to find Meteora position: %w", err)
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
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	if len(accounts) < 10 {
		return fmt.Errorf("not enough accounts for initializeReward instruction")
	}

	// Map accounts based on the documentation
	lbPairAddr := accounts[0]
	rewardVault := accounts[1]
	rewardMint := accounts[2]
	_ = accounts[4] // admin

	// Extract instruction data
	var rewardIndex uint64
	var rewardDuration uint64
	var funder string

	if val, ok := data["rewardIndex"].(float64); ok {
		rewardIndex = uint64(val)
	}

	if val, ok := data["rewardDuration"].(float64); ok {
		rewardDuration = uint64(val)
	}

	if val, ok := data["funder"].(string); ok {
		funder = val
	}

	fmt.Printf("Initialize Reward: LbPair: %s, RewardIndex: %d, Duration: %d, Funder: %s\n",
		lbPairAddr, rewardIndex, rewardDuration, funder)

	// Look up the pair in the database
	var pair models.MeteoraPair
	if err := p.db.Where("address = ?", lbPairAddr).First(&pair).Error; err != nil {
		return fmt.Errorf("failed to find Meteora pair: %w", err)
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
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	if len(accounts) < 9 {
		return fmt.Errorf("not enough accounts for fundReward instruction")
	}

	// Map accounts based on the documentation
	lbPairAddr := accounts[0]
	_ = accounts[1] // rewardVault
	_ = accounts[2] // rewardMint
	_ = accounts[3] // funderTokenAccount
	funderAddr := accounts[4]

	// Extract instruction data
	var rewardIndex uint64
	var amount uint64
	var carryForward bool

	if val, ok := data["rewardIndex"].(float64); ok {
		rewardIndex = uint64(val)
	}

	if val, ok := data["amount"].(string); ok {
		fmt.Sscanf(val, "%d", &amount)
	} else if val, ok := data["amount"].(float64); ok {
		amount = uint64(val)
	}

	if val, ok := data["carryForward"].(bool); ok {
		carryForward = val
	}

	fmt.Printf("Fund Reward: LbPair: %s, RewardIndex: %d, Amount: %d, CarryForward: %v\n",
		lbPairAddr, rewardIndex, amount, carryForward)

	// Look up the pair in the database
	var pair models.MeteoraPair
	if err := p.db.Where("address = ?", lbPairAddr).First(&pair).Error; err != nil {
		return fmt.Errorf("failed to find Meteora pair: %w", err)
	}

	// Look up the reward in the database
	var reward models.MeteoraReward
	if err := p.db.Where("pair_id = ? AND reward_index = ?", pair.ID, rewardIndex).First(&reward).Error; err != nil {
		return fmt.Errorf("failed to find Meteora reward: %w", err)
	}

	// Find or create the wallet
	var wallet models.Wallet
	if err := p.db.Where("address = ?", funderAddr).FirstOrCreate(&wallet, models.Wallet{Address: funderAddr}).Error; err != nil {
		return fmt.Errorf("failed to find/create wallet: %w", err)
	}

	// Save to database
	rewardFunding := &models.MeteoraRewardFunding{
		TransactionID: tx.ID,
		RewardID:      reward.ID,
		PairID:        pair.ID,
		WalletID:      wallet.ID,
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
	var accounts []string
	err := json.Unmarshal([]byte(instruction.Accounts), &accounts)
	if err != nil {
		return fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	if len(accounts) < 11 {
		return fmt.Errorf("not enough accounts for claimReward instruction")
	}

	// Map accounts based on the documentation
	lbPairAddr := accounts[0]
	positionAddr := accounts[1]
	senderAddr := accounts[4]
	_ = accounts[5] // rewardVault
	_ = accounts[6] // rewardMint
	_ = accounts[7] // userTokenAccount

	// Extract instruction data
	var rewardIndex uint64

	if val, ok := data["rewardIndex"].(float64); ok {
		rewardIndex = uint64(val)
	}

	fmt.Printf("Claim Reward: Position: %s, LbPair: %s, User: %s, RewardIndex: %d\n",
		positionAddr, lbPairAddr, senderAddr, rewardIndex)

	// Look up the pair in the database
	var pair models.MeteoraPair
	if err := p.db.Where("address = ?", lbPairAddr).First(&pair).Error; err != nil {
		return fmt.Errorf("failed to find Meteora pair: %w", err)
	}

	// Look up the position in the database
	var position models.MeteoraPosition
	if err := p.db.Where("address = ?", positionAddr).First(&position).Error; err != nil {
		return fmt.Errorf("failed to find Meteora position: %w", err)
	}

	// Look up the reward in the database
	var reward models.MeteoraReward
	if err := p.db.Where("pair_id = ? AND reward_index = ?", pair.ID, rewardIndex).First(&reward).Error; err != nil {
		return fmt.Errorf("failed to find Meteora reward: %w", err)
	}

	// Find or create the wallet
	var wallet models.Wallet
	if err := p.db.Where("address = ?", senderAddr).FirstOrCreate(&wallet, models.Wallet{Address: senderAddr}).Error; err != nil {
		return fmt.Errorf("failed to find/create wallet: %w", err)
	}

	// Save to database
	rewardClaim := &models.MeteoraRewardClaim{
		TransactionID: tx.ID,
		PositionID:    position.ID,
		RewardID:      reward.ID,
		PairID:        pair.ID,
		WalletID:      wallet.ID,
		User:          senderAddr,
		ClaimTime:     tx.BlockTime,
	}

	result := p.db.Create(rewardClaim)
	if result.Error != nil {
		return fmt.Errorf("failed to save Meteora reward claim: %w", result.Error)
	}

	return nil
}

// parseInstructions extracts and stores instruction data
func (p *TransactionParser) parseInstructions(tx *gorm.DB, rpcTx *rpc.TransactionWithMeta, transactionID uint) error {

	return nil
}

// parseTokenTransfers extracts and stores token transfer information
func (p *TransactionParser) parseTokenTransfers(tx *gorm.DB, rpcTx *rpc.TransactionWithMeta, transactionID uint) error {
	return nil
}

// getProgramName returns a human-readable name for known program IDs
func getProgramName(programID string) string {
	knownPrograms := map[string]string{
		"11111111111111111111111111111111":            "System Program",
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA": "Token Program",
		"LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo": "Meteora DLMM Program",
		"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb": "Token-2022 Program",
	}

	if name, ok := knownPrograms[programID]; ok {
		return name
	}

	return "Unknown Program"
}
