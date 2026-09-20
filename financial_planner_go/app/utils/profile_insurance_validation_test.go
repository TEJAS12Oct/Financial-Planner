package utils

import (
	"strings"
	"testing"
)

func validBaseRequestForProfileValidation() FinancialPlannerRequest {
	return FinancialPlannerRequest{
		FullName:              "Sample User",
		Contact:               "9999999999",
		Email:                 "sample.user@example.com",
		DOB:                   "01/01/1990",
		City:                  "Sample City",
		MaritalStatus:         "single",
		ChildrenCount:         "0",
		MonthlyIncome:         "100000",
		MonthlyExpenses:       "30000",
		MonthlySavings:        "60000",
		RiskAppetite:          "medium",
		InvestmentExperience:  "Intermediate",
		ExistingLifeTerm:      "no",
		ExistingLifeTermCover: "0",
		ExistingHealth:        "no",
		ExistingHealthCover:   "0",
		ExistingCritical:      "no",
		ExistingCriticalCover: "0",
		ExistingPersonal:      "no",
		ExistingPersonalCover: "0",
		SelectedGoals: []SelectedGoal{
			{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
		},
	}
}

func TestValidateRequestRejectsInvalidInvestmentExperience(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.InvestmentExperience = "Expert"

	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "investment_experience") {
		t.Fatalf("expected investment experience error, got: %v", err)
	}
}

func TestValidateRequestRejectsInvalidInsuranceSelection(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.ExistingHealth = "maybe"

	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "existing_health must be one of") {
		t.Fatalf("expected insurance selection error, got: %v", err)
	}
}

func TestValidateRequestRequiresPositiveCoverWhenInsuranceSelected(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.ExistingLifeTerm = "yes"
	req.ExistingLifeTermCover = "0"

	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "existing_life_term_cover must be greater than 0") {
		t.Fatalf("expected positive cover error, got: %v", err)
	}
}

func TestValidateRequestRequiresZeroCoverWhenInsuranceNotSelected(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.ExistingPersonal = "no"
	req.ExistingPersonalCover = "80000"

	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "existing_personal_cover must be 0") {
		t.Fatalf("expected zero cover error, got: %v", err)
	}
}
func TestInstantOnlyGoalsRequireFamilyFieldsToBeEmptyOrZero(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.SelectedGoals = []SelectedGoal{{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12}}
	req.MaritalStatus = ""
	req.Dependents = "0"
	req.SpouseAge = "0"
	req.SpouseIncome = "0"
	req.ChildrenCount = "0"
	req.Children = nil
	NormalizeFinancialPlannerRequest(&req)

	if err := ValidateFinancialPlannerRequest(req); err != nil {
		t.Fatalf("expected instant-only request with empty/zero family details to pass, got: %v", err)
	}

	req.ChildrenCount = "2"
	if err := ValidateFinancialPlannerRequest(req); err == nil || !strings.Contains(err.Error(), "Family details are not applicable for instant goals") {
		t.Fatalf("expected instant-only family validation error, got: %v", err)
	}
}
func TestExplorerDoesNotUseInstantOnlyFamilyValidation(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MaritalStatus = "married"
	req.SpouseAge = "30"
	req.SpouseIncome = "0"
	req.ChildrenCount = "2"
	req.Dependents = "2"
	req.SelectedGoals = []SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
		{Key: "retirement", Label: "Retirement Planning", Type: "long", Amount: 20000000, Years: 25},
	}
	NormalizeFinancialPlannerRequest(&req)
	if err := ValidateFinancialPlannerRequest(req); err != nil {
		t.Fatalf("expected Explorer request with family details to pass instant-only validation, got: %v", err)
	}
}
func TestShortGoalStillRequiresMaritalStatus(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MaritalStatus = ""
	req.Dependents = "0"
	req.SelectedGoals = []SelectedGoal{
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
	}
	NormalizeFinancialPlannerRequest(&req)
	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "marital_status must be single or married") {
		t.Fatalf("expected marital status validation for short goal, got: %v", err)
	}
}
func TestLongGoalStillRequiresMaritalStatus(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MaritalStatus = ""
	req.Dependents = "0"
	req.SelectedGoals = []SelectedGoal{
		{Key: "retirement", Label: "Retirement Planning", Type: "long", Amount: 20000000, Years: 25},
	}
	NormalizeFinancialPlannerRequest(&req)
	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "marital_status must be single or married") {
		t.Fatalf("expected marital status validation for long goal, got: %v", err)
	}
}

func TestMarriedSpouseAge18IsAccepted(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MaritalStatus = "married"
	req.SpouseAge = "18"
	req.SpouseIncome = "0"
	req.ChildrenCount = "0"

	if err := ValidateFinancialPlannerRequest(req); err != nil {
		t.Fatalf("expected spouse_age=18 to pass, got: %v", err)
	}
}

func TestDecimalCashflowComparison(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MonthlyIncome = "100000.50"
	req.MonthlyExpenses = "60000.25"
	req.MonthlySavings = "40000.26"

	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "monthly_expenses plus monthly_savings") {
		t.Fatalf("expected decimal cashflow total validation error, got: %v", err)
	}
}

func TestAssumptionUpperBounds(t *testing.T) {
	tests := []struct {
		name string
		set  func(*FinancialPlannerRequest)
		want string
	}{
		{"inflation", func(r *FinancialPlannerRequest) { r.Education1Inflation = "20.01" }, "education_1_inflation must be between 0 and 20"},
		{"return", func(r *FinancialPlannerRequest) { r.WealthSIPReturn = "50.01" }, "wealth_sip_return must be between 0 and 50"},
		{"target age", func(r *FinancialPlannerRequest) { r.Education1TargetAge = "101" }, "education_1_target_age must be between 1 and 100"},
		{"target age whole", func(r *FinancialPlannerRequest) { r.Education1TargetAge = "25.5" }, "education_1_target_age must be a whole number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validBaseRequestForProfileValidation()
			tt.set(&req)
			err := ValidateFinancialPlannerRequest(req)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got: %v", tt.want, err)
			}
		})
	}
}

func TestInstantFlowRejectsInsuranceInsteadOfClearingIt(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.SelectedGoals = []SelectedGoal{{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12}}
	req.MaritalStatus = ""
	req.Dependents = "0"
	req.SpouseAge = "0"
	req.SpouseIncome = "0"
	req.ChildrenCount = "0"
	req.Children = nil
	req.ExistingHealth = "yes"
	req.ExistingHealthCover = "500000"
	NormalizeFinancialPlannerRequest(&req)

	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "insurance details are not applicable for instant goals") {
		t.Fatalf("expected instant insurance validation error, got: %v", err)
	}
}
