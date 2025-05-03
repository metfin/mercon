package solana

import (
	"encoding/json"

	"github.com/google/uuid"
)

type RpcBody struct {
	Method  string        `json:"method"`
	Jsonrpc string        `json:"jsonrpc"`
	Params  []interface{} `json:"params"`
	Id      string        `json:"id"`
}

func (b *RpcBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(b)
}

func (b *RpcBody) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, b)
}

// NewRpcBody creates a new RpcBody with a specified method, params, and a new id
func NewRpcBody(method string, params []interface{}) ([]byte, error) {
	body := &RpcBody{
		Method:  method,
		Jsonrpc: "2.0",
		Params:  params,
		Id:      uuid.New().String(),
	}
	return json.Marshal(body)
}
