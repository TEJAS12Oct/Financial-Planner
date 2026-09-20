package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FinancialPlannerRequest accepts the Boltpe payload documented by the legacy-compatible API.
// selected_goals_json may be either a stringified JSON array (documented format)
// or a normal JSON array (convenience format).
type FinancialPlannerRequest struct {
	GoalSelectionMode string `json:"goal_selection_mode,omitempty"`
	FullName          string `json:"full_name" binding:"required"`
	Contact           string `json:"contact" binding:"required"`
	Email             string `json:"email" binding:"required,email"`
	DOB               string `json:"dob" binding:"required"`
	City              string `json:"city"`
	MaritalStatus     string `json:"marital_status"`
	Dependents        string `json:"dependents"`
	ChildrenCount     string `json:"children_count"`
	MonthlyIncome     string `json:"monthly_income"`
	MonthlyExpenses   string `json:"monthly_expenses"`
	MonthlySavings    string `json:"monthly_savings"`
	SpouseIncome      string `json:"spouse_income"`
	SpouseDOB         string `json:"spouse_dob"`
	SpouseAge         string `json:"-"`
	// Legacy single-child aliases. These map to child_1_name and child_1_dob.
	ChildName            string `json:"child_name,omitempty"`
	ChildDOB             string `json:"child_dob,omitempty"`
	RiskAppetite         string `json:"risk_appetite"`
	InvestmentExperience string `json:"investment_experience"`

	ExistingLifeTerm      any    `json:"existing_life_term,omitempty"`
	ExistingHealth        any    `json:"existing_health,omitempty"`
	ExistingCritical      any    `json:"existing_critical,omitempty"`
	ExistingPersonal      any    `json:"existing_personal,omitempty"`
	ExistingLifeTermCover string `json:"existing_life_term_cover"`
	ExistingHealthCover   string `json:"existing_health_cover"`
	ExistingCriticalCover string `json:"existing_critical_cover"`
	ExistingPersonalCover string `json:"existing_personal_cover"`

	Education1Inflation      string `json:"education_1_inflation,omitempty"`
	Education1LumpsumReturn  string `json:"education_1_lumpsum_return,omitempty"`
	Education1SIPReturn      string `json:"education_1_sip_return,omitempty"`
	Education1TargetAge      string `json:"education_1_target_age,omitempty"`
	Marriage1Inflation       string `json:"marriage_1_inflation,omitempty"`
	Marriage1LumpsumReturn   string `json:"marriage_1_lumpsum_return,omitempty"`
	Marriage1SIPReturn       string `json:"marriage_1_sip_return,omitempty"`
	Marriage1TargetAge       string `json:"marriage_1_target_age,omitempty"`
	RetirementInflation      string `json:"retirement_inflation,omitempty"`
	RetirementPreReturn      string `json:"retirement_pre_return,omitempty"`
	RetirementPostReturn     string `json:"retirement_post_return,omitempty"`
	WealthInflation          string `json:"wealth_inflation,omitempty"`
	WealthLumpsumReturn      string `json:"wealth_lumpsum_return,omitempty"`
	WealthSIPReturn          string `json:"wealth_sip_return,omitempty"`
	GenericGoalInflation     string `json:"generic_goal_inflation,omitempty"`
	GenericGoalLumpsumReturn string `json:"generic_goal_lumpsum_return,omitempty"`
	GenericGoalSIPReturn     string `json:"generic_goal_sip_return,omitempty"`

	SelectedGoals []SelectedGoal `json:"selected_goals_json" binding:"required,min=1"`
	Children      []ChildDetail  `json:"-"`
}
type ChildDetail struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	DOB   string `json:"dob"`
	Age   int    `json:"age"`
}

type SelectedGoal struct {
	Key           string  `json:"key" binding:"required"`
	Label         string  `json:"label" binding:"required"`
	Type          string  `json:"type" binding:"required"`
	CategoryTitle string  `json:"categoryTitle,omitempty"`
	Timeline      string  `json:"timeline,omitempty"`
	Amount        float64 `json:"amount" binding:"required"`
	Years         int     `json:"years,omitempty"`
	Months        int     `json:"months,omitempty"`
	CustomName    string  `json:"customName,omitempty"`
	Retirement    bool    `json:"retirement,omitempty"`
	AnnualBudget  bool    `json:"annualBudget,omitempty"`
}

// UnmarshalJSON accepts years and months as JSON numbers, numeric strings,
// empty strings, null, or omitted values. Empty/null values are normalized to 0.
func (g *SelectedGoal) UnmarshalJSON(data []byte) error {
	type selectedGoalAlias struct {
		Key           string          `json:"key"`
		Label         string          `json:"label"`
		Type          string          `json:"type"`
		CategoryTitle string          `json:"categoryTitle"`
		Timeline      string          `json:"timeline"`
		Amount        float64         `json:"amount"`
		Years         json.RawMessage `json:"years"`
		Months        json.RawMessage `json:"months"`
		CustomName    string          `json:"customName"`
		Retirement    bool            `json:"retirement"`
		AnnualBudget  bool            `json:"annualBudget"`
	}

	var decoded selectedGoalAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	years, err := parseFlexibleInt(decoded.Years, "years")
	if err != nil {
		return err
	}
	months, err := parseFlexibleInt(decoded.Months, "months")
	if err != nil {
		return err
	}

	*g = SelectedGoal{
		Key:           decoded.Key,
		Label:         decoded.Label,
		Type:          decoded.Type,
		CategoryTitle: decoded.CategoryTitle,
		Timeline:      decoded.Timeline,
		Amount:        decoded.Amount,
		Years:         years,
		Months:        months,
		CustomName:    decoded.CustomName,
		Retirement:    decoded.Retirement,
		AnnualBudget:  decoded.AnnualBudget,
	}
	return nil
}

func parseFlexibleInt(raw json.RawMessage, field string) (int, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0, nil
	}

	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, nil
	}

	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, fmt.Errorf("%s must be a whole number or an empty string", field)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("%s must be a whole number or an empty string", field)
	}
	return parsed, nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder, "request"); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("invalid JSON payload")
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, path string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}

	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}

	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("invalid JSON object key")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON field %q is not allowed", key)
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder, path+"."+key); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		index := 0
		for decoder.More() {
			if err := scanJSONValue(decoder, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
			index++
		}
		_, err = decoder.Token()
		return err
	default:
		return fmt.Errorf("invalid JSON delimiter")
	}
}

func (r *FinancialPlannerRequest) UnmarshalJSON(data []byte) error {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}

	type alias FinancialPlannerRequest
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	selectedRaw, ok := fields["selected_goals_json"]
	if !ok || len(selectedRaw) == 0 || bytes.Equal(selectedRaw, []byte("null")) {
		return fmt.Errorf("selected_goals_json is required")
	}
	delete(fields, "selected_goals_json")

	// The legacy-compatible API accepts numeric values as either JSON numbers or strings.
	stringOrNumberFields := []string{
		"dependents", "children_count", "monthly_income", "monthly_expenses", "monthly_savings",
		"spouse_income", "existing_life_term_cover", "existing_health_cover",
		"existing_critical_cover", "existing_personal_cover", "education_1_inflation",
		"education_1_lumpsum_return", "education_1_sip_return", "education_1_target_age",
		"marriage_1_inflation", "marriage_1_lumpsum_return", "marriage_1_sip_return",
		"marriage_1_target_age", "retirement_inflation", "retirement_pre_return",
		"retirement_post_return", "wealth_inflation", "wealth_lumpsum_return",
		"wealth_sip_return", "generic_goal_inflation", "generic_goal_lumpsum_return",
		"generic_goal_sip_return",
	}
	for _, key := range stringOrNumberFields {
		raw, exists := fields[key]
		if !exists || len(raw) == 0 || bytes.Equal(raw, []byte("null")) || raw[0] == '"' {
			continue
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("%s has invalid value", key)
		}
		text := fmt.Sprint(value)
		encoded, _ := json.Marshal(text)
		fields[key] = encoded
	}

	normalized, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	var decoded alias
	if err := json.Unmarshal(normalized, &decoded); err != nil {
		return err
	}
	*r = FinancialPlannerRequest(decoded)

	children, err := parseDynamicChildren(fields)
	if err != nil {
		return err
	}
	r.Children = children
	// Keep the existing internal age-based calculation/reporting logic unchanged.
	// Ages are derived from the submitted DD/MM/YYYY birthdates and are never accepted as API input.
	now := time.Now()
	if spouseDOB, err := time.Parse("02/01/2006", strings.TrimSpace(r.SpouseDOB)); err == nil {
		r.SpouseAge = strconv.Itoa(calculateAge(spouseDOB, now))
	}
	for i := range r.Children {
		if childDOB, err := time.Parse("02/01/2006", strings.TrimSpace(r.Children[i].DOB)); err == nil {
			r.Children[i].Age = calculateAge(childDOB, now)
		}
	}
	var encoded string
	if err := json.Unmarshal(selectedRaw, &encoded); err == nil {
		if err := json.Unmarshal([]byte(encoded), &r.SelectedGoals); err != nil {
			return fmt.Errorf("selected_goals_json must be a valid stringified JSON array: %w", err)
		}
		return nil
	}
	if err := json.Unmarshal(selectedRaw, &r.SelectedGoals); err != nil {
		return fmt.Errorf("selected_goals_json must be a JSON array or stringified JSON array: %w", err)
	}
	return nil
}

var childFieldPattern = regexp.MustCompile(`^child_([1-9][0-9]*)_(name|dob)$`)

func parseDynamicChildren(fields map[string]json.RawMessage) ([]ChildDetail, error) {
	children := map[int]*ChildDetail{}

	// Backward-compatible aliases: child_name/child_dob represent child_1.
	aliasFields := map[string]string{"child_name": "name", "child_dob": "dob"}
	for key, part := range aliasFields {
		raw, exists := fields[key]
		if !exists {
			continue
		}
		child := children[1]
		if child == nil {
			child = &ChildDetail{Index: 1}
			children[1] = child
		}
		if part == "name" {
			if err := json.Unmarshal(raw, &child.Name); err != nil {
				return nil, fmt.Errorf("child_name must be a string")
			}
			child.Name = strings.TrimSpace(child.Name)
		} else {
			if err := json.Unmarshal(raw, &child.DOB); err != nil {
				return nil, fmt.Errorf("child_dob must be a string in DD/MM/YYYY format")
			}
			child.DOB = strings.TrimSpace(child.DOB)
		}
	}

	for key, raw := range fields {
		matches := childFieldPattern.FindStringSubmatch(key)
		if len(matches) != 3 {
			continue
		}
		index, _ := strconv.Atoi(matches[1])
		child := children[index]
		if child == nil {
			child = &ChildDetail{Index: index}
			children[index] = child
		}
		switch matches[2] {
		case "name":
			if index == 1 && child.Name != "" && key == "child_1_name" {
				return nil, fmt.Errorf("use either child_name or child_1_name, not both")
			}
			if err := json.Unmarshal(raw, &child.Name); err != nil {
				return nil, fmt.Errorf("%s must be a string", key)
			}
			child.Name = strings.TrimSpace(child.Name)
		case "dob":
			if index == 1 && child.DOB != "" && key == "child_1_dob" {
				return nil, fmt.Errorf("use either child_dob or child_1_dob, not both")
			}
			if err := json.Unmarshal(raw, &child.DOB); err != nil {
				return nil, fmt.Errorf("%s must be a string in DD/MM/YYYY format", key)
			}
			child.DOB = strings.TrimSpace(child.DOB)
		}
	}
	indices := make([]int, 0, len(children))
	for index := range children {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	result := make([]ChildDetail, 0, len(indices))
	for _, index := range indices {
		result = append(result, *children[index])
	}
	return result, nil
}
