package utils

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRejectsDuplicateChildAliasFields(t *testing.T) {
	payload := `{"children_count":1,"child_name":"Aarav","child_dob":"05/08/2023","child_name":"Aav","child_dob":"05/07/2023","selected_goals_json":[{"key":"x","label":"x","type":"short","amount":1}]}`
	var req FinancialPlannerRequest
	err := json.Unmarshal([]byte(payload), &req)
	if err == nil || !strings.Contains(err.Error(), "duplicate JSON field") {
		t.Fatalf("expected duplicate JSON field validation error, got: %v", err)
	}
}

func TestChildrenCountMustMatchProvidedDetails(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.MaritalStatus = "married"
	req.SpouseDOB = "01/01/1995"
	req.SpouseAge = "31"
	req.SpouseIncome = "0"
	req.SelectedGoals = []SelectedGoal{{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3}}
	req.ChildrenCount = "2"
	req.Children = []ChildDetail{{Index: 1, Name: "Aarav", DOB: "05/08/2023"}}
	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "children_count is 2 but 1 child detail") {
		t.Fatalf("expected child count mismatch error, got: %v", err)
	}
}
