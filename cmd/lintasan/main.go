package main

import (
	"fmt"
	"os"

	"github.com/sanhaji182/lintasan-go/internal/config"
	"github.com/sanhaji182/lintasan-go/internal/db"
	"github.com/sanhaji182/lintasan-go/internal/server"
	"github.com/spf13/cobra"
)

var version = "2.0.0"

func main() {
	rootCmd := &cobra.Command{
		Use:     "lintasan",
		Short:   "Lintasan - High-performance LLM proxy router",
		Version: version,
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Lintasan proxy server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			database, err := db.Open(cfg.DBPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer database.Close()

			srv := server.New(cfg, database)
			fmt.Printf("🚀 Lintasan v%s listening on :%d\n", version, cfg.Port)
			return srv.Start()
		},
	}

	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Interactive setup wizard",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup()
		},
	}

	mitmCmd := &cobra.Command{
		Use:   "mitm",
		Short: "MITM bridge for IDE interception",
	}

	mitmStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start MITM bridge",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			database, err := db.Open(cfg.DBPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer database.Close()

			return runMITM(cfg, database)
		},
	}

	mitmCmd.AddCommand(mitmStartCmd)
	rootCmd.AddCommand(startCmd, setupCmd, mitmCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSetup() error {
	fmt.Println("🔧 Lintasan Setup Wizard")
	fmt.Println("========================")
	fmt.Println("TODO: Interactive setup")
	return nil
}

func runMITM(cfg *config.Config, database *db.DB) error {
	fmt.Println("🔒 MITM Bridge - TODO")
	return nil
}
