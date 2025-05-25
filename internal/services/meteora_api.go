package services

import (
	"fmt"

	"github.com/wnt/mercon/internal/utils"
)

type MeteoraPubClient struct {
	httpClient *utils.HTTPClient
}

// NewMeteoraPubClient creates a new client for the Meteora public API
func NewMeteoraPubClient() *MeteoraPubClient {
	return &MeteoraPubClient{
		httpClient: utils.NewHTTPClient(
			utils.WithDefaultHeaders(map[string]string{
				"Content-Type": "application/json",
			}),
		),
	}
}

// Position represents a position with APY in the Meteora protocol
type PositionWithApy struct {
	Address               string  `json:"address"`
	PairAddress           string  `json:"pair_address"`
	Owner                 string  `json:"owner"`
	TotalFeeXClaimed      int64   `json:"total_fee_x_claimed"`
	TotalFeeYClaimed      int64   `json:"total_fee_y_claimed"`
	TotalRewardXClaimed   int64   `json:"total_reward_x_claimed"`
	TotalRewardYClaimed   int64   `json:"total_reward_y_claimed"`
	TotalFeeUSDClaimed    float64 `json:"total_fee_usd_claimed"`
	TotalRewardUSDClaimed float64 `json:"total_reward_usd_claimed"`
	FeeApy24h             float64 `json:"fee_apy_24h"`
	FeeApr24h             float64 `json:"fee_apr_24h"`
	DailyFeeYield         float64 `json:"daily_fee_yield"`
}

// Position represents a position in the Meteora protocol (v2)
type Position struct {
	Address               string  `json:"address"`
	PairAddress           string  `json:"pair_address"`
	Owner                 string  `json:"owner"`
	TotalFeeXClaimed      int64   `json:"total_fee_x_claimed"`
	TotalFeeYClaimed      int64   `json:"total_fee_y_claimed"`
	TotalRewardXClaimed   int64   `json:"total_reward_x_claimed"`
	TotalRewardYClaimed   int64   `json:"total_reward_y_claimed"`
	TotalFeeUSDClaimed    float64 `json:"total_fee_usd_claimed"`
	TotalRewardUSDClaimed float64 `json:"total_reward_usd_claimed"`
	CreatedAt             string  `json:"created_at"`
}

// ClaimFee represents a fee claim transaction
type ClaimFee struct {
	TxID             string  `json:"tx_id"`
	PositionAddress  string  `json:"position_address"`
	PairAddress      string  `json:"pair_address"`
	TokenXAmount     int64   `json:"token_x_amount"`
	TokenYAmount     int64   `json:"token_y_amount"`
	TokenXUSDAmount  float64 `json:"token_x_usd_amount"`
	TokenYUSDAmount  float64 `json:"token_y_usd_amount"`
	OnchainTimestamp int64   `json:"onchain_timestamp"`
}

// DepositWithdraw represents a deposit or withdraw transaction
type DepositWithdraw struct {
	TxID             string  `json:"tx_id"`
	PositionAddress  string  `json:"position_address"`
	PairAddress      string  `json:"pair_address"`
	ActiveBinID      int64   `json:"active_bin_id"`
	TokenXAmount     int64   `json:"token_x_amount"`
	TokenYAmount     int64   `json:"token_y_amount"`
	Price            float64 `json:"price"`
	TokenXUSDAmount  float64 `json:"token_x_usd_amount"`
	TokenYUSDAmount  float64 `json:"token_y_usd_amount"`
	OnchainTimestamp int64   `json:"onchain_timestamp"`
}

// FeeData represents fee data in different timeframes
type FeeData struct {
	Min30  float64 `json:"min_30"`
	Hour1  float64 `json:"hour_1"`
	Hour2  float64 `json:"hour_2"`
	Hour4  float64 `json:"hour_4"`
	Hour12 float64 `json:"hour_12"`
	Hour24 float64 `json:"hour_24"`
}

// FeeTvlRatioData represents fee/TVL ratio data in different timeframes
type FeeTvlRatioData struct {
	Min30  float64 `json:"min_30"`
	Hour1  float64 `json:"hour_1"`
	Hour2  float64 `json:"hour_2"`
	Hour4  float64 `json:"hour_4"`
	Hour12 float64 `json:"hour_12"`
	Hour24 float64 `json:"hour_24"`
}

// VolumeData represents volume data in different timeframes
type VolumeData struct {
	Min30  float64 `json:"min_30"`
	Hour1  float64 `json:"hour_1"`
	Hour2  float64 `json:"hour_2"`
	Hour4  float64 `json:"hour_4"`
	Hour12 float64 `json:"hour_12"`
	Hour24 float64 `json:"hour_24"`
}

// PairInfo represents a liquidity pair info
type PairInfo struct {
	Address               string          `json:"address"`
	Name                  string          `json:"name"`
	MintX                 string          `json:"mint_x"`
	MintY                 string          `json:"mint_y"`
	ReserveX              string          `json:"reserve_x"`
	ReserveY              string          `json:"reserve_y"`
	ReserveXAmount        int64           `json:"reserve_x_amount"`
	ReserveYAmount        int64           `json:"reserve_y_amount"`
	BinStep               int32           `json:"bin_step"`
	BaseFeePercentage     string          `json:"base_fee_percentage"`
	MaxFeePercentage      string          `json:"max_fee_percentage"`
	ProtocolFeePercentage string          `json:"protocol_fee_percentage"`
	Liquidity             string          `json:"liquidity"`
	RewardMintX           string          `json:"reward_mint_x"`
	RewardMintY           string          `json:"reward_mint_y"`
	Fees24h               float64         `json:"fees_24h"`
	TodayFees             float64         `json:"today_fees"`
	TradeVolume24h        float64         `json:"trade_volume_24h"`
	CumulativeTradeVolume string          `json:"cumulative_trade_volume"`
	CumulativeFeeVolume   string          `json:"cumulative_fee_volume"`
	CurrentPrice          float64         `json:"current_price"`
	Apr                   float64         `json:"apr"`
	Apy                   float64         `json:"apy"`
	FarmApr               float64         `json:"farm_apr"`
	FarmApy               float64         `json:"farm_apy"`
	Hide                  bool            `json:"hide"`
	IsBlacklisted         bool            `json:"is_blacklisted"`
	Fees                  FeeData         `json:"fees"`
	FeeTvlRatio           FeeTvlRatioData `json:"fee_tvl_ratio"`
	Volume                VolumeData      `json:"volume"`
	Tags                  []string        `json:"tags"`
}

// PairGroup represents a group of pairs with the same tokens
type PairGroup struct {
	Name  string     `json:"name"`
	Pairs []PairInfo `json:"pairs"`
}

// AllGroupOfPairs represents all groups of pairs with pagination
type AllGroupOfPairs struct {
	Groups []PairGroup `json:"groups"`
	Total  int         `json:"total"`
}

// GetPosition fetches a position by address with APY information
func (c *MeteoraPubClient) GetPosition(positionAddress string) (*PositionWithApy, error) {
	path := fmt.Sprintf("/position/%s", positionAddress)

	response, err := c.httpClient.Get(path, nil, nil)
	if err != nil {
		return nil, err
	}

	var position PositionWithApy
	if err := response.DecodeJSON(&position); err != nil {
		return nil, err
	}

	return &position, nil
}

// GetPositionV2 fetches a position by address with the v2 endpoint
func (c *MeteoraPubClient) GetPositionV2(positionAddress string) (*Position, error) {
	path := fmt.Sprintf("/position_v2/%s", positionAddress)

	response, err := c.httpClient.Get(path, nil, nil)
	if err != nil {
		return nil, err
	}

	var position Position
	if err := response.DecodeJSON(&position); err != nil {
		return nil, err
	}

	return &position, nil
}

// GetClaimFees fetches the claim fees for a position
func (c *MeteoraPubClient) GetClaimFees(positionAddress string) ([]ClaimFee, error) {
	path := fmt.Sprintf("/position/%s/claim_fees", positionAddress)

	response, err := c.httpClient.Get(path, nil, nil)
	if err != nil {
		return nil, err
	}

	var claimFees []ClaimFee
	if err := response.DecodeJSON(&claimFees); err != nil {
		return nil, err
	}

	return claimFees, nil
}

// GetWithdraws fetches the withdraws for a position
func (c *MeteoraPubClient) GetWithdraws(positionAddress string) ([]DepositWithdraw, error) {
	path := fmt.Sprintf("/position/%s/withdraws", positionAddress)

	response, err := c.httpClient.Get(path, nil, nil)
	if err != nil {
		return nil, err
	}

	var withdraws []DepositWithdraw
	if err := response.DecodeJSON(&withdraws); err != nil {
		return nil, err
	}

	return withdraws, nil
}

// GetDeposits fetches the deposits for a position
func (c *MeteoraPubClient) GetDeposits(positionAddress string) ([]DepositWithdraw, error) {
	path := fmt.Sprintf("/position/%s/deposits", positionAddress)

	response, err := c.httpClient.Get(path, nil, nil)
	if err != nil {
		return nil, err
	}

	var deposits []DepositWithdraw
	if err := response.DecodeJSON(&deposits); err != nil {
		return nil, err
	}

	return deposits, nil
}

// GetAllPairs fetches all pairs
func (c *MeteoraPubClient) GetAllPairs(includeUnknown *bool) ([]PairInfo, error) {
	queryParams := make(map[string]string)
	if includeUnknown != nil {
		queryParams["include_unknown"] = fmt.Sprintf("%t", *includeUnknown)
	}

	response, err := c.httpClient.Get("/pair/all", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var pairs []PairInfo
	if err := response.DecodeJSON(&pairs); err != nil {
		return nil, err
	}

	return pairs, nil
}

// GetAllPairsByGroups fetches all pairs grouped by token pairs
func (c *MeteoraPubClient) GetAllPairsByGroups(params map[string]interface{}) (*AllGroupOfPairs, error) {
	queryParams := make(map[string]string)

	for key, value := range params {
		switch v := value.(type) {
		case string:
			queryParams[key] = v
		case int:
			queryParams[key] = fmt.Sprintf("%d", v)
		case float64:
			queryParams[key] = fmt.Sprintf("%f", v)
		case bool:
			queryParams[key] = fmt.Sprintf("%t", v)
		case []string:
			// This is a bit tricky with our current HTTP client
			// For simplicity, we'll just take the first value for now
			if len(v) > 0 {
				queryParams[key] = v[0]
			}
		}
	}

	response, err := c.httpClient.Get("/pair/all_by_groups", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var allGroups AllGroupOfPairs
	if err := response.DecodeJSON(&allGroups); err != nil {
		return nil, err
	}

	return &allGroups, nil
}

// GetPair fetches a single pair by address
func (c *MeteoraPubClient) GetPair(pairAddress string) (*PairInfo, error) {
	path := fmt.Sprintf("/pair/%s", pairAddress)

	response, err := c.httpClient.Get(path, nil, nil)
	if err != nil {
		return nil, err
	}

	var pair PairInfo
	if err := response.DecodeJSON(&pair); err != nil {
		return nil, err
	}

	return &pair, nil
}
