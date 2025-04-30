import type { TokenMeta } from "../types/tokens";

export class TokenPriceService {
  private jupiterUrl: string;

  constructor(jupiterUrl = "https://price.jup.ag/v4") {
    this.jupiterUrl = jupiterUrl;
  }

  /**
   * Helper to handle retries with exponential backoff for API requests
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
        console.error(`[TOKEN API] ${operationName} failed: ${errorMsg}`);

        // Check if this is a rate limit error
        const isRateLimit =
          (error instanceof Response && error.status === 429) ||
          errorMsg.includes("Too Many Requests");

        // If we've reached max retries or error is not a rate limit issue, throw
        if (retries > maxRetries || !isRateLimit) {
          console.error(
            `[TOKEN API] ${operationName} failed after ${retries} attempts, giving up.`
          );
          throw error;
        }

        // Calculate exponential backoff with jitter
        const delay = Math.min(
          baseDelay * 2 ** retries + Math.random() * 300,
          10000 // Max 10 second delay
        );

        console.log(
          `[TOKEN API] ${operationName} rate limited (attempt ${retries}/${maxRetries}). Retrying after ${Math.round(delay)}ms delay...`
        );
        await new Promise((resolve) => setTimeout(resolve, delay));
      }
    }
  }

  /**
   * Fetch token data including price and metadata
   */
  public async getTokenData(tokenAddress: string): Promise<TokenMeta | null> {
    console.log(
      `[TOKEN API] Fetching token data for ${tokenAddress.slice(0, 8)}...`
    );

    return this.withRetry(
      async () => {
        try {
          // Jupiter API call to get token price info
          const priceResponse = await fetch(
            `${this.jupiterUrl}/price?ids=${tokenAddress}`
          );

          if (!priceResponse.ok && priceResponse.status !== 404) {
            throw new Error(
              `Price request failed with status ${priceResponse.status}`
            );
          }

          const priceData = priceResponse.ok
            ? ((await priceResponse.json()) as {
                data?: {
                  [key: string]: {
                    price: number;
                  };
                };
              })
            : { data: {} };

          // Get token metadata from Jupiter token list
          const tokenListResponse = await fetch("https://token.jup.ag/all");

          if (!tokenListResponse.ok) {
            throw new Error(
              `Token list request failed with status ${tokenListResponse.status}`
            );
          }

          // Define token interface to avoid 'any'
          interface JupiterToken {
            address: string;
            name?: string;
            symbol?: string;
            decimals?: number;
            logoURI?: string;
          }

          const tokenList = (await tokenListResponse.json()) as JupiterToken[];

          const tokenInfo = tokenList.find(
            (token: JupiterToken) => token.address === tokenAddress
          );

          let price = 0;
          if (priceData.data?.[tokenAddress]) {
            price = priceData.data[tokenAddress].price;
          }

          // Create token metadata
          const tokenData: TokenMeta = {
            address: tokenAddress,
            name:
              tokenInfo?.name || `Unknown Token (${tokenAddress.slice(0, 4)})`,
            symbol: tokenInfo?.symbol || "???",
            decimals: tokenInfo?.decimals || 9,
            logoURI: tokenInfo?.logoURI || "",
          };

          console.log(
            `[TOKEN API] Successfully fetched token data for ${tokenAddress.slice(0, 8)}`
          );
          return tokenData;
        } catch (error) {
          // If the token isn't found, create a basic record
          if (error instanceof Error && error.message.includes("404")) {
            console.log(
              `[TOKEN API] Token ${tokenAddress.slice(0, 8)} not found, creating basic record`
            );
            return {
              address: tokenAddress,
              name: `Unknown Token (${tokenAddress.slice(0, 4)})`,
              symbol: "???",
              decimals: 9,
              logoURI: "",
            };
          }

          console.error("[TOKEN API] Error fetching token data:", error);
          throw error; // Rethrow for retry mechanism
        }
      },
      5,
      500,
      `Fetch token ${tokenAddress.slice(0, 8)}`
    ).catch((error) => {
      console.error(
        `[TOKEN API] Failed to fetch token after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return null;
    });
  }

  /**
   * Get historical price for a token at a specific timestamp
   * Note: Jupiter doesn't provide historical prices, we'll implement a fallback
   */
  public async getHistoricalPrice(
    tokenAddress: string,
    timestamp: number
  ): Promise<number | null> {
    console.log(
      `[TOKEN API] Fetching price for ${tokenAddress.slice(0, 8)} at ${new Date(timestamp * 1000).toISOString()}`
    );

    return this.withRetry(
      async () => {
        try {
          // Since Jupiter doesn't have historical prices API,
          // we fall back to current price as a demo
          const response = await fetch(
            `${this.jupiterUrl}/price?ids=${tokenAddress}`
          );

          if (!response.ok) {
            throw new Error(`Request failed with status ${response.status}`);
          }

          const data = (await response.json()) as {
            data?: {
              [key: string]: {
                price: number;
              };
            };
          };

          let price = 0;
          if (data.data?.[tokenAddress]) {
            price = data.data[tokenAddress].price;
          }

          console.log(
            `[TOKEN API] Price for ${tokenAddress.slice(0, 8)}: $${price}`
          );
          return price;
        } catch (error) {
          console.error("[TOKEN API] Error fetching price:", error);
          throw error;
        }
      },
      5,
      500,
      `Price ${tokenAddress.slice(0, 8)}`
    ).catch((error) => {
      console.error(
        `[TOKEN API] Failed to fetch price after retries: ${error instanceof Error ? error.message : String(error)}`
      );
      return null;
    });
  }
}
