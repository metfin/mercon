# Mercon

Mercon is a data scraper application written in Go that collects data from various sources and stores it in a PostgreSQL database.

## Project Structure

```
mercon/
├── cmd/
│   └── mercon/          # Application entrypoint
│       └── main.go
├── internal/            # Internal packages
│   ├── database/        # Database connection and operations
│   ├── models/          # Database models
│   └── scraper/         # Scraping logic
├── .env.example         # Example environment variables
├── go.mod               # Go module file
├── go.sum               # Go dependencies checksums
└── README.md            # This file
```

## Prerequisites

- Go 1.20 or higher
- PostgreSQL database
- Git

## Setup

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/mercon.git
   cd mercon
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Set up environment variables:
   ```
   cp .env.example .env
   ```
   Edit the `.env` file with your database credentials and other configurations.

4. Run the application:
   ```
   go run cmd/mercon/main.go
   ```

## License

Apache 2.0

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
