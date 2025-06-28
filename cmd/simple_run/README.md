# Simple Wallet Scanner

A simple command-line tool to scan a Solana wallet address and parse all Meteora DAMM and DLMM transactions.

## Features

- 🔍 Scans any Solana wallet address for transactions
- 📊 Parses Meteora DAMM (Dynamic Automated Market Making) instructions
- 🔵 Parses Meteora DLMM (Dynamic Liquidity Market Making) instructions
- 💻 Pretty-printed output with transaction details
- ⚡ No database or Redis required - just pure scanning and parsing
- 🔄 Configurable transaction limit

## Usage

### Basic Usage

```bash
# Scan a wallet with default settings (100 transactions)
go run main.go -wallet <WALLET_ADDRESS>

# Scan with custom transaction limit
go run main.go -wallet <WALLET_ADDRESS> -limit 50
```

### Examples

```bash
# Example with a real wallet address
go run main.go -wallet CSdRLr6SWaCrjCgSyJ4mSHAg3EwUzymtLT6e47uj5eX9 -limit 20

# Scan the last 10 transactions only
go run main.go -wallet ABC123... -limit 10
```

## Configuration

The scanner uses environment variables for RPC endpoints. You can set:

```bash
export SOLANA_RPC_URLS="https://api.mainnet-beta.solana.com,https://solana-api.projectserum.com"
export CACHE_TTL_SECONDS=300
```

Or create a `.env` file in the external service directory:

```env
SOLANA_RPC_URLS=https://api.mainnet-beta.solana.com,https://solana-api.projectserum.com
CACHE_TTL_SECONDS=300
```

## Output Format

The scanner provides detailed output for each transaction:

```
🔍 Transaction #1
📝 Signature: 5KJ8...abc123
⏰ Block Time: 2024-01-15 10:30:45 UTC
🎯 Slot: 12345678

🟢 DAMM Instructions Found: 1
  1. Type: AddLiquidity
     Parsed Data: {
       "position": "ABC123...",
       "pool": "DEF456...",
       "amounts": {
         "liquidity_delta": "1000000",
         "token_a_amount": "500000",
         "token_b_amount": "500000"
       }
     }
```

## Summary Report

At the end of each scan, you'll get a summary:

```
📈 SCAN SUMMARY
================================================================================
💳 Wallet: 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM
🔢 Total Transactions Scanned: 50
✅ Transactions with Meteora Instructions: 12
🟢 Total DAMM Instructions: 8
🔵 Total DLMM Instructions: 15
⚡ Scan completed successfully!
```

## Prerequisites

- Go 1.19+
- Access to Solana RPC endpoints
- Valid Solana wallet address to scan

## Dependencies

This scanner uses the existing metfin codebase components:

- `external/service` - For RPC communication
- `core/parsers/damm` - DAMM instruction parsing
- `core/parsers/dlmm` - DLMM instruction parsing

## Troubleshooting

### Common Issues

1. **Invalid wallet address**: Make sure the wallet address is a valid base58-encoded Solana public key
2. **RPC rate limits**: If you hit rate limits, try using multiple RPC endpoints or reducing the limit
3. **No transactions found**: The wallet might be new or inactive

### Error Messages

- `❌ Failed to get wallet transactions`: RPC connection issue or invalid wallet
- `⚠️ Failed to parse DAMM/DLMM instructions`: Transaction contains invalid or unknown instruction data
- `⚪ No Meteora instructions found`: Transaction doesn't contain any Meteora-related instructions

## Limitations

- Only parses Meteora DAMM and DLMM instructions
- Limited to the most recent transactions (configurable via `-limit`)
- Requires active RPC endpoints
- No historical data persistence (pure scanning tool)
