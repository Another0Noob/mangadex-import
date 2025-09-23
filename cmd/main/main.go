package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Another0Noob/mangadex-import/internal/config"
	"github.com/Another0Noob/mangadex-import/internal/malparser"
	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	// Example: parse MAL file
	malPtr, err := malparser.ParseMALFile("file.xml")
	if err != nil {
		return fmt.Errorf("parse MAL file: %w", err)
	}
	for _, manga := range malPtr.Entries {
		fmt.Println(manga.Title)
	}

	// Authenticate with MangaDex
	if err := apiTest(); err != nil {
		return err
	}

	return nil
}

func apiTest() error {
	auth, err := config.LoadAuth("config.ini")
	if err != nil {
		return fmt.Errorf("load auth: %w", err)
	}

	c := mangadexapi.NewClient()
	ctx := context.Background()

	if err := c.Authenticate(ctx, auth); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	fmt.Println("Authenticated successfully")
	return nil
}
