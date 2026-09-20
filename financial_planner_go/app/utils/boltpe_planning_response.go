package utils

type BoltPePlanningResponse struct {
	Success bool                   `json:"success"`
	Data    BoltPePlanningData     `json:"data"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Raw     map[string]interface{} `json:"-"`
}

type GoalReport struct {
	Type            string          `json:"type"`
	Title           string          `json:"title"`
	Priority        string          `json:"priority"`
	SubTitle        string          `json:"sub_title"`
	Recommendations []string        `json:"recommendations"`
	SummaryTitle    string          `json:"summary_title"`
	Summary         GoalSummary     `json:"summary"`
	Assumptions     GoalAssumptions `json:"assumptions"`
	Feasibility     GoalFeasibility `json:"feasibility"`
	Table           []GoalTable     `json:"table"`
}

type GoalSummary struct {
	GoalName      string `json:"goal_name"`
	GoalType      string `json:"goal_type"`
	TargetAmount  string `json:"target_amount"`
	TargetHorizon string `json:"target_horizon"`
	Timeline      string `json:"timeline"`
}

type GoalAssumptions struct {
	ExpectedReturn string `json:"expected_return"`
	InflationUsed  string `json:"inflation_used"`
	PlanningStyle  string `json:"planning_style"`
}

type GoalFeasibility struct {
	Title                         string `json:"title"`
	AIAdvisoryNote                string `json:"ai_advisory_note"`
	AchievableTimelineWithSavings string `json:"achievable_timeline_with_savings"`
	CurrentMonthlySavings         string `json:"current_monthly_savings"`
	ExtraTimeRequired             string `json:"extra_time_required"`
	RequiredMonthlyIncome         string `json:"required_monthly_income"`
	SelectedTimeline              string `json:"selected_timeline"`
}

type GoalTable struct {
	Goal                  string `json:"goal"`
	GoalType              string `json:"goal_type"`
	FutureCorpusNeeded    string `json:"future_corpus_needed"`
	LumpsumNeededToday    string `json:"lumpsum_needed_today"`
	MonthlySavingRequired string `json:"monthly_saving_required"`
	YearsMonthsLeft       string `json:"years_months_left"`
}

type BoltPePlanningData struct {
	CommonRecommendations         []string               `json:"common_recommendations"`
	CommonRecommendationsSubtitle string                 `json:"common_recommendations_subtitle"`
	CommonRecommendationsTitle    string                 `json:"common_recommendations_title"`
	CompanyContact                map[string]interface{} `json:"company_contact"`
	Customer                      map[string]interface{} `json:"customer"`
	DatabaseStatus                BoltPeDBStatus         `json:"database_status"`
	EmailStatus                   map[string]interface{} `json:"email_status"`
	GeneratedAt                   string                 `json:"generated_at"`
	GoalTypeEngine                bool                   `json:"goal_type_engine"`
	InsuranceInvestment           map[string]interface{} `json:"insurance_investment"`
	PDFStatus                     BoltPePDFStatus        `json:"pdf_status"`
	PriorityRoadmap               []string               `json:"priority_roadmap"`
	ReportPDFPath                 string                 `json:"report_pdf_path"`
	ReportPDFURL                  string                 `json:"report_pdf_url"`
	Reports                       []GoalReport           `json:"reports"`
	Score                         BoltPeScore            `json:"score"`
	Summary                       map[string]interface{} `json:"summary"`
}

type BoltPeScore struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}
type BoltPeDBStatus struct {
	ID            int    `json:"id"`
	Saved         bool   `json:"saved"`
	ReportPDFURL  string `json:"report_pdf_url"`
	ReportPDFPath string `json:"report_pdf_path"`
	Message       string `json:"message"`
}
type BoltPePDFStatus struct {
	Saved         bool   `json:"saved"`
	ReportPDFURL  string `json:"report_pdf_url"`
	ReportPDFPath string `json:"report_pdf_path"`
}
