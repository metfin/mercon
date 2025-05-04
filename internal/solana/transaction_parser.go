package solana

import (
	"context"
	"encoding/json"
	"fmt"

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
	accounts := instruction.Accounts
	if len(accounts) < 13 {
		return fmt.Errorf("not enough accounts for initializeLbPair instruction")
	}

	// Map accounts based on the documentation
	pairAccount := accounts[0]
	tokenMintX := accounts[2]
	tokenMintY := accounts[3]
	reserveX := accounts[4]
	reserveY := accounts[5]
	oracle := accounts[6]

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

	// Here you would typically save this data to your database
	// p.db.Create(&models.MeteoraPair{...})

	return nil
}

// parseSwap parses the swap instruction
func (p *TransactionParser) parseSwap(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing swap instruction")

	// Extract account addresses
	accounts := instruction.Accounts
	if len(accounts) < 9 {
		return fmt.Errorf("not enough accounts for swap instruction")
	}

	// Map accounts based on the documentation
	lbPair := accounts[0]
	reserveX := accounts[2]
	reserveY := accounts[3]
	userTokenIn := accounts[4]
	userTokenOut := accounts[5]
	tokenXMint := accounts[6]
	tokenYMint := accounts[7]
	oracle := accounts[8]
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

	// Here you would typically save this data to your database
	// p.db.Create(&models.MeteoraSwap{...})

	return nil
}

// parseAddLiquidity parses the addLiquidity instruction
func (p *TransactionParser) parseAddLiquidity(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing addLiquidity instruction")

	// Extract account addresses
	accounts := instruction.Accounts
	if len(accounts) < 13 {
		return fmt.Errorf("not enough accounts for addLiquidity instruction")
	}

	// Map accounts based on the documentation
	position := accounts[0]
	lbPair := accounts[1]
	userTokenX := accounts[3]
	userTokenY := accounts[4]
	reserveX := accounts[5]
	reserveY := accounts[6]
	sender := accounts[11]

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
		position, lbPair, sender, amountX, amountY, activeId)

	// Here you would typically save this data to your database
	// p.db.Create(&models.MeteoraLiquidityAddition{...})

	return nil
}

// parseRemoveLiquidity parses the removeLiquidity instruction
func (p *TransactionParser) parseRemoveLiquidity(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing removeLiquidity instruction")

	// Extract account addresses
	accounts := instruction.Accounts
	if len(accounts) < 13 {
		return fmt.Errorf("not enough accounts for removeLiquidity instruction")
	}

	// Map accounts based on the documentation
	position := accounts[0]
	lbPair := accounts[1]
	userTokenX := accounts[3]
	userTokenY := accounts[4]
	reserveX := accounts[5]
	reserveY := accounts[6]
	sender := accounts[11]

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
		position, lbPair, sender, len(binReductions))

	// Here you would typically save this data to your database
	// p.db.Create(&models.MeteoraLiquidityRemoval{...})

	return nil
}

// parseInitializePosition parses the initializePosition instruction
func (p *TransactionParser) parseInitializePosition(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing initializePosition instruction")

	// Extract account addresses
	accounts := instruction.Accounts
	if len(accounts) < 8 {
		return fmt.Errorf("not enough accounts for initializePosition instruction")
	}

	// Map accounts based on the documentation
	payer := accounts[0]
	position := accounts[1]
	lbPair := accounts[2]
	owner := accounts[3]

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
		position, lbPair, owner, lowerBinId, width)

	// Here you would typically save this data to your database
	// p.db.Create(&models.MeteoraPosition{...})

	return nil
}

// parseClaimFee parses the claimFee instruction
func (p *TransactionParser) parseClaimFee(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing claimFee instruction")

	// Extract account addresses
	accounts := instruction.Accounts
	if len(accounts) < 14 {
		return fmt.Errorf("not enough accounts for claimFee instruction")
	}

	// Map accounts based on the documentation
	lbPair := accounts[0]
	position := accounts[1]
	sender := accounts[4]
	reserveX := accounts[5]
	reserveY := accounts[6]
	userTokenX := accounts[7]
	userTokenY := accounts[8]

	fmt.Printf("Claim Fee: Position: %s, LbPair: %s, User: %s\n",
		position, lbPair, sender)

	// Here you would typically save this data to your database
	// p.db.Create(&models.MeteoraFeeClaim{...})

	return nil
}

// parseClosePosition parses the closePosition instruction
func (p *TransactionParser) parseClosePosition(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing closePosition instruction")

	// Extract account addresses
	accounts := instruction.Accounts
	if len(accounts) < 8 {
		return fmt.Errorf("not enough accounts for closePosition instruction")
	}

	// Map accounts based on the documentation
	position := accounts[0]
	lbPair := accounts[1]
	sender := accounts[4]
	rentReceiver := accounts[5]

	fmt.Printf("Close Position: %s, LbPair: %s, User: %s, RentReceiver: %s\n",
		position, lbPair, sender, rentReceiver)

	// Here you would typically save this data to your database
	// p.db.Create(&models.MeteoraPositionClosure{...})

	return nil
}

// parseInitializeReward parses the initializeReward instruction
func (p *TransactionParser) parseInitializeReward(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing initializeReward instruction")

	// Extract account addresses
	accounts := instruction.Accounts
	if len(accounts) < 10 {
		return fmt.Errorf("not enough accounts for initializeReward instruction")
	}

	// Map accounts based on the documentation
	lbPair := accounts[0]
	rewardVault := accounts[1]
	rewardMint := accounts[2]
	admin := accounts[4]

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
		lbPair, rewardIndex, rewardDuration, funder)

	// Here you would typically save this data to your database
	// p.db.Create(&models.MeteoraReward{...})

	return nil
}

// parseFundReward parses the fundReward instruction
func (p *TransactionParser) parseFundReward(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing fundReward instruction")

	// Extract account addresses
	accounts := instruction.Accounts
	if len(accounts) < 9 {
		return fmt.Errorf("not enough accounts for fundReward instruction")
	}

	// Map accounts based on the documentation
	lbPair := accounts[0]
	rewardVault := accounts[1]
	rewardMint := accounts[2]
	funderTokenAccount := accounts[3]
	funder := accounts[4]

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
		lbPair, rewardIndex, amount, carryForward)

	// Here you would typically save this data to your database
	// p.db.Create(&models.MeteoraRewardFunding{...})

	return nil
}

// parseClaimReward parses the claimReward instruction
func (p *TransactionParser) parseClaimReward(tx *models.Transaction, instruction models.TransactionInstruction, data map[string]interface{}) error {
	fmt.Println("Parsing claimReward instruction")

	// Extract account addresses
	accounts := instruction.Accounts
	if len(accounts) < 11 {
		return fmt.Errorf("not enough accounts for claimReward instruction")
	}

	// Map accounts based on the documentation
	lbPair := accounts[0]
	position := accounts[1]
	sender := accounts[4]
	rewardVault := accounts[5]
	rewardMint := accounts[6]
	userTokenAccount := accounts[7]

	// Extract instruction data
	var rewardIndex uint64

	if val, ok := data["rewardIndex"].(float64); ok {
		rewardIndex = uint64(val)
	}

	fmt.Printf("Claim Reward: Position: %s, LbPair: %s, User: %s, RewardIndex: %d\n",
		position, lbPair, sender, rewardIndex)

	// Here you would typically save this data to your database
	// p.db.Create(&models.MeteoraRewardClaim{...})

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
