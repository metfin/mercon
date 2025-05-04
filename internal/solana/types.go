package solana

type RPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	Result  Result `json:"result"`
	ID      string `json:"id"`
	Error   struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type Result struct {
	BlockTime   int64       `json:"blockTime"`
	Meta        Meta        `json:"meta"`
	Transaction Transaction `json:"transaction"`
}

type Transaction struct {
	Message    Message  `json:"message"`
	Signatures []string `json:"signatures"`
}

type Message struct {
	AccountKeys     []AccountKey  `json:"accountKeys"`
	Instructions    []Instruction `json:"instructions"`
	RecentBlockhash string        `json:"recentBlockhash"`
	Header          MessageHeader `json:"header"`
}

type AccountKey struct {
	Pubkey   string `json:"pubkey"`
	Signer   bool   `json:"signer"`
	Source   string `json:"source"`
	Writable bool   `json:"writable"`
}

type MessageHeader struct {
	NumRequiredSignatures       int `json:"numRequiredSignatures"`
	NumReadonlySignedAccounts   int `json:"numReadonlySignedAccounts"`
	NumReadonlyUnsignedAccounts int `json:"numReadonlyUnsignedAccounts"`
}

type Meta struct {
	ComputeUnitsConsumed int64              `json:"computeUnitsConsumed"`
	Err                  interface{}        `json:"err"`
	Fee                  int64              `json:"fee"`
	InnerInstructions    []InnerInstruction `json:"innerInstructions"`
	LogMessages          []string           `json:"logMessages,omitempty"`
	PostBalances         []int64            `json:"postBalances,omitempty"`
	PreBalances          []int64            `json:"preBalances,omitempty"`
	PriorityFee          int64              `json:"priorityFee,omitempty"`
	Status               TransactionStatus  `json:"status,omitempty"`
}

type TransactionStatus struct {
	Ok  interface{} `json:"Ok,omitempty"`
	Err interface{} `json:"Err,omitempty"`
}

type InnerInstruction struct {
	Index        int           `json:"index"`
	Instructions []Instruction `json:"instructions"`
}

type Instruction struct {
	Parsed      *Parsed  `json:"parsed,omitempty"`
	Program     string   `json:"program"`
	ProgramID   string   `json:"programId"`
	StackHeight int      `json:"stackHeight"`
	Accounts    []string `json:"accounts,omitempty"`
	Data        string   `json:"data,omitempty"`
}

type Parsed struct {
	Info Info   `json:"info"`
	Type string `json:"type"`
}

type Info struct {
	// These fields vary depending on the instruction type.
	// We'll use json.RawMessage or interface{} for flexible parsing,
	// or define a union struct with pointers.
	ExtensionTypes []string     `json:"extensionTypes,omitempty"`
	Mint           string       `json:"mint,omitempty"`
	Lamports       int64        `json:"lamports,omitempty"`
	NewAccount     string       `json:"newAccount,omitempty"`
	Owner          string       `json:"owner,omitempty"`
	Source         string       `json:"source,omitempty"`
	Space          int          `json:"space,omitempty"`
	Account        string       `json:"account,omitempty"`
	Destination    string       `json:"destination,omitempty"`
	Authority      string       `json:"authority,omitempty"`
	TokenAmount    *TokenAmount `json:"tokenAmount,omitempty"`
}

type TokenAmount struct {
	Amount         string  `json:"amount"`
	Decimals       int     `json:"decimals"`
	UIAmount       float64 `json:"uiAmount"`
	UIAmountString string  `json:"uiAmountString"`
}
