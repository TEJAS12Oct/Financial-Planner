package utils

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChildNameAndDOBMapToFirstChild(t *testing.T) {
	payload := `{"selected_goals_json":[{"key":"x","label":"x","type":"short","amount":1}],"child_name":"Aarav","child_dob":"05/08/2023"}`
	var req FinancialPlannerRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		t.Fatal(err)
	}
	if req.ChildName != "Aarav" || req.ChildDOB != "05/08/2023" {
		t.Fatalf("aliases not decoded: %+v", req)
	}
	if len(req.Children) != 1 || req.Children[0].Index != 1 {
		t.Fatalf("alias not mapped: %+v", req.Children)
	}
}

func TestRejectSecondChildWhenCountIsOne(t *testing.T) {
	req := validBaseRequestForProfileValidation()
	req.ChildrenCount = "1"
	req.Children = []ChildDetail{{Index: 1, Name: "A", DOB: "05/08/2023"}, {Index: 2, Name: "B", DOB: "10/02/2024"}}
	err := ValidateFinancialPlannerRequest(req)
	if err == nil || !strings.Contains(err.Error(), "child_2 details cannot be provided when children_count is 1") {
		t.Fatalf("expected count mismatch error, got %v", err)
	}
}
