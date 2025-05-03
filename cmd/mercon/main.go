package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/wnt/mercon/internal/database"
	"github.com/wnt/mercon/internal/scraper"
)

func main() {
	// Parse command-line arguments
	envFile := flag.String("envFile", ".env", "Path to .env file")
	flag.Parse()

	// Load environment variables from the specified file
	if err := godotenv.Load(*envFile); err != nil {
		log.Printf("No .env file found at %s, using environment variables", *envFile)
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	s, err := scraper.NewScraper(db)
	if err != nil {
		log.Fatalf("Failed to initialize scraper: %v", err)
	}

	if err := s.Run(); err != nil {
		log.Fatalf("Scraper failed: %v", err)
	}

	fmt.Println("Scraping completed successfully")
}
