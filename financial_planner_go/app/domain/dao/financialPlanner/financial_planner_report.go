package financialPlanner

import (
	"time"

	"gorm.io/datatypes"
)

type FinancialPlannerReport struct {
	ID     uint `gorm:"type:bigint;primaryKey;autoIncrement" json:"id"`
	UserID uint `gorm:"type:bigint;index" json:"user_id"`

	FullName string `gorm:"size:255;not null" json:"full_name"`
	Contact  string `gorm:"size:20;not null" json:"contact"`
	Email    string `gorm:"size:255;not null;index" json:"email"`
	DOB      string `gorm:"size:20;not null" json:"dob"`
	City     string `gorm:"size:100;not null;index" json:"city"`

	MaritalStatus string `gorm:"size:30" json:"marital_status"`
	Dependents    int    `gorm:"type:integer;not null;default:0" json:"dependents"`
	ChildrenCount int    `gorm:"type:integer;not null;default:0" json:"children_count"`

	MonthlyIncome   float64 `gorm:"type:numeric(18,2);not null;default:0" json:"monthly_income"`
	MonthlyExpenses float64 `gorm:"type:numeric(18,2);not null;default:0" json:"monthly_expenses"`
	MonthlySavings  float64 `gorm:"type:numeric(18,2);not null;default:0" json:"monthly_savings"`

	SpouseIncome float64 `gorm:"type:numeric(18,2);not null;default:0" json:"spouse_income"`
	SpouseAge    int     `gorm:"type:bigint" json:"spouse_age"`

	RiskAppetite         string `gorm:"size:30" json:"risk_appetite"`
	InvestmentExperience string `gorm:"size:50" json:"investment_experience"`

	ExistingLifeTerm bool `gorm:"not null;default:false" json:"existing_life_term"`
	ExistingHealth   bool `gorm:"not null;default:false" json:"existing_health"`
	ExistingCritical bool `gorm:"not null;default:false" json:"existing_critical"`
	ExistingPersonal bool `gorm:"not null;default:false" json:"existing_personal"`

	ExistingLifeTermCover float64 `gorm:"type:numeric(18,2);not null;default:0" json:"existing_life_term_cover"`
	ExistingHealthCover   float64 `gorm:"type:numeric(18,2);not null;default:0" json:"existing_health_cover"`
	ExistingCriticalCover float64 `gorm:"type:numeric(18,2);not null;default:0" json:"existing_critical_cover"`
	ExistingPersonalCover float64 `gorm:"type:numeric(18,2);not null;default:0" json:"existing_personal_cover"`

	Education1Inflation     float64 `gorm:"type:numeric(5,2);not null;default:0" json:"education_1_inflation"`
	Education1LumpsumReturn float64 `gorm:"type:numeric(5,2);not null;default:0" json:"education_1_lumpsum_return"`
	Education1SIPReturn     float64 `gorm:"type:numeric(5,2);not null;default:0" json:"education_1_sip_return"`
	Education1TargetAge     int     `gorm:"type:bigint" json:"education_1_target_age"`

	Marriage1Inflation     float64 `gorm:"type:numeric(5,2);not null;default:0" json:"marriage_1_inflation"`
	Marriage1LumpsumReturn float64 `gorm:"type:numeric(5,2);not null;default:0" json:"marriage_1_lumpsum_return"`
	Marriage1SIPReturn     float64 `gorm:"type:numeric(5,2);not null;default:0" json:"marriage_1_sip_return"`
	Marriage1TargetAge     int     `gorm:"type:bigint" json:"marriage_1_target_age"`

	RetirementInflation  float64 `gorm:"type:numeric(5,2);not null;default:0" json:"retirement_inflation"`
	RetirementPreReturn  float64 `gorm:"type:numeric(5,2);not null;default:0" json:"retirement_pre_return"`
	RetirementPostReturn float64 `gorm:"type:numeric(5,2);not null;default:0" json:"retirement_post_return"`

	WealthInflation     float64 `gorm:"type:numeric(5,2);not null;default:0" json:"wealth_inflation"`
	WealthLumpsumReturn float64 `gorm:"type:numeric(5,2);not null;default:0" json:"wealth_lumpsum_return"`
	WealthSIPReturn     float64 `gorm:"type:numeric(5,2);not null;default:0" json:"wealth_sip_return"`

	GenericGoalInflation     float64 `gorm:"type:numeric(5,2);not null;default:0" json:"generic_goal_inflation"`
	GenericGoalLumpsumReturn float64 `gorm:"type:numeric(5,2);not null;default:0" json:"generic_goal_lumpsum_return"`
	GenericGoalSIPReturn     float64 `gorm:"type:numeric(5,2);not null;default:0" json:"generic_goal_sip_return"`

	ScoreValue int    `gorm:"type:bigint" json:"score_value"`
	ScoreLabel string `gorm:"size:100" json:"score_label"`

	Status string `gorm:"size:50;not null;default:'report_requested';index" json:"status"`

	RequestType              string `gorm:"size:50;index" json:"request_type,omitempty"`
	CalculationEngine        string `gorm:"type:text" json:"calculation_engine,omitempty"`
	CalculationStatusCode    int    `gorm:"type:integer;not null;default:0" json:"calculation_status_code,omitempty"`
	CalculationErrorCode     string `gorm:"size:100" json:"calculation_error_code,omitempty"`
	CalculationErrorMessage  string `gorm:"type:text" json:"calculation_error_message,omitempty"`
	FailedModule             string `gorm:"size:100" json:"failed_module,omitempty"`
	CalculationEngineVersion string `gorm:"size:50" json:"calculation_engine_version,omitempty"`
	AssumptionVersion        string `gorm:"size:50" json:"assumption_version,omitempty"`
	ResponseSchemaVersion    string `gorm:"size:50" json:"response_schema_version,omitempty"`
	PDFStatus                string `gorm:"size:50;index" json:"pdf_status,omitempty"`
	StorageStatus            string `gorm:"size:50;index" json:"storage_status,omitempty"`

	ReportPDFURL  string `gorm:"type:text" json:"report_pdf_url"`
	ReportPDFPath string `gorm:"type:text" json:"report_pdf_path"`

	ChildrenJSON       datatypes.JSON `gorm:"type:jsonb" json:"children"`
	SelectedGoalsJSON  datatypes.JSON `gorm:"type:jsonb" json:"selected_goals_json"`
	RequestPayload     datatypes.JSON `gorm:"type:json" json:"-"`
	CalculationPayload datatypes.JSON `gorm:"type:json" json:"-"`
	ResponsePayload    datatypes.JSON `gorm:"type:json" json:"-"`

	CommonRecommendationsTitle    string         `gorm:"type:text" json:"common_recommendations_title"`
	CommonRecommendationsSubtitle string         `gorm:"type:text" json:"common_recommendations_subtitle"`
	CommonRecommendations         datatypes.JSON `gorm:"type:jsonb" json:"common_recommendations"`
	PriorityRoadmap               datatypes.JSON `gorm:"type:jsonb" json:"priority_roadmap"`
	CompanyContact                datatypes.JSON `gorm:"type:jsonb" json:"company_contact"`
	Customer                      datatypes.JSON `gorm:"type:jsonb" json:"customer"`
	InsuranceInvestment           datatypes.JSON `gorm:"type:jsonb" json:"insurance_investment"`
	EmailStatus                   datatypes.JSON `gorm:"type:jsonb" json:"email_status"`
	DatabaseStatus                datatypes.JSON `gorm:"type:jsonb" json:"database_status"`

	GeneratedAt *time.Time `json:"generated_at"`

	GoalTypeEngine bool `gorm:"not null;default:false" json:"goal_type_engine"`

	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	SelectedGoals []FinancialPlannerSelectedGoal `gorm:"foreignKey:ReportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"selected_goals,omitempty"`
	CreatedAt     time.Time                      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time                      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (FinancialPlannerReport) TableName() string {
	return "financial_planner_reports"
}
