package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/nghiack7/game-ad-service/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ad-service",
	Short: "Game Ad Service",
	Long:  `Game Ad Service is a service that serves game ads`,
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Load configuration
	var err error
	err = config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Invalid configuration: %v\n", err)
		os.Exit(1)
	}
	if err := config.AppConfig.Validate(); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		os.Exit(1)
	}
}
