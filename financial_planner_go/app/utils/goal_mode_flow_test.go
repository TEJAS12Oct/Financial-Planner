package utils

import (
	"strings"
	"testing"
)

func TestGoalFlowIsInferredFromSelectedGoals(t *testing.T) {
	tests := []struct {
		name  string
		goals []SelectedGoal
		want  string
	}{
		{"instant", []SelectedGoal{{Type: "instant"}}, "instant"},
		{"short", []SelectedGoal{{Type: "short"}}, "short"},
		{"long", []SelectedGoal{{Type: "long"}}, "long"},
		{"explorer", []SelectedGoal{{Type: "instant"}, {Type: "short"}}, "explorer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectGoalFlow(tt.goals); got != tt.want {
				t.Fatalf("detectGoalFlow() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExplorerMixedGoalsRequireDependents(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.Dependents = ""
	req.SelectedGoals = []SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
	}

	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "Explorer flow") {
		t.Fatalf("expected Explorer dependents validation, got: %v", err)
	}
}

func TestInstantFlowRejectsFamilyDataWithoutSilentlyClearingInsurance(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MaritalStatus = ""
	req.Dependents = "0"
	req.SpouseAge = "0"
	req.SpouseIncome = "0"
	req.ChildrenCount = "2"
	req.ExistingLifeTerm = "yes"
	req.ExistingLifeTermCover = "10000000"
	req.SelectedGoals = []SelectedGoal{{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12}}

	NormalizeFinancialPlannerRequest(&req)

	if req.ExistingLifeTerm != "yes" || req.ExistingLifeTermCover != "10000000" {
		t.Fatalf("instant insurance values must remain untouched until validation")
	}
	if err := ValidateFinancialPlannerRequest(req); err == nil || !strings.Contains(err.Error(), "Family details are not applicable for instant goals") {
		t.Fatalf("expected instant family validation error, got: %v", err)
	}
}

func TestShortFlowRequiresChildrenCountButAllowsZero(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MaritalStatus = "married"
	req.SpouseAge = "30"
	req.SpouseIncome = "0"
	req.ChildrenCount = "0"
	req.SelectedGoals = []SelectedGoal{
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
	}

	if err := ValidateFinancialPlannerRequest(req); err != nil {
		t.Fatalf("expected short flow with children_count=0 to pass, got: %v", err)
	}

	req.ChildrenCount = ""
	if err := ValidateFinancialPlannerRequest(req); err == nil || !strings.Contains(err.Error(), "children_count is required") {
		t.Fatalf("expected short flow to require children_count field, got: %v", err)
	}
}

func TestLongFlowRequiresChildrenCountButAllowsZero(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MaritalStatus = "married"
	req.SpouseAge = "30"
	req.SpouseIncome = "0"
	req.ChildrenCount = "0"
	req.SelectedGoals = []SelectedGoal{
		{Key: "retirement", Label: "Retirement Planning", Type: "long", Amount: 20000000, Years: 25},
	}

	if err := ValidateFinancialPlannerRequest(req); err != nil {
		t.Fatalf("expected long flow with children_count=0 to pass, got: %v", err)
	}
}

func TestExplorerRequiresMaritalChildrenAndDependents(t *testing.T) {
	base := validBaseRequestForProfileValidation()
	base.SelectedGoals = []SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
	}

	req := base
	req.Dependents = "0"
	req.ChildrenCount = "0"
	if err := ValidateFinancialPlannerRequest(req); err != nil {
		t.Fatalf("expected Explorer flow to accept dependents=0 and children_count=0, got: %v", err)
	}

	req = base
	req.Dependents = "0"
	req.ChildrenCount = ""
	if err := ValidateFinancialPlannerRequest(req); err == nil || !strings.Contains(err.Error(), "children_count is required") {
		t.Fatalf("expected Explorer flow to require children_count, got: %v", err)
	}

	req = base
	req.MaritalStatus = ""
	req.Dependents = "0"
	req.ChildrenCount = "0"
	if err := ValidateFinancialPlannerRequest(req); err == nil || !strings.Contains(err.Error(), "marital_status") {
		t.Fatalf("expected Explorer flow to require marital_status, got: %v", err)
	}
}

func TestNonInstantSingleKeepsExistingFamilyValidation(t *testing.T) {
	flows := []struct {
		name string
		goal SelectedGoal
	}{
		{"short", SelectedGoal{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3}},
		{"long", SelectedGoal{Key: "retirement", Label: "Retirement Planning", Type: "long", Amount: 20000000, Years: 25}},
	}

	for _, tt := range flows {
		t.Run(tt.name, func(t *testing.T) {
			req := validBaseRequestForProfileValidation()
			req.MaritalStatus = "single"
			req.ChildrenCount = "0"
			req.SpouseAge = "0"
			req.SpouseIncome = "0"
			req.SelectedGoals = []SelectedGoal{tt.goal}

			if err := ValidateFinancialPlannerRequest(req); err != nil {
				t.Fatalf("expected %s single flow to pass with zero spouse/children values, got: %v", tt.name, err)
			}

			req.SpouseAge = "25"
			if err := ValidateFinancialPlannerRequest(req); err == nil || !strings.Contains(err.Error(), "spouse_age must be 0 when marital_status is single") {
				t.Fatalf("expected %s single flow to reject spouse_age for single user, got: %v", tt.name, err)
			}

			req.SpouseAge = "0"
			req.ChildrenCount = "1"
			if err := ValidateFinancialPlannerRequest(req); err == nil || !strings.Contains(err.Error(), "children_count must be 0 when marital_status is single") {
				t.Fatalf("expected %s single flow to reject children_count greater than 0, got: %v", tt.name, err)
			}
		})
	}
}

func TestNonInstantMarriedKeepsExistingFamilyValidationAndAllowsZeroChildren(t *testing.T) {
	flows := []struct {
		name  string
		goals []SelectedGoal
	}{
		{"short", []SelectedGoal{{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3}}},
		{"long", []SelectedGoal{{Key: "retirement", Label: "Retirement Planning", Type: "long", Amount: 20000000, Years: 25}}},
		{"explorer", []SelectedGoal{
			{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
			{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
		}},
	}

	for _, tt := range flows {
		t.Run(tt.name, func(t *testing.T) {
			req := validBaseRequestForProfileValidation()
			req.MaritalStatus = "married"
			req.SpouseAge = "30"
			req.SpouseIncome = "0"
			req.ChildrenCount = "0"
			req.Dependents = "0"
			req.SelectedGoals = tt.goals

			if err := ValidateFinancialPlannerRequest(req); err != nil {
				t.Fatalf("expected %s married flow with children_count=0 to pass, got: %v", tt.name, err)
			}

			req.SpouseAge = ""
			if err := ValidateFinancialPlannerRequest(req); err == nil || !strings.Contains(err.Error(), "spouse_age is required when marital_status is married") {
				t.Fatalf("expected %s married flow to require spouse_age, got: %v", tt.name, err)
			}

			req.SpouseAge = "30"
			req.ChildrenCount = ""
			if err := ValidateFinancialPlannerRequest(req); err == nil ||
				!strings.Contains(err.Error(), "children_count is required") {
				t.Fatalf(
					"expected %s married flow to require children_count, got: %v",
					tt.name,
					err,
				)
			}

			req.ChildrenCount = "2"

			if err := ValidateFinancialPlannerRequest(req); err != nil {
				t.Fatalf(
					"expected %s married flow with spouse_age and children_count to pass, got: %v",
					tt.name,
					err,
				)
			}
		})
	}
}
func TestNonInstantRejectsChildDetailsWhenChildrenCountIsZero(t *testing.T) {
	tests := []struct {
		name  string
		goals []SelectedGoal
	}{
		{"short", []SelectedGoal{{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3}}},
		{"long", []SelectedGoal{{Key: "retirement", Label: "Retirement Planning", Type: "long", Amount: 20000000, Years: 25}}},
		{"explorer", []SelectedGoal{
			{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
			{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validBaseRequestForProfileValidation()
			req.MaritalStatus = "married"
			req.SpouseAge = "30"
			req.SpouseIncome = "0"
			req.ChildrenCount = "0"
			req.Dependents = "0"
			req.SelectedGoals = tt.goals
			req.Children = []ChildDetail{{Index: 1, Name: "Aarav", Age: 5}}
			err := ValidateFinancialPlannerRequest(req)
			if err == nil || !strings.Contains(err.Error(), "child details cannot be provided when children_count is 0") {
				t.Fatalf("expected %s flow to reject child details when children_count=0, got: %v", tt.name, err)
			}
		})
	}
}
func TestMarriedRequiresChildDetailsToMatchChildrenCount(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MaritalStatus = "married"
	req.SpouseAge = "30"
	req.SpouseIncome = "0"
	req.ChildrenCount = "2"
	req.SelectedGoals = []SelectedGoal{{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3}}
	req.Children = nil
	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "children_count is 2 but 0 child detail") {
		t.Fatalf("expected exact child detail count validation, got: %v", err)
	}
}
func TestChildrenCountRejectsExtraChildDetails(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MaritalStatus = "married"
	req.SpouseAge = "30"
	req.SpouseIncome = "0"
	req.ChildrenCount = "1"
	req.SelectedGoals = []SelectedGoal{
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
	}
	req.Children = []ChildDetail{
		{Index: 1, Name: "Aarav", Age: 5},
		{Index: 2, Name: "Anaya", Age: 3},
	}
	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "child_2 details cannot be provided when children_count is 1") {
		t.Fatalf("expected extra child details validation, got: %v", err)
	}
}

func TestCreateYourOwnSingleInstantUsesNormalFamilyAndInsuranceValidation(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.GoalSelectionMode = "create_your_own"
	req.MaritalStatus = "married"
	req.SpouseAge = "30"
	req.SpouseIncome = "50000"
	req.ChildrenCount = "1"
	req.Dependents = "1"
	req.ExistingLifeTerm = "yes"
	req.ExistingLifeTermCover = "10000000"
	req.SelectedGoals = []SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
	}

	if got := effectiveGoalFlow(req); got != "explorer" {
		t.Fatalf("effectiveGoalFlow() = %q, want explorer for create_your_own", got)
	}
	if err := ValidateFinancialPlannerRequest(req); err != nil {
		t.Fatalf("expected create-your-own single instant goal to allow family and insurance details, got: %v", err)
	}
}

func TestDirectInstantStillRejectsFamilyAndInsuranceDetails(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.GoalSelectionMode = "instant"
	req.MaritalStatus = "married"
	req.SpouseAge = "30"
	req.ChildrenCount = "0"
	req.Dependents = "0"
	req.SelectedGoals = []SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
	}

	if err := ValidateFinancialPlannerRequest(req); err == nil || !strings.Contains(err.Error(), "Family details are not applicable for instant goals") {
		t.Fatalf("expected direct instant goal to keep family restriction, got: %v", err)
	}
}

func TestCreateYourOwnMixedGoalsRemainsExplorer(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.GoalSelectionMode = "create_your_own"
	req.MaritalStatus = "single"
	req.ChildrenCount = "0"
	req.Dependents = "0"
	req.SelectedGoals = []SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
	}

	if err := ValidateFinancialPlannerRequest(req); err != nil {
		t.Fatalf("expected create-your-own mixed goals to use normal Explorer validation, got: %v", err)
	}
}
func TestExplicitInstantModeTakesPrecedenceOverMixedChildGoalTypes(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.GoalSelectionMode = "instant"
	req.MaritalStatus = "married"
	req.SpouseAge = "21"
	req.ChildrenCount = "2"
	req.Dependents = "2"
	req.SelectedGoals = []SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
		{Key: "retirement", Label: "Retirement Planning", Type: "long", Amount: 20000000, Years: 25},
	}
	if got := effectiveGoalFlow(req); got != "instant" {
		t.Fatalf("effectiveGoalFlow() = %q, want instant when goal_selection_mode is explicit", got)
	}
	if err := ValidateFinancialPlannerRequest(req); err == nil || !strings.Contains(err.Error(), "Family details are not applicable for instant goals") {
		t.Fatalf("expected explicit instant mode to apply instant family restriction, got: %v", err)
	}
}
func TestExplicitCreateYourOwnStillOverridesAllInstantChildren(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.GoalSelectionMode = "create_your_own"
	req.MaritalStatus = "married"
	req.SpouseAge = "30"
	req.SpouseIncome = "50000"
	req.ChildrenCount = "1"
	req.Dependents = "1"
	req.ExistingLifeTerm = "yes"
	req.ExistingLifeTermCover = "10000000"
	req.SelectedGoals = []SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
		{Key: "bike", Label: "Bike", Type: "instant", Amount: 150000, Months: 12},
	}
	if got := effectiveGoalFlow(req); got != "explorer" {
		t.Fatalf("effectiveGoalFlow() = %q, want explorer for create_your_own", got)
	}
	if err := ValidateFinancialPlannerRequest(req); err != nil {
		t.Fatalf("expected create-your-own all-instant child goals to allow normal family/insurance validation, got: %v", err)
	}
}

func TestGoalSelectionModeRejectsOtherGoalTypes(t *testing.T) {
	tests := []struct {
		name string
		mode string
		goal SelectedGoal
	}{
		{name: "short rejects instant", mode: "short", goal: SelectedGoal{Type: "instant"}},
		{name: "short rejects long", mode: "short", goal: SelectedGoal{Type: "long"}},
		{name: "instant rejects short", mode: "instant", goal: SelectedGoal{Type: "short"}},
		{name: "long rejects short", mode: "long", goal: SelectedGoal{Type: "short"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := FinancialPlannerRequest{GoalSelectionMode: tt.mode, SelectedGoals: []SelectedGoal{tt.goal}}
			if err := validateGoalSelectionModeAndTypes(req); err == nil {
				t.Fatalf("expected mode/type mismatch to be rejected")
			}
		})
	}
}

func TestCreateYourOwnAllowsMixedGoalTypes(t *testing.T) {
	req := FinancialPlannerRequest{
		GoalSelectionMode: "create_your_own",
		SelectedGoals:     []SelectedGoal{{Type: "instant"}, {Type: "short"}, {Type: "long"}},
	}
	if err := validateGoalSelectionModeAndTypes(req); err != nil {
		t.Fatalf("create_your_own should allow mixed goal types: %v", err)
	}
}

func TestGoalSelectionModeIsRequired(t *testing.T) {
	req := FinancialPlannerRequest{SelectedGoals: []SelectedGoal{{Type: "short"}}}
	if err := validateGoalSelectionModeAndTypes(req); err == nil {
		t.Fatalf("expected missing goal_selection_mode to be rejected")
	}
}
