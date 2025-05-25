package solana

type Transaction struct {
	Description      string            `json:"description"`
	Type             string            `json:"type"`
	Source           string            `json:"source"`
	Fee              int64             `json:"fee"`
	FeePayer         string            `json:"feePayer"`
	Signature        string            `json:"signature"`
	Slot             int64             `json:"slot"`
	Timestamp        int64             `json:"timestamp"`
	NativeTransfers  []NativeTransfer  `json:"nativeTransfers"`
	TokenTransfers   []TokenTransfer   `json:"tokenTransfers"`
	AccountData      []AccountData     `json:"accountData"`
	TransactionError *TransactionError `json:"transactionError"`
	Instructions     []Instruction     `json:"instructions"`
	Events           Events            `json:"events"`
}

type NativeTransfer struct {
	FromUserAccount string `json:"fromUserAccount"`
	ToUserAccount   string `json:"toUserAccount"`
	Amount          int64  `json:"amount"`
}

type TokenTransfer struct {
	FromUserAccount  string `json:"fromUserAccount"`
	ToUserAccount    string `json:"toUserAccount"`
	FromTokenAccount string `json:"fromTokenAccount"`
	ToTokenAccount   string `json:"toTokenAccount"`
	TokenAmount      int64  `json:"tokenAmount"`
	Mint             string `json:"mint"`
}

type AccountData struct {
	Account             string               `json:"account"`
	NativeBalanceChange int64                `json:"nativeBalanceChange"`
	TokenBalanceChanges []TokenBalanceChange `json:"tokenBalanceChanges"`
}

type TokenBalanceChange struct {
	UserAccount    string         `json:"userAccount"`
	TokenAccount   string         `json:"tokenAccount"`
	Mint           string         `json:"mint"`
	RawTokenAmount RawTokenAmount `json:"rawTokenAmount"`
}

type RawTokenAmount struct {
	TokenAmount string `json:"tokenAmount"`
	Decimals    int    `json:"decimals"`
}

type TransactionError struct {
	Error string `json:"error"`
}

type Instruction struct {
	Accounts          []string           `json:"accounts"`
	Data              string             `json:"data"`
	ProgramId         string             `json:"programId"`
	InnerInstructions []InnerInstruction `json:"innerInstructions"`
}

type InnerInstruction struct {
	Accounts  []string `json:"accounts"`
	Data      string   `json:"data"`
	ProgramId string   `json:"programId"`
}

type Events struct {
	NFT                          *NFTEvent               `json:"nft"`
	Swap                         *SwapEvent              `json:"swap"`
	Compressed                   *CompressedEvent        `json:"compressed"`
	DistributeCompressionRewards *CompressionRewardEvent `json:"distributeCompressionRewards"`
	SetAuthority                 *SetAuthorityEvent      `json:"setAuthority"`
}

type NFTEvent struct {
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Source      string    `json:"source"`
	Amount      int64     `json:"amount"`
	Fee         int64     `json:"fee"`
	FeePayer    string    `json:"feePayer"`
	Signature   string    `json:"signature"`
	Slot        int64     `json:"slot"`
	Timestamp   int64     `json:"timestamp"`
	SaleType    string    `json:"saleType"`
	Buyer       string    `json:"buyer"`
	Seller      string    `json:"seller"`
	Staker      string    `json:"staker"`
	NFTs        []NFTItem `json:"nfts"`
}

type NFTItem struct {
	Mint          string `json:"mint"`
	TokenStandard string `json:"tokenStandard"`
}

type SwapEvent struct {
	NativeInput  NativeSwapInfo   `json:"nativeInput"`
	NativeOutput NativeSwapInfo   `json:"nativeOutput"`
	TokenInputs  []TokenSwapInfo  `json:"tokenInputs"`
	TokenOutputs []TokenSwapInfo  `json:"tokenOutputs"`
	TokenFees    []TokenSwapInfo  `json:"tokenFees"`
	NativeFees   []NativeSwapInfo `json:"nativeFees"`
	InnerSwaps   []InnerSwap      `json:"innerSwaps"`
}

type NativeSwapInfo struct {
	Account string `json:"account"`
	Amount  string `json:"amount"`
}

type TokenSwapInfo struct {
	UserAccount    string         `json:"userAccount"`
	TokenAccount   string         `json:"tokenAccount"`
	Mint           string         `json:"mint"`
	RawTokenAmount RawTokenAmount `json:"rawTokenAmount"`
}

type InnerSwap struct {
	TokenInputs  []InnerSwapTransfer `json:"tokenInputs"`
	TokenOutputs []InnerSwapTransfer `json:"tokenOutputs"`
	TokenFees    []InnerSwapTransfer `json:"tokenFees"`
	NativeFees   []InnerSwapNative   `json:"nativeFees"`
	ProgramInfo  SwapProgramInfo     `json:"programInfo"`
}

type InnerSwapTransfer struct {
	FromUserAccount  string `json:"fromUserAccount"`
	ToUserAccount    string `json:"toUserAccount"`
	FromTokenAccount string `json:"fromTokenAccount"`
	ToTokenAccount   string `json:"toTokenAccount"`
	TokenAmount      int64  `json:"tokenAmount"`
	Mint             string `json:"mint"`
}

type InnerSwapNative struct {
	FromUserAccount string `json:"fromUserAccount"`
	ToUserAccount   string `json:"toUserAccount"`
	Amount          int64  `json:"amount"`
}

type SwapProgramInfo struct {
	Source          string `json:"source"`
	Account         string `json:"account"`
	ProgramName     string `json:"programName"`
	InstructionName string `json:"instructionName"`
}

type CompressedEvent struct {
	Type                  string `json:"type"`
	TreeId                string `json:"treeId"`
	AssetId               string `json:"assetId"`
	LeafIndex             int    `json:"leafIndex"`
	InstructionIndex      int    `json:"instructionIndex"`
	InnerInstructionIndex int    `json:"innerInstructionIndex"`
	NewLeafOwner          string `json:"newLeafOwner"`
	OldLeafOwner          string `json:"oldLeafOwner"`
}

type CompressionRewardEvent struct {
	Amount int `json:"amount"`
}

type SetAuthorityEvent struct {
	Account               string `json:"account"`
	From                  string `json:"from"`
	To                    string `json:"to"`
	InstructionIndex      int    `json:"instructionIndex"`
	InnerInstructionIndex int    `json:"innerInstructionIndex"`
}
