# Mercon

An all in one library for downloading data related to Meteora Transactions, Positions and Pools. Inspired by [@GeekLad's](https://github.com/GeekLad) [meteora-dlmm-db](https://github.com/GeekLad/meteora-dlmm-db).


## Features

- Fetch transaction history for a Solana wallet
- Retrieve Meteora liquidity pool data and positions
- Get token price information
- Progress tracking and callbacks

## Installation

```bash
# Using npm
npm install mercon
```

```bash
# Using yarn
yarn add mercon
```

```bash
# Using bun
bun add mercon
```

## Quick Start

```typescript
import { DataDownloader } from 'mercon';

// Configure the data downloader
const downloader = new DataDownloader({
  walletAddress: 'YOUR_WALLET_ADDRESS', // Solana public key
  rpcUrl: 'https://api.mainnet-beta.solana.com', // Or your preferred RPC
  callbacks: {
    onProgress: (progress, message) => {
      console.log(`${progress}% - ${message}`);
    },
    onDone: (data) => {
      console.log('Download completed!', data);
    },
    onError: (error) => {
      console.error('Error:', error);
    }
  }
});

// Start downloading data (limit to 100 transactions)
downloader.download(100)
  .then(data => {
    console.log(`Downloaded ${data.transactions?.length} transactions`);
  })
  .catch(error => {
    console.error('Download failed:', error);
  });
```

## Configuration

The `DataDownloader` accepts a configuration object with the following properties:

```typescript
interface DataDownloaderConfig {
  walletAddress: string;      // Solana public key as string
  rpcUrl: string;             // RPC URL for Solana connection
  callbacks: {
    onDone?: (data: DownloadedData) => void;
    onProgress?: (progress: number, message: string) => void;
    onError?: (error: Error) => void;
  };
}
```

### Using Environment Variables

You can also configure the downloader using environment variables:

```bash
# Set environment variables
export WALLET_ADDRESS=your_wallet_address
export RPC_URL=your_rpc_url

# Then in your code
import { DataDownloader } from 'mercon';

// Create using environment variables
const downloader = DataDownloader.fromEnv();
downloader.download();
```

## Advanced Usage

For more control, you can use the individual services directly:

```typescript
import { 
  TransactionService, 
  MeteoraService, 
  TokenPriceService 
} from 'mercon';

// Initialize services individually
const txService = new TransactionService(rpcUrl, walletAddress);
const meteoraService = new MeteoraService(walletAddress);
const priceService = new TokenPriceService();

// Fetch data using the services directly
const transactions = await txService.fetchAllTransactions(100);
const meteoraData = await meteoraService.fetchAllData();
const prices = await priceService.fetchPrices(['SOL', 'USDC']);
```

## Examples

Check out the examples directory for complete usage examples:

- `examples/basic.ts` - Basic usage with progress tracking
- `examples/advanced.ts` - Advanced usage with individual services

## Development

```bash
# Clone the repository
git clone https://github.com/heywinit/mercon.git
cd mercon

# Install dependencies
bun install

# Run tests
bun test

# Run example
bun run examples/basic.ts
```

## License

Apache 2.0

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
