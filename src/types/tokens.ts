/**
 * Metadata for a token
 */
export interface TokenMeta {
	address: string;
	name?: string;
	symbol?: string;
	decimals: number;
	logoURI?: string;
}

/**
 * Token list format
 */
export interface TokenList {
	name: string;
	logoURI: string;
	keywords: string[];
	tags: Record<string, TokenListTag>;
	timestamp: string;
	tokens: TokenMeta[];
}

/**
 * Token tag definition
 */
export interface TokenListTag {
	name: string;
	description: string;
}
