/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
	"github.com/spf13/cobra"
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExport(cfgFile)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(
		&cfgFile,
		"config",
		"c",
		"",
		"path to config file",
	)
	exportCmd.MarkFlagRequired("config")

}

func runExport(configPath string) error {
	client := mangadexapi.NewClient()
	ctx := context.Background()

	err := client.LoadAuth(configPath)
	if err != nil {
		return fmt.Errorf("load auth: %w", err)
	}

	if err := client.Authenticate(ctx); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	fmt.Println("--- Requesting Mangadex Manga ---")

	followedManga, err := client.GetAllFollowed(ctx)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}

	fmt.Printf("Got %d MangaDex manga.\n", len(followedManga))

	t := time.Now()

	file, err := os.Create(fmt.Sprintf("%d-%d-%d-mangadex.txt", t.Year(), t.Month(), t.Day()))
	if err != nil { // Check for an error during file creation
		panic(err)
	}
	defer file.Close()

	for _, manga := range followedManga {
		_, err = fmt.Fprintf(file, "https://mangadex.org/title/%v\n", manga.ID)
		if err != nil {
			return err
		}
	}

	return nil
}
