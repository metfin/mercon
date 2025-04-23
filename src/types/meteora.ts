/**
 * Represents a Meteora DLMM Pair
 */
export interface MeteoraDlmmPairData {
	lbPair: string;
	name: string;
	mintX: string;
	mintY: string;
	binStep: number;
	baseFeeBps: number;
}

/**
 * Transaction types for a position
 */
export interface MeteoraTransactionEntry {
	tx_id: string;
	token_x_usd_amount: number;
	token_y_usd_amount: number;
}

/**
 * Position transactions grouped by type
 */
export interface MeteoraPositionTransactions {
	deposits: MeteoraTransactionEntry[];
	withdrawals: MeteoraTransactionEntry[];
	fees: MeteoraTransactionEntry[];
}

/**
 * Database schema for transactions
 */
export interface MeteoraDlmmDbTransactions {
	block_time: number;
	is_hawksight: boolean;
	signature: string;
	position_address: string;
	owner_address: string;
	pair_address: string;
	base_mint: string;
	base_symbol: string;
	base_decimals: number;
	base_logo: string;
	quote_mint: string;
	quote_symbol: string;
	quote_decimals: number;
	quote_logo: string;
	is_inverted: number;
	position_is_open: number;
	is_opening_transaction: number;
	is_closing_transaction: number;
	price: number;
	fee_amount: number;
	deposit: number;
	withdrawal: number;
	usd_fee_amount: number;
	usd_deposit: number;
	usd_withdrawal: number;
}
