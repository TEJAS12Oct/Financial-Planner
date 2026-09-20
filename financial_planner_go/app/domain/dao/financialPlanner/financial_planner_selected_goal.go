package financialPlanner

import (
	"time"

	"gorm.io/datatypes"
)

type FinancialPlannerSelectedGoal struct {
	ID       uint `gorm:"type:bigint;primaryKey;autoIncrement" json:"id"`
	ReportID uint `gorm:"type:bigint;not null;index" json:"report_id"`

	GoalKey   string `gorm:"size:100;index" json:"goal_key"`
	GoalLabel string `gorm:"size:255" json:"goal_label"`
	GoalType  string `gorm:"size:30;index" json:"goal_type"`

	CategoryTitle string `gorm:"size:255" json:"category_title"`
	CustomName    string `gorm:"size:255" json:"custom_name"`

	Amount float64 `gorm:"type:numeric(18,2);not null;default:0" json:"amount"`
	Years  int     `gorm:"type:bigint" json:"years"`
	Months int     `gorm:"type:bigint" json:"months"`

	Priority string `gorm:"type:text" json:"priority"`
	SubTitle string `gorm:"size:255" json:"sub_title"`
	Title    string `gorm:"type:text" json:"title"`
	Type     string `gorm:"size:100" json:"type"`

	SummaryTitle    string         `gorm:"size:255" json:"summary_title"`
	Summary         datatypes.JSON `gorm:"type:jsonb" json:"summary"`
	Recommendations datatypes.JSON `gorm:"type:jsonb" json:"recommendations"`
	Assumptions     datatypes.JSON `gorm:"type:jsonb" json:"assumptions"`
	Feasibility     datatypes.JSON `gorm:"type:jsonb" json:"feasibility"`
	GoalTable       datatypes.JSON `gorm:"type:jsonb" json:"goal_table"`

	Timeline              string `gorm:"type:text" json:"timeline"`
	TargetAmount          string `gorm:"size:100" json:"target_amount"`
	TargetHorizon         string `gorm:"size:100" json:"target_horizon"`
	FutureCorpusNeeded    string `gorm:"size:100" json:"future_corpus_needed"`
	LumpsumNeededToday    string `gorm:"size:100" json:"lumpsum_needed_today"`
	MonthlySIPRequired    string `gorm:"size:100" json:"monthly_sip_required"`
	YearsMonthsLeft       string `gorm:"size:100" json:"years_months_left"`
	SelectedTimeline      string `gorm:"size:100" json:"selected_timeline"`
	AchievableTimeline    string `gorm:"size:100" json:"achievable_timeline"`
	ExtraTimeRequired     string `gorm:"size:100" json:"extra_time_required"`
	CurrentMonthlySavings string `gorm:"size:100" json:"current_monthly_savings"`
	RequiredMonthlyIncome string `gorm:"size:100" json:"required_monthly_income"`

	AIAdvisoryNote string `gorm:"type:text" json:"ai_advisory_note"`

	ExpectedReturn string `gorm:"size:100" json:"expected_return"`
	InflationUsed  string `gorm:"size:100" json:"inflation_used"`
	PlanningStyle  string `gorm:"size:100" json:"planning_style"`

	GoalName          string `gorm:"size:255" json:"goal_name"`
	GoalTypeDisplay   string `gorm:"size:100" json:"goal_type_display"`
	TargetAmountToday string `gorm:"size:100" json:"target_amount_today"`

	IsRetirement   bool `gorm:"not null;default:false" json:"is_retirement"`
	IsAnnualBudget bool `gorm:"not null;default:false" json:"is_annual_budget"`

	GoalSequence int `gorm:"type:integer;not null;default:1" json:"goal_sequence"`

	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (FinancialPlannerSelectedGoal) TableName() string {
	return "financial_planner_selected_goals"
}
