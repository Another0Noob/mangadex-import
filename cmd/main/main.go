package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Another0Noob/mangadex-import/internal/comickparser"
	"github.com/Another0Noob/mangadex-import/internal/config"
	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
	"github.com/Another0Noob/mangadex-import/internal/match"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	fmt.Println("--- Reading Comick Manga ---")

	// Parse Comick file
	comickManga, err := comickparser.ParseComickFile("comick-mylist-2025-11-18.csv")
	if err != nil {
		return fmt.Errorf("parse Comick file: %w", err)
	}

	fmt.Printf("Got %d Comick manga.\n", len(comickManga))

	// Load configuration
	auth, err := config.LoadAuth("config.ini")
	if err != nil {
		return fmt.Errorf("load auth: %w", err)
	}

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
	if err := client.EnsureToken(ctx, auth); err != nil {
		return err
	}
	firstPage, s, err := client.GetFollowedMangaList(ctx, mangadexapi.QueryParams{Limit: limit, Offset: offset})
	followedManga := firstPage
	for err == nil && len(followedManga) != s.Total {
		if err := client.EnsureToken(ctx, auth); err != nil {
			return err
		}
		offset += len(firstPage)
		firstPage, _, err = client.GetFollowedMangaList(ctx, mangadexapi.QueryParams{Limit: limit, Offset: offset})
		if err == nil {
			followedManga = append(followedManga, firstPage...)
		}
	}
	if err != nil {
		return fmt.Errorf("get followed manga list: %w", err)
	}

	fmt.Printf("Got %d MangaDex manga.\n", len(followedManga))

	fmt.Println("--- Matching Manga ---")

	matchResult := match.MatchDirect(followedManga, comickManga)
	countDirect := len(matchResult.Matches)
	fmt.Printf("Matched %d manga directly.\n", countDirect)

	matchResult = match.FuzzyMatch(matchResult)
	fmt.Printf("Fuzzy matched %d manga.\n", len(matchResult.Matches)-countDirect)

	fmt.Printf("%d MAL manga remaining.\n", len(matchResult.Unmatched.Import))

	// Search for unmatched manga
	fmt.Println("--- Searching for unmatched manga ---")
	newMatches := 0
	stillUnmatched := []match.ImportEntry{}
	for _, importEntry := range matchResult.Unmatched.Import {
		if err := client.EnsureToken(ctx, auth); err != nil {
			return err
		}
		matchInfo, id, err := match.SearchAndMatch(ctx, client, importEntry, 10)
		if err != nil {
			log.Printf("error searching for %q: %v", importEntry.Original, err)
			stillUnmatched = append(stillUnmatched, importEntry)
			continue
		}

		if matchInfo != nil {
			fmt.Printf("Found new match for %q: %q (%s)\n", importEntry.Original, matchInfo.MangaDexTitle, matchInfo.MatchType)
			newMatches++
			if err := client.EnsureToken(ctx, auth); err != nil {
				return err
			}
			if err := client.FollowManga(ctx, id); err != nil {
				return err
			}
		} else {
			fmt.Printf("No match found for %q\n", importEntry.Original)
			stillUnmatched = append(stillUnmatched, importEntry)
		}
	}

	fmt.Printf("\nFound %d new matches.\n", newMatches)
	fmt.Printf("%d manga remain unmatched.\n", len(stillUnmatched))

	return nil
}

func test() error {
	// Load configuration
	auth, err := config.LoadAuth("config.ini")
	if err != nil {
		return fmt.Errorf("load auth: %w", err)
	}

	// Create MangaDex client
	client := mangadexapi.NewClient()
	ctx := context.Background()

	// Authenticate with MangaDex
	if err := client.Authenticate(ctx, auth); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	testTitle := "300-nen Fuuinsareshi Jaryuu-chan to Tomodachi ni Narimashita"
	fmt.Println(testTitle)

	params := mangadexapi.QueryParams{
		Title: testTitle,
		Limit: 1,
		Order: mangadexapi.OrderParams{"relevance": "desc"},
	}
	mangas, err := client.GetMangaList(ctx, params)
	if err != nil {
		return err
	}

	if len(mangas) == 0 {
		return errors.New("No search results")
	}

	fmt.Println(mangas)

	testTitle = match.NormalizeTitle(testTitle)

	fmt.Println(testTitle)

	params.Title = match.NormalizeTitle(testTitle)

	mangas2, err2 := client.GetMangaList(ctx, params)
	if err2 != nil {
		return err
	}

	if len(mangas2) == 0 {
		return errors.New("No search results")
	}

	fmt.Println(mangas2)

	return nil
}

func test2() error {
	mangadex := "Please Bully Me, Miss Villainess!"
	comick := "Please Bully Me, Miss Villainess!"
	fmt.Println(match.NormalizeTitle(mangadex))
	fmt.Println(match.NormalizeTitle(comick))

	fmt.Println("--- Reading Comick Manga ---")

	// Parse Comick file
	comickManga, err := comickparser.ParseComickFile("test.csv")
	if err != nil {
		return fmt.Errorf("parse Comick file: %w", err)
	}

	fmt.Println(comickManga)

	id := "8b34f37a-0181-4f0b-8ce3-01217e9a602c"

	auth, err := config.LoadAuth("config.ini")
	if err != nil {
		return fmt.Errorf("load auth: %w", err)
	}

	// Create MangaDex client
	client := mangadexapi.NewClient()
	ctx := context.Background()

	// Authenticate with MangaDex
	if err := client.Authenticate(ctx, auth); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	manga, err := client.GetManga(ctx, id, mangadexapi.QueryParams{})
	if err != nil {
		return err
	}
	fmt.Println(manga)

	firstPage, _, err := client.GetFollowedMangaList(ctx, mangadexapi.QueryParams{Limit: 100, Offset: 0})
	if err != nil {
		return err
	}
	fmt.Println(firstPage)

	return nil
}
