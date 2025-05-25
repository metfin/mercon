package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/wnt/mercon/internal/database"
	"github.com/wnt/mercon/internal/scraper"
	"github.com/wnt/mercon/internal/services"
)

func main() {
	// Parse command-line arguments
	envFile := flag.String("envFile", ".env", "Path to .env file")
	flag.Parse()

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to listen for termination signals
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %s. Starting graceful shutdown...", sig)
		cancel()
	}()

	// Try to load environment variables from the specified file
	if err := godotenv.Load(*envFile); err != nil {
		log.Printf("No .env file found at %s, using environment variables", *envFile)
	}

	// Initialize database connection
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize the data enricher service
	dataEnricher := services.NewMeteoraDataEnricher(db)

	// Start a separate goroutine for the data enrichment service
	enrichmentTicker := time.NewTicker(2 * time.Hour)
	go func() {
		// Run an initial enrichment
		if err := dataEnricher.EnrichPairs(); err != nil {
			log.Printf("Initial data pairs enrichment failed: %v", err)
		}
		if err := dataEnricher.EnrichPositions(); err != nil {
			log.Printf("Initial data positions enrichment failed: %v", err)
		}

		// Set up periodic updates
		for {
			select {
			case <-enrichmentTicker.C:
				log.Println("Running scheduled data enrichment...")
				if err := dataEnricher.EnrichPairs(); err != nil {
					log.Printf("Pairs data enrichment failed: %v", err)
				}
				if err := dataEnricher.EnrichPositions(); err != nil {
					log.Printf("Positions data enrichment failed: %v", err)
				}
			case <-ctx.Done():
				log.Println("Stopping data enrichment service...")
				enrichmentTicker.Stop()
				return
			}
		}
	}()
	log.Println("Started USD data enrichment service")

	// Initialize the scraper
	s, err := scraper.NewScraper(db)
	if err != nil {
		log.Fatalf("Failed to initialize scraper: %v", err)
	}

	// Run the scraper with context for cancellation
	if err := runScraper(ctx, s); err != nil {
		log.Fatalf("Scraper failed: %v", err)
	}

	log.Println("Program completed successfully")
}

// runScraper runs the scraper with context support
func runScraper(ctx context.Context, s *scraper.Scraper) error {
	// Create a channel to capture the result of the scraper
	resultChan := make(chan error, 1)

	// Run the scraper in a goroutine
	go func() {
		resultChan <- s.RunWithContext(ctx)
	}()

	// Wait for either the scraper to complete or context to be canceled
	select {
	case err := <-resultChan:
		return err
	case <-ctx.Done():
		log.Println("Scraper interrupted, shutting down...")
		return ctx.Err()
	}
}
