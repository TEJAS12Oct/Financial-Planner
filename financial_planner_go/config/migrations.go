package config

import (
	dao "FinancialPlannerGo/app/domain/dao/financialPlanner"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// RunDatabaseMigrations creates and updates all application database objects.
//
// It is safe to call on every application startup:
//   - AutoMigrate creates missing tables and adds newly introduced model columns.
//   - EnsureExactPayloadJSONColumns preserves payload columns as PostgreSQL JSON.
//   - EnsureFinancialPlannerSchema creates custom indexes, constraints,
//     foreign keys and other schema objects that cannot be represented exactly
//     through GORM model tags.
func RunDatabaseMigrations(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is required")
	}

	log.Println("starting automatic database migration")

	if err := db.AutoMigrate(
		&dao.FinancialPlannerReport{},
		&dao.FinancialPlannerSelectedGoal{},
	); err != nil {
		return fmt.Errorf("auto migrate tables and columns: %w", err)
	}

	// AutoMigrate does not create PostgreSQL trigger functions/triggers.
	// Create and verify them immediately after the tables exist.
	if err := EnsureUpdatedAtTriggers(db); err != nil {
		return fmt.Errorf("create updated_at triggers: %w", err)
	}

	if err := EnsureExactPayloadJSONColumns(db); err != nil {
		return fmt.Errorf("migrate exact JSON payload columns: %w", err)
	}

	if err := EnsureFinancialPlannerSchema(db); err != nil {
		return fmt.Errorf("apply financial planner schema objects: %w", err)
	}

	log.Println("automatic database migration completed successfully")
	return nil
}
