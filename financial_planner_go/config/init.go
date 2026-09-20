package config

import (
	envconfig "FinancialPlannerGo/config/env_config"
	"log"

	"github.com/joho/godotenv"
)

// Init loads environment variables from the supplied .env file and then
// builds the central typed application configuration.
//
// Application usage:
//
//	config.Init()
//
// Migration usage when executed from the migrate directory:
//
//	config.Init("../.env")
func Init(envFiles ...string) error {
	if len(envFiles) == 0 {
		envFiles = []string{".env"}
	}

	// Ignore missing .env file (Kubernetes provides env vars directly)
	if err := godotenv.Load(envFiles...); err != nil {
		log.Printf("No .env file found, using environment variables: %v", err)
	}

	envconfig.Load()
	return nil
}
