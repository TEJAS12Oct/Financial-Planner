package utils

import (
	"strings"
	"testing"
)

func TestValidateSelectedGoalsAcceptsSupportedGoals(t *testing.T) {
	goals := []SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12, Timeline: "12 Months"},
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3, Timeline: "3 Years"},
		{Key: "retirement", Label: "Retirement Planning", Type: "long", Amount: 20000000, Years: 25, Timeline: "25 Years"},
	}

	if err := validateSelectedGoals(goals); err != nil {
		t.Fatalf("expected valid goals, got error: %v", err)
	}
}

func TestValidateSelectedGoalsRejectsUnknownKey(t *testing.T) {
	goals := []SelectedGoal{
		{Key: "mob", Label: "mb", Type: "instant", Amount: 90000, Months: 12, Timeline: "12 Months"},
	}

	err := validateSelectedGoals(goals)
	if err == nil || !strings.Contains(err.Error(), "key 'mob' is invalid") {
		t.Fatalf("expected invalid key error, got: %v", err)
	}
}

func TestValidateSelectedGoalsRejectsWrongLabel(t *testing.T) {
	goals := []SelectedGoal{
		{Key: "iphone", Label: "Mobile", Type: "instant", Amount: 90000, Months: 12, Timeline: "12 Months"},
	}

	err := validateSelectedGoals(goals)
	if err == nil || !strings.Contains(err.Error(), "expected label is 'Phone'") {
		t.Fatalf("expected invalid label error, got: %v", err)
	}
}

func TestValidateSelectedGoalsRejectsWrongType(t *testing.T) {
	goals := []SelectedGoal{
		{Key: "retirement", Label: "Retirement Planning", Type: "instant", Amount: 20000000, Months: 12, Timeline: "12 Months"},
	}

	err := validateSelectedGoals(goals)
	if err == nil || !strings.Contains(err.Error(), "invalid for type 'instant'") {
		t.Fatalf("expected type-specific key error, got: %v", err)
	}
}

func TestValidateSelectedGoalsRejectsInstantMonthOutsideRange(t *testing.T) {
	goals := []SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 5, Timeline: "5 Months"},
	}

	err := validateSelectedGoals(goals)
	if err == nil || !strings.Contains(err.Error(), "months must be between 6 and 12") {
		t.Fatalf("expected instant month range error, got: %v", err)
	}
}

func TestValidateSelectedGoalsRejectsShortYearOutsideRange(t *testing.T) {
	goals := []SelectedGoal{
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 6, Timeline: "6 Years"},
	}

	err := validateSelectedGoals(goals)
	if err == nil || !strings.Contains(err.Error(), "years must be between 1 and 5") {
		t.Fatalf("expected short year range error, got: %v", err)
	}
}

func TestValidateSelectedGoalsRejectsLongYearOutsideRange(t *testing.T) {
	goals := []SelectedGoal{
		{Key: "wealth", Label: "Wealth Creation", Type: "long", Amount: 10000000, Years: 30, Timeline: "30 Years"},
	}

	err := validateSelectedGoals(goals)
	if err == nil || !strings.Contains(err.Error(), "years must be between 20 and 25") {
		t.Fatalf("expected long year range error, got: %v", err)
	}
}

func TestValidateSelectedGoalsRequiresCustomNameForOthers(t *testing.T) {
	goals := []SelectedGoal{
		{Key: "others", Label: "Others", Type: "instant", Amount: 100000, Months: 12, Timeline: "12 Months"},
	}

	err := validateSelectedGoals(goals)
	if err == nil || !strings.Contains(err.Error(), "customName is required") {
		t.Fatalf("expected customName error, got: %v", err)
	}
}

func TestValidateSelectedGoalsAcceptsTimelineBoundaries(t *testing.T) {
	tests := []struct {
		name string
		goal SelectedGoal
	}{
		{"instant min", SelectedGoal{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 6, Timeline: "6 Months"}},
		{"instant max", SelectedGoal{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12, Timeline: "12 Months"}},
		{"short min", SelectedGoal{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 1, Timeline: "1 Years"}},
		{"short max", SelectedGoal{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 5, Timeline: "5 Years"}},
		{"long min", SelectedGoal{Key: "wealth", Label: "Wealth Creation", Type: "long", Amount: 10000000, Years: 20, Timeline: "20 Years"}},
		{"long max", SelectedGoal{Key: "wealth", Label: "Wealth Creation", Type: "long", Amount: 10000000, Years: 25, Timeline: "25 Years"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateSelectedGoals([]SelectedGoal{tc.goal}); err != nil {
				t.Fatalf("expected boundary value to be valid, got: %v", err)
			}
		})
	}
}

func TestValidateSelectedGoalsRejectsTimelineCrossLimits(t *testing.T) {
	tests := []struct {
		name        string
		goal        SelectedGoal
		wantMessage string
	}{
		{"instant below 6", SelectedGoal{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 5, Timeline: "5 Months"}, "months must be between 6 and 12"},
		{"instant above 12", SelectedGoal{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 13, Timeline: "13 Months"}, "months must be between 6 and 12"},
		{"instant with years", SelectedGoal{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12, Timeline: "12 Months", Years: 1}, "years must not be provided"},
		{"short below 1", SelectedGoal{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 0, Timeline: "0 Years"}, "years must be between 1 and 5"},
		{"short above 5", SelectedGoal{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 6, Timeline: "6 Years"}, "years must be between 1 and 5"},
		{"short with months", SelectedGoal{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3, Timeline: "3 Years", Months: 1}, "months must not be provided"},
		{"long below 20", SelectedGoal{Key: "wealth", Label: "Wealth Creation", Type: "long", Amount: 10000000, Years: 19, Timeline: "19 Years"}, "years must be between 20 and 25"},
		{"long above 25", SelectedGoal{Key: "wealth", Label: "Wealth Creation", Type: "long", Amount: 10000000, Years: 26, Timeline: "26 Years"}, "years must be between 20 and 25"},
		{"long with months", SelectedGoal{Key: "wealth", Label: "Wealth Creation", Type: "long", Amount: 10000000, Years: 20, Timeline: "20 Years", Months: 1}, "months must not be provided"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSelectedGoals([]SelectedGoal{tc.goal})
			if err == nil || !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("expected %q error, got: %v", tc.wantMessage, err)
			}
		})
	}
}

func TestValidateSelectedGoalsRejectsTimelineFieldMismatch(t *testing.T) {
	tests := []struct {
		name        string
		goal        SelectedGoal
		wantMessage string
	}{
		{"instant timeline over limit", SelectedGoal{Key: "bike", Label: "Bike", Type: "instant", Amount: 150000, Months: 12, Timeline: "13 Months"}, "timeline must be between 6 and 12 Months"},
		{"instant timeline mismatch", SelectedGoal{Key: "bike", Label: "Bike", Type: "instant", Amount: 150000, Months: 12, Timeline: "11 Months"}, "does not match months=12"},
		{"instant wrong unit", SelectedGoal{Key: "bike", Label: "Bike", Type: "instant", Amount: 150000, Months: 12, Timeline: "1 Year"}, "timeline must use Months"},
		{"short timeline over limit", SelectedGoal{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 5, Timeline: "6 Years"}, "timeline must be between 1 and 5 Years"},
		{"short timeline mismatch", SelectedGoal{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3, Timeline: "4 Years"}, "does not match years=3"},
		{"long timeline below limit", SelectedGoal{Key: "retirement", Label: "Retirement Planning", Type: "long", Amount: 20000000, Years: 20, Timeline: "19 Years"}, "timeline must be between 20 and 25 Years"},
		{"missing timeline", SelectedGoal{Key: "bike", Label: "Bike", Type: "instant", Amount: 150000, Months: 12}, "timeline is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSelectedGoals([]SelectedGoal{tc.goal})
			if err == nil || !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("expected %q error, got: %v", tc.wantMessage, err)
			}
		})
	}
}
