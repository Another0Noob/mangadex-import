package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Another0Noob/mangadex-import/internal/config"
	"github.com/Another0Noob/mangadex-import/internal/malparser"
	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
	"github.com/Another0Noob/mangadex-import/internal/match"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	// Load configuration
	auth, err := config.LoadAuth("config.ini")
	if err != nil {
		return fmt.Errorf("load auth: %w", err)
	}

	fmt.Println("--- Reading MAL Manga ---")

	// Parse MAL file
	malManga, err := malparser.ParseMALFile("file.xml")
	if err != nil {
		return fmt.Errorf("parse MAL file: %w", err)
	}

	fmt.Printf("Got %d MAL manga.\n", len(malManga))

	// Create MangaDex client
	client := mangadexapi.NewClient()
	ctx := context.Background()

	// Authenticate with MangaDex
	if err := client.Authenticate(ctx, auth); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	fmt.Println("--- Requesting Mangadex Manga ---")

	// Get user's followed manga list from MangaDex
	// Retrieve all followed manga with pagination (limit 100)
	limit := 100
	offset := 0

	firstPage, err := client.GetFollowedMangaList(ctx, mangadexapi.QueryParams{Limit: limit, Offset: offset})
	followedManga := firstPage
	for err == nil && len(firstPage) == limit {
		offset += limit
		firstPage, err = client.GetFollowedMangaList(ctx, mangadexapi.QueryParams{Limit: limit, Offset: offset})
		if err == nil {
			followedManga = append(followedManga, firstPage...)
		}
	}
	if err != nil {
		return fmt.Errorf("get followed manga list: %w", err)
	}

	fmt.Printf("Got %d MangaDex manga.\n", len(followedManga))

	fmt.Println("--- Matching Manga ---")

	matchResult := match.MatchDirect(followedManga, malManga)
	countDirect := len(matchResult.Matches)
	fmt.Printf("Matched %d manga directly.\n", countDirect)

	matchResult = match.FuzzyMatch(matchResult)
	fmt.Printf("Fuzzy matched %d manga.\n", len(matchResult.Matches)-countDirect)

	fmt.Printf("%d MAL manga remaining.\n", len(matchResult.Unmatched.MAL))

	// Search for unmatched manga
	fmt.Println("--- Searching for unmatched manga ---")
	newMatches := 0
	stillUnmatched := []match.MALEntry{}
	for _, malEntry := range matchResult.Unmatched.MAL {
		matchInfo, _, err := match.SearchAndMatch(ctx, client, malEntry)
		if err != nil {
			log.Printf("error searching for %q: %v", malEntry.Original.Title, err)
			stillUnmatched = append(stillUnmatched, malEntry)
			continue
		}

		if matchInfo != nil {
			fmt.Printf("Found new match for %q: %q (%s)\n", malEntry.Original.Title, matchInfo.MangaDexTitle, matchInfo.MatchType)
			newMatches++
		} else {
			fmt.Printf("No match found for %q\n", malEntry.Original.Title)
			stillUnmatched = append(stillUnmatched, malEntry)
		}
	}

	fmt.Printf("\nFound %d new matches.\n", newMatches)
	fmt.Printf("%d manga remain unmatched.\n", len(stillUnmatched))

	return nil
}
