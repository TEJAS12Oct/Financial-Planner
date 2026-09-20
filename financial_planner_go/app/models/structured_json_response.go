package models

type StructuredJSONReportResponse struct {
	APIVersion   string                   `json:"api_version"`
	Data         StructuredJSONReportData `json:"data"`
	Message      string                   `json:"message"`
	ResponseType string                   `json:"response_type"`
	Success      bool                     `json:"success"`
}

type StructuredJSONReportData struct {
	CommonRecommendations []string `json:"common_recommendations"`

	CommonRecommendationsSubtitle string `json:"common_recommendations_subtitle"`

	CommonRecommendationsTitle string `json:"common_recommendations_title"`

	CompanyContact map[string]interface{} `json:"company_contact"`

	Customer map[string]interface{} `json:"customer"`

	GeneratedAt string `json:"generated_at"`

	GoalTypeEngine bool `json:"goal_type_engine"`

	InsuranceInvestment map[string]interface{} `json:"insurance_investment"`

	PriorityRoadmap []string `json:"priority_roadmap"`

	Reports []StructuredJSONGoalReport `json:"reports"`

	Score StructuredJSONScore `json:"score"`

	Summary map[string]interface{} `json:"summary"`
}

type StructuredJSONScore struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type StructuredJSONGoalReport struct {
	Assumptions StructuredJSONGoalAssumptions `json:"assumptions"`

	Feasibility StructuredJSONGoalFeasibility `json:"feasibility"`

	Priority string `json:"priority"`

	Recommendations []string `json:"recommendations"`

	SubTitle string `json:"sub_title"`

	Summary StructuredJSONGoalSummary `json:"summary"`

	SummaryTitle string `json:"summary_title"`

	Table []StructuredJSONGoalTable `json:"table"`

	Title string `json:"title"`

	Type string `json:"type"`
}

type StructuredJSONGoalAssumptions struct {
	ExpectedReturn string `json:"Expected Return"`
	InflationUsed  string `json:"Inflation Used"`
	PlanningStyle  string `json:"Planning Style"`
}

type StructuredJSONGoalFeasibility struct {
	AIAdvisoryNote string `json:"Boltpe Advisory Note"`

	AchievableTimeline string `json:"Achievable Timeline With Current Savings"`

	CurrentMonthlySavings string `json:"Current Monthly Savings"`

	ExtraTimeRequired string `json:"Extra Time Required"`

	RequiredMonthlyIncome string `json:"Required Monthly Income To Keep Same Timeline"`

	SelectedTimeline string `json:"Selected Timeline"`

	Title string `json:"title"`
}

type StructuredJSONGoalSummary struct {
	GoalName string `json:"Goal Name"`

	GoalType string `json:"Goal Type"`

	TargetAmount string `json:"Target Amount / Today's Cost"`

	TargetHorizon string `json:"Target Horizon"`

	Timeline string `json:"Timeline"`
}

type StructuredJSONGoalTable struct {
	FutureCorpusNeeded string `json:"Future Corpus Needed"`

	Goal string `json:"Goal"`

	GoalType string `json:"Goal Type"`

	LumpsumNeededToday string `json:"Lumpsum Needed Today"`

	MonthlySavingRequired string `json:"Monthly Saving / SIP Required"`

	YearsMonthsLeft string `json:"Years / Months Left"`
}
