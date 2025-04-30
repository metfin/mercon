// Meteora DLMM API Service aligned with OpenAPI documentation
// This is a drop-in replacement with enhanced capabilities based on the API spec

// Types from the API specification
export interface PairInfo {
  address: string;
  id: string;
  name: string;
  mint_x: string;
  mint_y: string;
  reserve_x: string;
  reserve_y: string;
  reserve_x_amount: number;
  reserve_y_amount: number;
  liquidity: string;
  bin_step: number;
  current_price: number;
  base_fee_percentage: string;
  max_fee_percentage: string;
  protocol_fee_percentage: string;
  cumulative_fee_volume: string;
  cumulative_trade_volume: string;
  trade_volume_24h: number;
  fees_24h: number;
  today_fees: number;
  apr: number;
  apy: number;
  farm_apr: number;
  farm_apy: number;
  volume: VolumeData;
  fees: VolumeData;
  fee_tvl_ratio: VolumeData;
  hide: boolean;
  is_blacklisted: boolean;
  reward_mint_x: string;
  reward_mint_y: string;
  tags: string[];
}

export interface VolumeData {
  min_30: number;
  hour_1: number;
  hour_2: number;
  hour_4: number;
  hour_12: number;
  hour_24: number;
}

export interface PositionWithApy {
  address: string;
  pair_address: string;
  owner: string;
  total_fee_x_claimed: number;
  total_fee_y_claimed: number;
  total_reward_x_claimed: number;
  total_reward_y_claimed: number;
  total_fee_usd_claimed: number;
  total_reward_usd_claimed: number;
  fee_apy_24h: number;
  fee_apr_24h: number;
  daily_fee_yield: number;
}

export interface PositionTransactions {
  deposits: Deposit[];
  withdrawals: Withdrawal[];
  fees: Fee[];
}

export interface Deposit {
  tx_id: string;
  token_x_usd_amount: number;
  token_y_usd_amount: number;
}

export interface Withdrawal {
  tx_id: string;
  token_x_usd_amount: number;
  token_y_usd_amount: number;
}

export interface Fee {
  tx_id: string;
  token_x_usd_amount: number;
  token_y_usd_amount: number;
}

export interface ProtocolMetrics {
  total_tvl: number;
  daily_trade_volume: number;
  total_trade_volume: number;
  daily_fee: number;
  total_fee: number;
}

// For backward compatibility
export interface MeteoraPair extends PairInfo {}
export interface MeteoraPosition extends PositionWithApy {}
export interface MeteoraPositionTransactions extends PositionTransactions {}

// Sort options from API
export enum PairSortKey {
  Volume = "Volume",
  Tvl = "Tvl",
  Apr = "Apr",
  Apy = "Apy",
  FarmApr = "FarmApr",
  FarmApy = "FarmApy",
  Fee = "Fee",
  CreatedAt = "CreatedAt",
}

export enum OrderBy {
  Ascending = "Ascending",
  Descending = "Descending",
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
   * @param operationName Name of the operation for logging
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
   * Fetch protocol metrics
   */
  public async getProtocolMetrics(): Promise<
    ProtocolMetrics | null | undefined
  > {
    console.log("[METEORA API] Fetching protocol metrics...");

    try {
      const result = await this.withRetry(
        async () => {
          try {
            const response = await fetch(
              `${this.baseUrl}/info/protocol_metrics`
            );

            if (!response.ok) {
              throw new Error(`Request failed with status ${response.status}`);
            }

            const data = (await response.json()) as ProtocolMetrics[];
            if (!data || data.length === 0) {
              return null;
            }

            console.log("[METEORA API] Successfully fetched protocol metrics");
            return data[0];
          } catch (error) {
            console.error(
              "[METEORA API] Error fetching protocol metrics:",
              error
            );
            throw error; // Rethrow for retry mechanism
          }
        },
        5,
        500,
        "Fetch protocol metrics"
      );

      return result; // This will be of type ProtocolMetrics | null
    } catch (error) {
      console.error(
        `[METEORA API] Failed to fetch protocol metrics after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return null;
    }
  }

  /**
   * Fetch all pairs
   * @param includeUnknown Include pools with unverified tokens
   */
  public async getAllPairs(includeUnknown = true): Promise<PairInfo[]> {
    console.log("[METEORA API] Fetching all pairs...");

    return this.withRetry(
      async () => {
        try {
          const url = new URL(`${this.baseUrl}/pair/all`);
          url.searchParams.append("include_unknown", String(includeUnknown));

          const response = await fetch(url.toString());

          if (!response.ok) {
            throw new Error(`Request failed with status ${response.status}`);
          }

          const data = (await response.json()) as PairInfo[];

          console.log(
            `[METEORA API] Successfully fetched ${data.length} pairs`
          );
          return data;
        } catch (error) {
          console.error("[METEORA API] Error fetching all pairs:", error);
          throw error; // Rethrow for retry mechanism
        }
      },
      5,
      500,
      "Fetch all pairs"
    ).catch((error) => {
      console.error(
        `[METEORA API] Failed to fetch all pairs after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return [];
    });
  }

  /**
   * Fetch pairs with pagination
   * @param options Pagination and filter options
   */
  public async getPairsWithPagination(
    options: {
      page?: number;
      limit?: number;
      skipSize?: number;
      poolsToTop?: string[];
      sortKey?: PairSortKey;
      orderBy?: OrderBy;
      searchTerm?: string;
      includeUnknown?: boolean;
      hideLowTvl?: number;
      hideLowApr?: boolean;
      includeTokenMints?: string[];
      includePoolTokenPairs?: string[];
      tags?: string[];
    } = {}
  ): Promise<{
    pairs: PairInfo[];
    total: number;
    page: number;
  }> {
    console.log("[METEORA API] Fetching pairs with pagination...");

    return this.withRetry(
      async () => {
        try {
          const url = new URL(`${this.baseUrl}/pair/all_with_pagination`);

          // Add all options as query parameters
          if (options.page !== undefined)
            url.searchParams.append("page", String(options.page));
          if (options.limit !== undefined)
            url.searchParams.append("limit", String(options.limit));
          if (options.skipSize !== undefined)
            url.searchParams.append("skip_size", String(options.skipSize));
          if (options.poolsToTop?.length) {
            for (const pool of options.poolsToTop) {
              url.searchParams.append("pools_to_top", pool);
            }
          }
          if (options.sortKey)
            url.searchParams.append("sort_key", options.sortKey);
          if (options.orderBy)
            url.searchParams.append("order_by", options.orderBy);
          if (options.searchTerm)
            url.searchParams.append("search_term", options.searchTerm);
          if (options.includeUnknown !== undefined)
            url.searchParams.append(
              "include_unknown",
              String(options.includeUnknown)
            );
          if (options.hideLowTvl !== undefined)
            url.searchParams.append("hide_low_tvl", String(options.hideLowTvl));
          if (options.hideLowApr !== undefined)
            url.searchParams.append("hide_low_apr", String(options.hideLowApr));
          if (options.includeTokenMints?.length) {
            for (const mint of options.includeTokenMints) {
              url.searchParams.append("include_token_mints", mint);
            }
          }
          if (options.includePoolTokenPairs?.length) {
            for (const pair of options.includePoolTokenPairs) {
              url.searchParams.append("include_pool_token_pairs", pair);
            }
          }
          if (options.tags?.length) {
            for (const tag of options.tags) {
              url.searchParams.append("tags", tag);
            }
          }

          const response = await fetch(url.toString());

          if (!response.ok) {
            throw new Error(`Request failed with status ${response.status}`);
          }

          const data = (await response.json()) as {
            data: PairInfo[];
            total: number;
            page: number;
          };

          console.log(
            `[METEORA API] Successfully fetched ${data.data.length} pairs (page ${data.page} of ${Math.ceil(data.total / (options.limit || 50))})`
          );
          return {
            pairs: data.data,
            total: data.total,
            page: data.page,
          };
        } catch (error) {
          console.error(
            "[METEORA API] Error fetching pairs with pagination:",
            error
          );
          throw error; // Rethrow for retry mechanism
        }
      },
      5,
      500,
      "Fetch pairs with pagination"
    ).catch((error) => {
      console.error(
        `[METEORA API] Failed to fetch pairs with pagination after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return {
        pairs: [],
        total: 0,
        page: 0,
      };
    });
  }

  /**
   * Fetch data for a specific pair
   * @param pairAddress The address of the pair to fetch
   */
  public async getPair(pairAddress: string): Promise<MeteoraPair | null> {
    if (!pairAddress || pairAddress === "" || pairAddress.length !== 44) {
      console.error("[METEORA API] Pair address is required");
      return null;
    }

    console.log(
      `[METEORA API] Fetching pair data for ${pairAddress.slice(0, 8)}...`
    );

    return this.withRetry(
      async () => {
        try {
          const response = await fetch(`${this.baseUrl}/pair/${pairAddress}`);

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
   * Fetch data for a specific position
   * @param positionAddress The address of the position to fetch
   */
  public async getPosition(
    positionAddress: string
  ): Promise<MeteoraPosition | null> {
    console.log(
      `[METEORA API] Fetching position data for ${positionAddress.slice(0, 8)}...`
    );

    try {
      const result = await this.withRetry(
        async () => {
          try {
            const response = await fetch(
              `${this.baseUrl}/position/${positionAddress}`
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
              "[METEORA API] Error fetching Meteora position data:",
              error
            );
            throw error; // Rethrow for retry mechanism
          }
        },
        5,
        500,
        `Fetch position ${positionAddress.slice(0, 8)}`
      );

      return result;
    } catch (error) {
      console.error(
        `[METEORA API] Failed to fetch position after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return null;
    }
  }

  /**
   * Fetch transactions for a position
   * @param positionAddress The address of the position to fetch transactions for
   */
  public async getPositionTransactions(
    positionAddress: string
  ): Promise<MeteoraPositionTransactions | null> {
    console.log(
      `[METEORA API] Fetching transactions for position ${positionAddress.slice(0, 8)}...`
    );

    try {
      const result = await this.withRetry(
        async () => {
          try {
            const response = await fetch(
              `${this.baseUrl}/position/${positionAddress}/transactions`
            );

            if (!response.ok) {
              throw new Error(`Request failed with status ${response.status}`);
            }

            const data = (await response.json()) as {
              deposits: Deposit[];
              withdrawals: Withdrawal[];
              fees: Fee[];
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
      );

      return result;
    } catch (error) {
      console.error(
        `[METEORA API] Failed to fetch position transactions after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return null;
    }
  }

  /**
   * Fetch all positions for a wallet
   * @param walletAddress The wallet address to fetch positions for
   */
  public async getPositionsForWallet(
    walletAddress: string
  ): Promise<MeteoraPosition[]> {
    console.log(
      `[METEORA API] Fetching positions for wallet ${walletAddress.slice(0, 8)}...`
    );

    try {
      const result = await this.withRetry(
        async () => {
          try {
            const response = await fetch(
              `${this.baseUrl}/wallet/${walletAddress}/positions`
            );

            if (!response.ok) {
              throw new Error(`Request failed with status ${response.status}`);
            }

            const data = (await response.json()) as MeteoraPosition[];

            console.log(
              `[METEORA API] Successfully fetched ${data.length} positions for wallet ${walletAddress.slice(0, 8)}`
            );
            return data;
          } catch (error) {
            console.error(
              "[METEORA API] Error fetching wallet positions:",
              error
            );
            throw error; // Rethrow for retry mechanism
          }
        },
        5,
        500,
        `Fetch positions for wallet ${walletAddress.slice(0, 8)}`
      );

      return result;
    } catch (error) {
      console.error(
        `[METEORA API] Failed to fetch wallet positions after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return [];
    }
  }
}
