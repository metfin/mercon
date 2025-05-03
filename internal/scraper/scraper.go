package scraper

import (
	"fmt"

	"gorm.io/gorm"
)

type Scraper struct {
	db *gorm.DB
}

func NewScraper(db *gorm.DB) *Scraper {
	return &Scraper{
		db: db,
	}
}

func (s *Scraper) Run() error {
	fmt.Println("Starting scraper...")



	return nil
} 