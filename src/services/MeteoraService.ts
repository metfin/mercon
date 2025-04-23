import axios from "axios";

export interface MeteoraPair {
	id: string;
	address: string;
	apr: number;
	apy: number;
	base_fee_percentage: string;
	bin_step: number;
	cumulative_fee_volume: string;
	cumulative_trade_volume: string;
	current_price: number;
	farm_apr: number;
	farm_apy: number;
	fee_tvl_ratio: {
		hour_1: number;
		hour_12: number;
		hour_2: number;
		hour_24: number;
		hour_4: number;
		min_30: number;
	};
	fees: {
		hour_1: number;
		hour_12: number;
		hour_2: number;
		hour_24: number;
		hour_4: number;
		min_30: number;
	};
	fees_24h: number;
	hide: boolean;
	is_blacklisted: boolean;
	liquidity: string;
	max_fee_percentage: string;
	mint_x: string;
	mint_y: string;
	name: string;
	protocol_fee_percentage: string;
	reserve_x: string;
	reserve_x_amount: number;
	reserve_y: string;
	reserve_y_amount: number;
	reward_mint_x: string;
	reward_mint_y: string;
	tags: string[];
	today_fees: number;
	trade_volume_24h: number;
	volume: {
		hour_1: number;
		hour_12: number;
		hour_2: number;
		hour_24: number;
		hour_4: number;
		min_30: number;
	};
}

export interface MeteoraPosition {
	address: string;
	daily_fee_yield: number;
	fee_apr_24h: number;
	fee_apy_24h: number;
	owner: string;
	pair_address: string;
	total_fee_usd_claimed: number;
	total_fee_x_claimed: number;
	total_fee_y_claimed: number;
	total_reward_usd_claimed: number;
	total_reward_x_claimed: number;
	total_reward_y_claimed: number;
}

export class MeteoraService {
	private baseUrl: string;

	constructor(baseUrl = "https://dlmm-api.meteora.ag") {
		this.baseUrl = baseUrl;
	}

	/**
	 * Fetch Meteora pair data for the wallet
	 * @param onProgress Optional progress callback
	 */
	public async getPair(pairAddress: string): Promise<MeteoraPair | null> {
		try {
			const response = await axios.get<MeteoraPair>(
				`${this.baseUrl}/pairs/${pairAddress}`,
			);

			return response.data;
		} catch (error) {
			console.error("Error fetching Meteora pair data:", error);
			return null;
		}
	}

	/**
	 * Fetch user's liquidity positions
	 * @param onProgress Optional progress callback
	 */
	public async getPosition(
		positionAddress: string,
	): Promise<MeteoraPosition | null> {
		try {
			// This is a placeholder for the actual API call
			const response = await axios.get<MeteoraPosition>(
				`${this.baseUrl}/positions/${positionAddress}`,
			);

			return response.data;
		} catch (error) {
			console.error("Error fetching Meteora liquidity positions:", error);
			return null;
		}
	}
}
