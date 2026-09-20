package main

import (
	"FinancialPlannerGo/config"
	"log"
)

func main() {
	if err := config.Init("../.env"); err != nil {
		log.Fatal("configuration initialization failed: ", err)
	}

	db := config.ConnectToDB()

	if err := config.RunDatabaseMigrations(db); err != nil {
		log.Fatal("database migration failed: ", err)
	}
}
