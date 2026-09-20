package models

type FinancialPlannerRequest struct {
	GoalSelectionMode string `json:"goal_selection_mode,omitempty" example:"create_your_own" enums:"instant,short,long,create_your_own"`
	FullName          string `json:"full_name" example:"Sample User"`
	Contact           string `json:"contact" example:"9999999999"`
	Email             string `json:"email" example:"sample.user@example.com"`
	DOB               string `json:"dob" example:"01/01/1990"`
	City              string `json:"city" example:"Sample City"`

	MaritalStatus string  `json:"marital_status" example:"single"`
	Dependents    int     `json:"dependents" example:"2"`
	SpouseIncome  float64 `json:"spouse_income" example:"0"`
	SpouseDOB     string  `json:"spouse_dob" example:"15/06/1998"`
	Child1Name    string  `json:"child_1_name,omitempty" example:"Aarav"`
	Child1DOB     string  `json:"child_1_dob,omitempty" example:"05/08/2022"`
	Child2Name    string  `json:"child_2_name,omitempty" example:"Anaya"`
	Child2DOB     string  `json:"child_2_dob,omitempty" example:"10/02/2024"`
	ChildrenCount int     `json:"children_count" example:"0"`

	MonthlyIncome   float64 `json:"monthly_income" example:"100000"`
	MonthlyExpenses float64 `json:"monthly_expenses" example:"30000"`
	MonthlySavings  float64 `json:"monthly_savings" example:"60000"`

	RiskAppetite         string `json:"risk_appetite" example:"medium" enums:"low,medium,high"`
	InvestmentExperience string `json:"investment_experience" example:"Intermediate" enums:"Beginner,Intermediate,Advance"`

	ExistingLifeTerm string `json:"existing_life_term" example:"yes" enums:"yes,no,true,false,1,0"`

	ExistingHealth        string  `json:"existing_health" example:"yes" enums:"yes,no,true,false,1,0"`
	ExistingCritical      string  `json:"existing_critical" example:"yes" enums:"yes,no,true,false,1,0"`
	ExistingPersonal      string  `json:"existing_personal" example:"yes" enums:"yes,no,true,false,1,0"`
	ExistingLifeTermCover float64 `json:"existing_life_term_cover" example:"10000000"`
	ExistingHealthCover   float64 `json:"existing_health_cover" example:"100000"`
	ExistingCriticalCover float64 `json:"existing_critical_cover" example:"100000"`
	ExistingPersonalCover float64 `json:"existing_personal_cover" example:"80000"`

	Education1TargetAge     int     `json:"education_1_target_age" example:"18"`
	Education1Inflation     float64 `json:"education_1_inflation" example:"6"`
	Education1SIPReturn     float64 `json:"education_1_sip_return" example:"12"`
	Education1LumpsumReturn float64 `json:"education_1_lumpsum_return" example:"10"`

	Marriage1TargetAge     int     `json:"marriage_1_target_age" example:"25"`
	Marriage1Inflation     float64 `json:"marriage_1_inflation" example:"6"`
	Marriage1SIPReturn     float64 `json:"marriage_1_sip_return" example:"12"`
	Marriage1LumpsumReturn float64 `json:"marriage_1_lumpsum_return" example:"10"`

	RetirementInflation  float64 `json:"retirement_inflation" example:"6"`
	RetirementPreReturn  float64 `json:"retirement_pre_return" example:"12"`
	RetirementPostReturn float64 `json:"retirement_post_return" example:"7"`

	WealthInflation     float64 `json:"wealth_inflation" example:"6"`
	WealthSIPReturn     float64 `json:"wealth_sip_return" example:"12"`
	WealthLumpsumReturn float64 `json:"wealth_lumpsum_return" example:"10"`

	GenericGoalInflation     float64 `json:"generic_goal_inflation" example:"6"`
	GenericGoalSIPReturn     float64 `json:"generic_goal_sip_return" example:"12"`
	GenericGoalLumpsumReturn float64 `json:"generic_goal_lumpsum_return" example:"10"`

	SelectedGoals []SelectedGoalRequest `json:"selected_goals_json"`
}

type SelectedGoalRequest struct {
	Key           string  `json:"key" example:"home_purchase"`
	Label         string  `json:"label" example:"Home Purchase / Home Upgrade"`
	Type          string  `json:"type" example:"short" enums:"instant,short,long"`
	CategoryTitle string  `json:"categoryTitle" example:"Short Term Goals"`
	Timeline      string  `json:"timeline" example:"5 Years"`
	Amount        float64 `json:"amount" example:"2500000"`
	Years         int     `json:"years" example:"5" minimum:"0"`
	Months        int     `json:"months" example:"0" minimum:"0"`
	CustomName    string  `json:"customName" example:""`
}
