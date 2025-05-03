package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/wnt/mercon/internal/database"
	"github.com/wnt/mercon/internal/scraper"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
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