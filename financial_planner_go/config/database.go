package config

import (
	envconfig "FinancialPlannerGo/config/env_config"
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectToDB() *gorm.DB {
	if envconfig.AppConfig == nil {
		log.Fatal("application configuration is not initialized; call config.Init() before ConnectToDB()")
	}

	dbConfig := envconfig.AppConfig.Database

	timeZone := strings.TrimSpace(dbConfig.TimeZone)
	if timeZone == "" {
		timeZone = "Asia/Kolkata"
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbConfig.Host,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Name,
		dbConfig.Port,
		dbConfig.SSLMode,
		timeZone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatal("database connection failed: ", err)
	}

	DB = db
	log.Println("database connected successfully")

	return DB
}

// EnsureExactPayloadJSONColumns converts raw payload columns from JSONB to JSON.
// PostgreSQL JSON preserves the original serialized key order and number format;
// JSONB normalizes objects and therefore cannot store byte-for-byte JSON.
func EnsureExactPayloadJSONColumns(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is required")
	}

	columns := []string{"request_payload", "calculation_payload", "response_payload"}
	for _, column := range columns {
		var dataType string
		if err := db.Raw(`
			SELECT data_type
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'financial_planner_reports'
			  AND column_name = ?
		`, column).Scan(&dataType).Error; err != nil {
			return fmt.Errorf("inspect %s type: %w", column, err)
		}
		if dataType == "" {
			continue
		}
		if dataType != "json" && dataType != "jsonb" {
			return fmt.Errorf("unsupported %s type %q", column, dataType)
		}

		indexName := "idx_financial_planner_reports_" + column
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("DROP INDEX IF EXISTS public." + indexName).Error; err != nil {
				return err
			}
			if dataType == "jsonb" {
				stmt := fmt.Sprintf("ALTER TABLE public.financial_planner_reports ALTER COLUMN %s TYPE JSON USING %s::JSON", column, column)
				if err := tx.Exec(stmt).Error; err != nil {
					return err
				}
			}
			stmt := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON public.financial_planner_reports USING GIN ((%s::JSONB))", indexName, column)
			return tx.Exec(stmt).Error
		}); err != nil {
			return fmt.Errorf("migrate %s to JSON: %w", column, err)
		}
	}
	return nil
}

// EnsureUpdatedAtTriggers creates and verifies the PostgreSQL updated_at
// functions/triggers for both Financial Planner tables. GORM AutoMigrate
// creates tables/columns but does not create PostgreSQL triggers, so the DDL
// is executed explicitly after AutoMigrate. This function is idempotent and
// safe to run on every application startup.
func EnsureUpdatedAtTriggers(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is required")
	}

	statements := []string{
		`CREATE OR REPLACE FUNCTION public.set_financial_planner_updated_at()
		 RETURNS TRIGGER AS $$
		 BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		 END;
		 $$ LANGUAGE plpgsql`,

		`CREATE OR REPLACE FUNCTION public.set_financial_planner_reports_updated_at()
		 RETURNS TRIGGER AS $$
		 BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		 END;
		 $$ LANGUAGE plpgsql`,

		`CREATE OR REPLACE FUNCTION public.set_financial_planner_selected_goals_updated_at()
		 RETURNS TRIGGER AS $$
		 BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		 END;
		 $$ LANGUAGE plpgsql`,

		`DROP TRIGGER IF EXISTS trg_financial_planner_reports_updated_at
		 ON public.financial_planner_reports`,
		`CREATE TRIGGER trg_financial_planner_reports_updated_at
		 BEFORE UPDATE ON public.financial_planner_reports
		 FOR EACH ROW
		 EXECUTE FUNCTION public.set_financial_planner_reports_updated_at()`,

		`DROP TRIGGER IF EXISTS trg_financial_planner_selected_goals_updated_at
		 ON public.financial_planner_selected_goals`,
		`CREATE TRIGGER trg_financial_planner_selected_goals_updated_at
		 BEFORE UPDATE ON public.financial_planner_selected_goals
		 FOR EACH ROW
		 EXECUTE FUNCTION public.set_financial_planner_selected_goals_updated_at()`,
	}

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("create financial planner updated_at trigger objects: %w", err)
		}
	}

	var triggerCount int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.triggers
		WHERE trigger_schema = 'public'
		  AND trigger_name IN (
			'trg_financial_planner_reports_updated_at',
			'trg_financial_planner_selected_goals_updated_at'
		  )
	`).Scan(&triggerCount).Error; err != nil {
		return fmt.Errorf("verify financial planner updated_at triggers: %w", err)
	}

	if triggerCount != 2 {
		return fmt.Errorf("updated_at trigger creation incomplete: expected 2 triggers, found %d", triggerCount)
	}

	log.Printf("financial planner updated_at triggers ready (%d/2)", triggerCount)
	return nil
}

// EnsureFinancialPlannerSchema creates the PostgreSQL constraints, indexes,
// foreign key and table-specific updated_at triggers that GORM AutoMigrate
// cannot represent exactly. The statements are idempotent and safe to run on
// every application startup.
func EnsureFinancialPlannerSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is required")
	}

	statements := []string{
		`CREATE INDEX IF NOT EXISTS idx_financial_planner_reports_created_at
		 ON public.financial_planner_reports (created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_financial_planner_reports_selected_goals_json
		 ON public.financial_planner_reports USING GIN (selected_goals_json)`,
		`CREATE INDEX IF NOT EXISTS idx_financial_planner_selected_goals_created_at
		 ON public.financial_planner_selected_goals (created_at DESC)`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_financial_planner_reports_children_count') THEN
				ALTER TABLE public.financial_planner_reports
				ADD CONSTRAINT chk_financial_planner_reports_children_count CHECK (children_count >= 0);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_financial_planner_reports_dependents') THEN
				ALTER TABLE public.financial_planner_reports
				ADD CONSTRAINT chk_financial_planner_reports_dependents CHECK (dependents >= 0);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_financial_planner_reports_spouse_age') THEN
				ALTER TABLE public.financial_planner_reports
				ADD CONSTRAINT chk_financial_planner_reports_spouse_age CHECK (spouse_age >= 0);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_financial_planner_reports_financial_values') THEN
				ALTER TABLE public.financial_planner_reports
				ADD CONSTRAINT chk_financial_planner_reports_financial_values CHECK (
					monthly_income >= 0 AND monthly_expenses >= 0 AND monthly_savings >= 0
					AND spouse_income >= 0 AND existing_life_term_cover >= 0
					AND existing_health_cover >= 0 AND existing_critical_cover >= 0
					AND existing_personal_cover >= 0
				);
			END IF;
		END $$`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_financial_planner_selected_goals_amount') THEN
				ALTER TABLE public.financial_planner_selected_goals
				ADD CONSTRAINT chk_financial_planner_selected_goals_amount CHECK (amount > 0);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_financial_planner_selected_goals_months') THEN
				ALTER TABLE public.financial_planner_selected_goals
				ADD CONSTRAINT chk_financial_planner_selected_goals_months CHECK (months >= 0);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_financial_planner_selected_goals_years') THEN
				ALTER TABLE public.financial_planner_selected_goals
				ADD CONSTRAINT chk_financial_planner_selected_goals_years CHECK (years >= 0);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_financial_planner_selected_goals_sequence') THEN
				ALTER TABLE public.financial_planner_selected_goals
				ADD CONSTRAINT chk_financial_planner_selected_goals_sequence CHECK (goal_sequence > 0);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_financial_planner_goals_report_key') THEN
				ALTER TABLE public.financial_planner_selected_goals
				ADD CONSTRAINT uq_financial_planner_goals_report_key UNIQUE (report_id, goal_key);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_financial_planner_goals_report_sequence') THEN
				ALTER TABLE public.financial_planner_selected_goals
				ADD CONSTRAINT uq_financial_planner_goals_report_sequence UNIQUE (report_id, goal_sequence);
			END IF;
		END $$`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'financial_planner_selected_goals_report_id_fkey') THEN
				ALTER TABLE public.financial_planner_selected_goals
				ADD CONSTRAINT financial_planner_selected_goals_report_id_fkey
				FOREIGN KEY (report_id) REFERENCES public.financial_planner_reports(id)
				ON UPDATE CASCADE ON DELETE CASCADE;
			END IF;
		END $$`,
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("apply financial planner schema statement: %w", err)
			}
		}
		return nil
	})
}
