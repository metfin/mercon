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

export interface MeteoraPositionTransactions {
  deposits: Array<{
    tx_id: string;
    token_x_usd_amount: number;
    token_y_usd_amount: number;
  }>;
  withdrawals: Array<{
    tx_id: string;
    token_x_usd_amount: number;
    token_y_usd_amount: number;
  }>;
  fees: Array<{
    tx_id: string;
    token_x_usd_amount: number;
    token_y_usd_amount: number;
  }>;
}

export class MeteoraService {
  private baseUrl: string;

  constructor(baseUrl = "https://dlmm-api.meteora.ag") {
    this.baseUrl = baseUrl;
  }

  /**
   * Helper to handle retries with exponential backoff for API requests
   * @param operation The async function to retry
   * @param maxRetries Maximum number of retry attempts
   * @param baseDelay Initial delay in milliseconds
   */
  private async withRetry<T>(
    operation: () => Promise<T>,
    maxRetries = 5,
    baseDelay = 500,
    operationName = "API operation"
  ): Promise<T> {
    let retries = 0;

    while (true) {
      try {
        return await operation();
      } catch (error) {
        retries++;

        // Detailed error logging
        const errorMsg = error instanceof Error ? error.message : String(error);
        console.error(`[METEORA API] ${operationName} failed: ${errorMsg}`);

        // Check if this is a rate limit error
        const isRateLimit =
          (error instanceof Response && error.status === 429) ||
          errorMsg.includes("Too Many Requests");

        // If we've reached max retries or error is not a rate limit issue, throw
        if (retries > maxRetries || !isRateLimit) {
          console.error(
            `[METEORA API] ${operationName} failed after ${retries} attempts, giving up.`
          );
          throw error;
        }

        // Calculate exponential backoff with jitter
        const delay = Math.min(
          baseDelay * 2 ** retries + Math.random() * 300,
          10000 // Max 10 second delay
        );

        console.log(
          `[METEORA API] ${operationName} rate limited (attempt ${retries}/${maxRetries}). Retrying after ${Math.round(delay)}ms delay...`
        );
        await new Promise((resolve) => setTimeout(resolve, delay));
      }
    }
  }

  /**
   * Fetch Meteora pair data for the wallet
   * @param onProgress Optional progress callback
   */
  public async getPair(pairAddress: string): Promise<MeteoraPair | null> {
    console.log(
      `[METEORA API] Fetching pair data for ${pairAddress.slice(0, 8)}...`
    );

    return this.withRetry(
      async () => {
        try {
          const response = await fetch(`${this.baseUrl}/pairs/${pairAddress}`);

          if (!response.ok) {
            throw new Error(`Request failed with status ${response.status}`);
          }

          const data = (await response.json()) as MeteoraPair;

          console.log(
            `[METEORA API] Successfully fetched pair data for ${pairAddress.slice(0, 8)}`
          );
          return data;
        } catch (error) {
          console.error(
            "[METEORA API] Error fetching Meteora pair data:",
            error
          );
          throw error; // Rethrow for retry mechanism
        }
      },
      5,
      500,
      `Fetch pair ${pairAddress.slice(0, 8)}`
    ).catch((error) => {
      console.error(
        `[METEORA API] Failed to fetch pair after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return null;
    });
  }

  /**
   * Fetch user's liquidity positions
   * @param onProgress Optional progress callback
   */
  public async getPosition(
    positionAddress: string
  ): Promise<MeteoraPosition | null> {
    console.log(
      `[METEORA API] Fetching position data for ${positionAddress.slice(0, 8)}...`
    );

    return this.withRetry(
      async () => {
        try {
          // This is a placeholder for the actual API call
          const response = await fetch(
            `${this.baseUrl}/positions/${positionAddress}`
          );

          if (!response.ok) {
            throw new Error(`Request failed with status ${response.status}`);
          }

          const data = (await response.json()) as MeteoraPosition;

          console.log(
            `[METEORA API] Successfully fetched position data for ${positionAddress.slice(0, 8)}`
          );
          return data;
        } catch (error) {
          console.error(
            "[METEORA API] Error fetching Meteora liquidity positions:",
            error
          );
          throw error; // Rethrow for retry mechanism
        }
      },
      5,
      500,
      `Fetch position ${positionAddress.slice(0, 8)}`
    ).catch((error) => {
      console.error(
        `[METEORA API] Failed to fetch position after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return null;
    });
  }

  /**
   * Fetch transactions for a position
   * @param positionAddress The position address to fetch transactions for
   */
  public async getPositionTransactions(
    positionAddress: string
  ): Promise<MeteoraPositionTransactions | null> {
    console.log(
      `[METEORA API] Fetching transactions for position ${positionAddress.slice(0, 8)}...`
    );

    return this.withRetry(
      async () => {
        try {
          const response = await fetch(
            `${this.baseUrl}/positions/${positionAddress}/transactions`
          );

          if (!response.ok) {
            throw new Error(`Request failed with status ${response.status}`);
          }

          const data = (await response.json()) as {
            deposits: Array<{
              tx_id: string;
              token_x_usd_amount: number;
              token_y_usd_amount: number;
            }>;
            withdrawals: Array<{
              tx_id: string;
              token_x_usd_amount: number;
              token_y_usd_amount: number;
            }>;
            fees: Array<{
              tx_id: string;
              token_x_usd_amount: number;
              token_y_usd_amount: number;
            }>;
          };

          console.log(
            `[METEORA API] Successfully fetched transactions for position ${positionAddress.slice(0, 8)}`
          );
          return {
            deposits: data.deposits || [],
            withdrawals: data.withdrawals || [],
            fees: data.fees || [],
          };
        } catch (error) {
          console.error(
            "[METEORA API] Error fetching position transactions:",
            error
          );
          throw error; // Rethrow for retry mechanism
        }
      },
      5,
      500,
      `Fetch transactions for position ${positionAddress.slice(0, 8)}`
    ).catch((error) => {
      console.error(
        `[METEORA API] Failed to fetch position transactions after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return null;
    });
  }
}
