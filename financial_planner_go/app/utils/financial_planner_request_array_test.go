package utils

import (
	"encoding/json"
	"testing"
)

func TestFinancialPlannerRequestAcceptsSelectedGoalsArrayWithEmptyMonths(t *testing.T) {
	payload := []byte(`{
		"full_name":"Sample User",
		"contact":"9999999999",
		"email":"sample.user@example.com",
		"dob":"01/01/1990",
		"selected_goals_json":[{
			"key":"home_purchase",
			"label":"Home Purchase / Home Upgrade",
			"type":"short",
			"categoryTitle":"Short Term Goals",
			"timeline":"Within 5 years",
			"amount":2500000,
			"years":5,
			"months":"",
			"customName":""
		}]
	}`)

	var request FinancialPlannerRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("expected array payload to decode, got error: %v", err)
	}
	if len(request.SelectedGoals) != 1 {
		t.Fatalf("expected one selected goal, got %d", len(request.SelectedGoals))
	}
	goal := request.SelectedGoals[0]
	if goal.Years != 5 {
		t.Fatalf("expected years=5, got %d", goal.Years)
	}
	if goal.Months != 0 {
		t.Fatalf("expected empty months to normalize to 0, got %d", goal.Months)
	}
}
