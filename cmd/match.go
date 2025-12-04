/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
	"github.com/Another0Noob/mangadex-import/internal/mangaparser"
	"github.com/Another0Noob/mangadex-import/internal/match"
	"github.com/spf13/cobra"
)

// matchCmd represents the match command
var matchCmd = &cobra.Command{
	Use:   "match",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMatch(cfgFile, inputFile)
	},
}

func init() {
	rootCmd.AddCommand(matchCmd)

	matchCmd.Flags().StringVarP(
		&cfgFile,
		"config",
		"c",
		"",
		"path to config file",
	)
	matchCmd.MarkFlagRequired("config")

	matchCmd.Flags().StringVarP(
		&inputFile,
		"input",
		"i",
		"",
		"path to input file",
	)
	matchCmd.MarkFlagRequired("input")
}

func runMatch(configPath, inputPath string) error {
	fmt.Println("--- Reading Manga ---")

	inputManga, err := mangaparser.Parse(inputPath)
	if err != nil {
		return fmt.Errorf("parse file: %w", err)
	}

	fmt.Printf("Got %d manga.\n", len(inputManga))

	auth, err := mangadexapi.LoadAuth(configPath)
	if err != nil {
		return fmt.Errorf("load auth: %w", err)
	}

	client := mangadexapi.NewClient()
	ctx := context.Background()

	if err := client.Authenticate(ctx, &auth); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	fmt.Println("--- Requesting Mangadex Manga ---")

	followedManga, err := client.GetAllFollowed(ctx, &auth)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}

	fmt.Printf("Got %d MangaDex manga.\n", len(followedManga))

	fmt.Println("--- Matching Manga ---")

	matchResult := match.MatchDirect(followedManga, inputManga)
	countDirect := len(matchResult.Matches)
	fmt.Printf("Matched %d manga directly.\n", countDirect)

	matchResult = match.FuzzyMatch(matchResult)
	fmt.Printf("Fuzzy matched %d manga.\n", len(matchResult.Matches)-countDirect)

	fmt.Printf("%d MAL manga remaining.\n", len(matchResult.Unmatched.Import))

	// Search for unmatched manga
	fmt.Println("--- Searching for unmatched manga ---")

	newMatches, stillUnmatched, err := match.SearchAndFollow(ctx, client, &auth, matchResult.Unmatched.Import, false)
	if err != nil {
		return fmt.Errorf("Search: %w", err)
	}

	fmt.Printf("\nFound %d new matches.\n", len(newMatches))
	fmt.Printf("%d manga remain unmatched.\n", len(stillUnmatched))

	return nil
}
