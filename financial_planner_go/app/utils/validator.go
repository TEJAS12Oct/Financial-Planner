package utils

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var phoneRegex = regexp.MustCompile(`^[6-9][0-9]{9}$`)
var nameRegex = regexp.MustCompile(`^[A-Za-z][A-Za-z .'-]*$`)
var goalMaster map[string]GoalDefinition

func NormalizeFinancialPlannerRequest(req *FinancialPlannerRequest) {
	req.GoalSelectionMode = normalizeGoalSelectionMode(req.GoalSelectionMode)
	req.FullName = strings.TrimSpace(req.FullName)
	req.Contact = strings.TrimSpace(req.Contact)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.DOB = strings.TrimSpace(req.DOB)
	req.SpouseDOB = strings.TrimSpace(req.SpouseDOB)
	req.ChildName = strings.TrimSpace(req.ChildName)
	req.ChildDOB = strings.TrimSpace(req.ChildDOB)

	// Support requests/tests built directly with child_name and child_dob.
	if req.ChildName != "" || req.ChildDOB != "" {
		found := false
		for i := range req.Children {
			if req.Children[i].Index == 1 {
				if req.Children[i].Name == "" {
					req.Children[i].Name = req.ChildName
				}
				if req.Children[i].DOB == "" {
					req.Children[i].DOB = req.ChildDOB
				}
				found = true
				break
			}
		}
		if !found {
			req.Children = append(req.Children, ChildDetail{Index: 1, Name: req.ChildName, DOB: req.ChildDOB})
		}
	}
	for i := range req.Children {
		req.Children[i].Name = strings.TrimSpace(req.Children[i].Name)
		req.Children[i].DOB = strings.TrimSpace(req.Children[i].DOB)
	}
	req.City = strings.TrimSpace(req.City)
	req.MaritalStatus = strings.ToLower(strings.TrimSpace(req.MaritalStatus))
	req.RiskAppetite = strings.ToLower(strings.TrimSpace(req.RiskAppetite))
	switch strings.ToLower(strings.TrimSpace(req.InvestmentExperience)) {
	case "beginner":
		req.InvestmentExperience = "Beginner"
	case "intermediate":
		req.InvestmentExperience = "Intermediate"
	case "advance", "advanced":
		req.InvestmentExperience = "Advance"
	default:
		req.InvestmentExperience = strings.TrimSpace(req.InvestmentExperience)
	}

	for i := range req.SelectedGoals {
		req.SelectedGoals[i].Key = strings.TrimSpace(req.SelectedGoals[i].Key)
		req.SelectedGoals[i].Label = strings.TrimSpace(req.SelectedGoals[i].Label)
		req.SelectedGoals[i].Type = strings.ToLower(strings.TrimSpace(req.SelectedGoals[i].Type))
	}
	applyGoalBasedFamilyDefaults(req)
}

func normalizeGoalSelectionMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	mode = strings.ReplaceAll(mode, "-", "_")
	mode = strings.ReplaceAll(mode, " ", "_")
	switch mode {
	case "create_your_own", "create_your_own_goals", "custom", "explorer":
		return "create_your_own"
	case "instant", "short", "long":
		return mode
	default:
		return mode
	}
}

func validateGoalSelectionModeAndTypes(req FinancialPlannerRequest) error {
	mode := normalizeGoalSelectionMode(req.GoalSelectionMode)
	if mode == "" {
		return errors.New("goal_selection_mode is required and must be one of: instant, short, long, create_your_own")
	}
	if mode != "instant" && mode != "short" && mode != "long" && mode != "create_your_own" {
		return fmt.Errorf("goal_selection_mode '%s' is invalid; supported values are: instant, short, long, create_your_own", req.GoalSelectionMode)
	}

	// create_your_own intentionally allows any combination of instant, short and long goals.
	if mode == "create_your_own" {
		return nil
	}

	// Direct top-level modes are exclusive: every selected goal must match the mode.
	for i, goal := range req.SelectedGoals {
		goalType := strings.ToLower(strings.TrimSpace(goal.Type))
		if goalType != mode {
			return fmt.Errorf(
				"selected_goals_json[%d].type '%s' is not allowed when goal_selection_mode is '%s'; only '%s' goals are allowed. Use goal_selection_mode 'create_your_own' to select multiple goal types",
				i, goal.Type, mode, mode,
			)
		}
	}
	return nil
}

func effectiveGoalFlow(req FinancialPlannerRequest) string {
	// goal_selection_mode represents the TOP-LEVEL selection made by the user.
	// When it is explicitly provided, it must take precedence over the child
	// goal types inside selected_goals_json.
	//
	// This is important because Create Your Own may contain instant/short/long
	// child goals, while a direct Instant selection must always keep the strict
	// Instant family and insurance restrictions.
	switch normalizeGoalSelectionMode(req.GoalSelectionMode) {
	case "create_your_own":
		return "explorer"
	case "instant":
		return "instant"
	case "short":
		return "short"
	case "long":
		return "long"
	default:
		// Backward compatibility for older clients that do not send
		// goal_selection_mode: infer the flow from selected goal types.
		return detectGoalFlow(req.SelectedGoals)
	}
}

func detectGoalFlow(goals []SelectedGoal) string {
	if len(goals) == 0 {
		return ""
	}

	types := make(map[string]struct{}, 3)
	for _, goal := range goals {
		goalType := strings.ToLower(strings.TrimSpace(goal.Type))
		if goalType == "instant" || goalType == "short" || goalType == "long" {
			types[goalType] = struct{}{}
		}
	}

	if len(types) > 1 {
		return "explorer"
	}
	for goalType := range types {
		return goalType
	}
	return ""
}

func applyGoalBasedFamilyDefaults(req *FinancialPlannerRequest) {
	if req == nil {
		return
	}

	switch effectiveGoalFlow(*req) {
	case "instant":
		// Instant flow does not use family/dependent or insurance details.
		// Keep the submitted values untouched so validation can reject accidental
		// family or insurance input instead of silently overwriting it.
	case "short", "long":
		// Short/Long flow requires marital status and children_count.
		// Dependents remain optional, so normalize an omitted value to zero.
		if strings.TrimSpace(req.Dependents) == "" {
			req.Dependents = "0"
		}
	case "explorer":
		// Mixed goal types are treated as Explorer. Marital status,
		// children_count and dependents are validated as required fields below.
	}
}
func nonZeroOrNonEmpty(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && value != "0"
}
func hasProvidedChildDetails(children []ChildDetail) bool {
	for _, child := range children {
		if strings.TrimSpace(child.Name) != "" || strings.TrimSpace(child.DOB) != "" {
			return true
		}
	}
	return false
}

func ValidateFinancialPlannerRequest(req FinancialPlannerRequest) error {
	// Also support unit tests or internal callers that construct the request
	// directly instead of going through JSON unmarshalling/normalization.
	if strings.TrimSpace(req.ChildName) != "" || strings.TrimSpace(req.ChildDOB) != "" {
		found := false
		for i := range req.Children {
			if req.Children[i].Index == 1 {
				if strings.TrimSpace(req.Children[i].Name) == "" {
					req.Children[i].Name = strings.TrimSpace(req.ChildName)
				}
				if strings.TrimSpace(req.Children[i].DOB) == "" {
					req.Children[i].DOB = strings.TrimSpace(req.ChildDOB)
				}
				found = true
				break
			}
		}
		if !found {
			req.Children = append(req.Children, ChildDetail{Index: 1, Name: strings.TrimSpace(req.ChildName), DOB: strings.TrimSpace(req.ChildDOB)})
		}
	}
	if len(req.SelectedGoals) == 0 {
		return errors.New("selected_goals_json must contain at least one goal")
	}
	if err := validateGoalSelectionModeAndTypes(req); err != nil {
		return err
	}
	flow := effectiveGoalFlow(req)
	applyGoalBasedFamilyDefaults(&req)

	if len(req.FullName) < 3 || len(req.FullName) > 100 {
		return errors.New("full_name must be between 3 and 100 characters")
	}
	if !nameRegex.MatchString(req.FullName) {
		return errors.New("full_name may contain only letters, spaces, apostrophes, dots, and hyphens")
	}

	if !phoneRegex.MatchString(req.Contact) {
		return errors.New("contact must be valid 10 digit Indian mobile number")
	}

	if _, err := mail.ParseAddress(req.Email); err != nil {
		return errors.New("email must be valid")
	}

	dob, err := time.Parse("02/01/2006", req.DOB)
	if err != nil {
		return errors.New("dob must be in DD/MM/YYYY format")
	}

	now := time.Now()
	if dob.Year() < 1900 || dob.After(now) {
		return errors.New("dob year must be between 1900 and current year and cannot be future date")
	}
	age := calculateAge(dob, now)
	if age < 18 || age > 100 {
		return errors.New("customer age calculated from dob must be between 18 and 100 years")
	}

	if strings.TrimSpace(req.City) == "" {
		return errors.New("city is required")
	}
	if len(req.City) < 2 || len(req.City) > 100 {
		return errors.New("city must be between 2 and 100 characters")
	}

	// Strict family validation applies ONLY when every selected goal is instant.
	// Explorer is intentionally excluded, even when it contains instant goals.
	if flow == "instant" {
		if strings.TrimSpace(req.MaritalStatus) != "" ||
			strings.TrimSpace(req.SpouseDOB) != "" ||
			nonZeroOrNonEmpty(req.SpouseIncome) ||
			nonZeroOrNonEmpty(req.ChildrenCount) ||
			nonZeroOrNonEmpty(req.Dependents) ||
			hasProvidedChildDetails(req.Children) {
			return errors.New("Family details are not applicable for instant goals. marital_status and child details must be empty; spouse_dob, spouse_income, children_count, and dependents must be empty or 0")
		}
	}
	if flow == "explorer" && strings.TrimSpace(req.Dependents) == "" {
		return errors.New("dependents is required when selected goals contain multiple goal types (Explorer flow)")
	}
	if (flow == "short" || flow == "long" || flow == "explorer") && strings.TrimSpace(req.ChildrenCount) == "" {
		return errors.New("children_count is required for short-term, long-term, and Explorer flows; 0 is allowed")
	}
	if strings.TrimSpace(req.Dependents) != "" {
		if _, err := strconv.Atoi(strings.TrimSpace(req.Dependents)); err != nil {
			return errors.New("dependents must be a whole number")
		}
	}

	if flow != "instant" && req.MaritalStatus != "single" &&
		req.MaritalStatus != "married" {
		return errors.New("marital_status must be single or married")
	}

	if req.RiskAppetite != "low" &&
		req.RiskAppetite != "medium" &&
		req.RiskAppetite != "high" {
		return errors.New("risk_appetite must be one of: low, medium, high")
	}

	if req.InvestmentExperience != "Beginner" &&
		req.InvestmentExperience != "Intermediate" &&
		req.InvestmentExperience != "Advance" {
		return errors.New("investment_experience must be one of: Beginner, Intermediate, Advance")
	}

	if strings.TrimSpace(req.MonthlyIncome) == "" ||
		strings.TrimSpace(req.MonthlyExpenses) == "" ||
		strings.TrimSpace(req.MonthlySavings) == "" {
		return errors.New("monthly_income, monthly_expenses, and monthly_savings are required")
	}

	if err := validateNonNegativeNumbers(req); err != nil {
		return err
	}

	incomeValue, _ := floatDefault(req.MonthlyIncome)
	if incomeValue < 10000 {
		return errors.New("monthly_income must be at least 10000")
	}

	// -----------------------------------------------------
	// Marital Status Validation
	// -----------------------------------------------------

	// Keep spouseAge available for child validation after the marital-status block.
	// It remains 0 for single and instant flows.
	spouseAge := 0
	if flow == "instant" {
		// Family fields have already been strictly validated above.
	} else if strings.EqualFold(req.MaritalStatus, "single") {

		if strings.TrimSpace(req.SpouseDOB) != "" || nonZeroOrNonEmpty(req.SpouseAge) {
			return errors.New("spouse_dob must be empty when marital_status is single")
		}

		if strings.TrimSpace(req.SpouseIncome) != "" &&
			strings.TrimSpace(req.SpouseIncome) != "0" {
			return errors.New("spouse_income must be 0 when marital_status is single")
		}

		if strings.TrimSpace(req.ChildrenCount) != "" &&
			strings.TrimSpace(req.ChildrenCount) != "0" {
			return errors.New("children_count must be 0 when marital_status is single")
		}

	} else {

		// Married validations

		if strings.TrimSpace(req.SpouseDOB) == "" && strings.TrimSpace(req.SpouseAge) == "" {
			return errors.New("spouse_dob is required when marital_status is married")
		}

		if strings.TrimSpace(req.SpouseIncome) == "" {
			return errors.New("spouse_income is required when marital_status is married")
		}

		if strings.TrimSpace(req.ChildrenCount) == "" {
			return errors.New("children_count is required when marital_status is married")
		}

		if strings.TrimSpace(req.SpouseDOB) != "" {
			spouseDOB, parseErr := time.Parse("02/01/2006", strings.TrimSpace(req.SpouseDOB))
			if parseErr != nil {
				return errors.New("spouse_dob must be in DD/MM/YYYY format")
			}
			if spouseDOB.Year() < 1900 || spouseDOB.After(now) {
				return errors.New("spouse_dob year must be between 1900 and current year and cannot be a future date")
			}
			spouseAge = calculateAge(spouseDOB, now)
		} else {
			// Internal compatibility for existing unit tests and persisted records.
			spouseAge, err = strconv.Atoi(strings.TrimSpace(req.SpouseAge))
			if err != nil {
				return errors.New("spouse_dob must be in DD/MM/YYYY format")
			}
		}
		if spouseAge < 18 || spouseAge > 100 {
			return errors.New("spouse age calculated from spouse_dob must be between 18 and 100 years")
		}

		childrenCount, err := strconv.Atoi(strings.TrimSpace(req.ChildrenCount))
		if err != nil {
			return errors.New("children_count must be a valid number")
		}

		// children_count is mandatory in non-instant flows, but zero is valid.
		// Child name and age fields are optional and are not validated.
		if childrenCount < 0 {
			return errors.New("children_count cannot be negative")
		}
	}

	// Child name/DOB fields are optional when children exist, but they must not
	// be supplied when children_count is zero. This applies to Short, Long,
	// and Explorer flows. Instant flow is handled by the stricter family rule above.
	if flow != "instant" && strings.TrimSpace(req.ChildrenCount) == "0" && hasProvidedChildDetails(req.Children) {
		return errors.New("child details cannot be provided when children_count is 0")
	}
	// Child details are optional, but when supplied their index must not exceed
	// children_count. Example: children_count=1 cannot contain child_2 details.
	if flow != "instant" && strings.TrimSpace(req.ChildrenCount) != "" {
		childrenCount, err := strconv.Atoi(strings.TrimSpace(req.ChildrenCount))
		if err != nil {
			return errors.New("children_count must be a valid number")
		}
		if err := validateChildren(

			req.Children,
			childrenCount,
			age,
			req.MaritalStatus,
			spouseAge,
		); err != nil {

			return err

		}
	}
	// -----------------------------------------------------

	if flow == "instant" {
		if err := validateInstantInsurance(req); err != nil {
			return err
		}
	} else if err := validateExistingInsurance(req); err != nil {
		return err
	}

	if err := validateAssumptionRanges(req); err != nil {
		return err
	}

	income, _ := floatDefault(req.MonthlyIncome)
	expenses, _ := floatDefault(req.MonthlyExpenses)
	savings, _ := floatDefault(req.MonthlySavings)

	if income > 0 && expenses > income {
		return errors.New("monthly_expenses cannot be greater than monthly_income")
	}

	if income > 0 && savings > income {
		return errors.New("monthly_savings cannot be greater than monthly_income")
	}
	if income > 0 && expenses+savings > income {
		return errors.New("monthly_expenses plus monthly_savings cannot be greater than monthly_income")
	}

	if err := validateSelectedGoals(req.SelectedGoals); err != nil {
		return err
	}

	return nil
}

func validateNonNegativeNumbers(req FinancialPlannerRequest) error {
	type numericField struct {
		name, value string
		whole       bool
	}
	fields := []numericField{
		{"dependents", req.Dependents, true},
		{"children_count", req.ChildrenCount, true},
		{"monthly_income", req.MonthlyIncome, false},
		{"monthly_expenses", req.MonthlyExpenses, false},
		{"monthly_savings", req.MonthlySavings, false},
		{"spouse_income", req.SpouseIncome, false},
		{"existing_life_term_cover", req.ExistingLifeTermCover, false},
		{"existing_health_cover", req.ExistingHealthCover, false},
		{"existing_critical_cover", req.ExistingCriticalCover, false},
		{"existing_personal_cover", req.ExistingPersonalCover, false},
		{"education_1_inflation", req.Education1Inflation, false},
		{"education_1_lumpsum_return", req.Education1LumpsumReturn, false},
		{"education_1_sip_return", req.Education1SIPReturn, false},
		{"education_1_target_age", req.Education1TargetAge, true},
		{"marriage_1_inflation", req.Marriage1Inflation, false},
		{"marriage_1_lumpsum_return", req.Marriage1LumpsumReturn, false},
		{"marriage_1_sip_return", req.Marriage1SIPReturn, false},
		{"marriage_1_target_age", req.Marriage1TargetAge, true},
		{"retirement_inflation", req.RetirementInflation, false},
		{"retirement_pre_return", req.RetirementPreReturn, false},
		{"retirement_post_return", req.RetirementPostReturn, false},
		{"wealth_inflation", req.WealthInflation, false},
		{"wealth_lumpsum_return", req.WealthLumpsumReturn, false},
		{"wealth_sip_return", req.WealthSIPReturn, false},
		{"generic_goal_inflation", req.GenericGoalInflation, false},
		{"generic_goal_lumpsum_return", req.GenericGoalLumpsumReturn, false},
		{"generic_goal_sip_return", req.GenericGoalSIPReturn, false},
	}
	for _, field := range fields {
		value := strings.TrimSpace(field.value)
		if value == "" {
			continue
		}
		if strings.Contains(value, ".") {
			parts := strings.SplitN(value, ".", 2)
			if len(parts) == 2 && len(strings.TrimRight(parts[1], "0")) > 2 {
				return fmt.Errorf("%s can have maximum 2 decimal places", field.name)
			}
		}
		number, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("%s must be numeric", field.name)
		}
		if number < 0 {
			return fmt.Errorf("%s cannot be negative", field.name)
		}
		if field.whole && number != float64(int64(number)) {
			return fmt.Errorf("%s must be a whole number", field.name)
		}
	}
	return nil
}

var goalTimelineRegex = regexp.MustCompile(`(?i)^\s*(?:within\s+)?([0-9]+)\s*(month|months|year|years)\s*$`)

func validateGoalTimeline(prefix string, goal SelectedGoal, definition GoalDefinition, goalType, key string) error {
	timeline := strings.TrimSpace(goal.Timeline)
	if timeline == "" {
		return fmt.Errorf("%s.timeline is required for %s goal key '%s'", prefix, goalType, key)
	}

	matches := goalTimelineRegex.FindStringSubmatch(timeline)
	if len(matches) != 3 {
		if goalType == "instant" {
			return fmt.Errorf("%s.timeline must be in month format, for example '12 Months', for instant goal key '%s'", prefix, key)
		}
		return fmt.Errorf("%s.timeline must be in year format, for example '%d Years', for %s goal key '%s'", prefix, definition.MinYears, goalType, key)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return fmt.Errorf("%s.timeline contains an invalid numeric value", prefix)
	}
	unit := strings.ToLower(matches[2])

	switch goalType {
	case "instant":
		if unit != "month" && unit != "months" {
			return fmt.Errorf("%s.timeline must use Months for instant goal key '%s'", prefix, key)
		}
		if value < definition.MinMonths || value > definition.MaxMonths {
			return fmt.Errorf("%s.timeline must be between %d and %d Months for instant goal key '%s'", prefix, definition.MinMonths, definition.MaxMonths, key)
		}
		if goal.Months != value {
			return fmt.Errorf("%s.timeline '%s' does not match months=%d for instant goal key '%s'", prefix, timeline, goal.Months, key)
		}

	case "short", "long":
		if unit != "year" && unit != "years" {
			return fmt.Errorf("%s.timeline must use Years for %s goal key '%s'", prefix, goalType, key)
		}
		if value < definition.MinYears || value > definition.MaxYears {
			return fmt.Errorf("%s.timeline must be between %d and %d Years for %s goal key '%s'", prefix, definition.MinYears, definition.MaxYears, goalType, key)
		}
		if goal.Years != value {
			return fmt.Errorf("%s.timeline '%s' does not match years=%d for %s goal key '%s'", prefix, timeline, goal.Years, goalType, key)
		}
	}

	return nil
}

func validateSelectedGoals(goals []SelectedGoal) error {
	if len(goals) == 0 {
		return errors.New("selected_goals_json must contain at least one goal")
	}

	seenKeys := make(map[string]struct{}, len(goals))

	for i, goal := range goals {
		prefix := fmt.Sprintf("selected_goals_json[%d]", i)
		key := strings.ToLower(strings.TrimSpace(goal.Key))
		label := strings.TrimSpace(goal.Label)
		goalType := strings.ToLower(strings.TrimSpace(goal.Type))

		if goalType == "" {
			return fmt.Errorf("%s.type is required", prefix)
		}
		if goalType != "instant" && goalType != "short" && goalType != "long" {
			return fmt.Errorf("%s.type '%s' is invalid; supported types are: instant, short, long", prefix, goal.Type)
		}

		if key == "" {
			return fmt.Errorf("%s.key is required for type '%s'", prefix, goalType)
		}

		definition, exists :=
			GetGoalDefinition(goal.Key)

		if !exists {
			supportedKeys :=
				SupportedGoalKeysByType(goalType)

			return fmt.Errorf(
				"selected_goals_json[%d].key '%s' is invalid for type '%s'; supported %s goal keys are: %s",
				i,
				goal.Key,
				goalType,
				goalType,
				strings.Join(
					supportedKeys,
					", ",
				),
			)
		}

		if definition.Type != goalType {
			supportedKeys :=
				SupportedGoalKeysByType(goalType)

			return fmt.Errorf(
				"selected_goals_json[%d].key '%s' is invalid for type '%s'; "+
					"the goal '%s' belongs to type '%s'. Select type '%s'. "+
					"Supported %s goal keys are: %s",
				i,
				goal.Key,
				goalType,
				definition.Key,
				definition.Type,
				definition.Type,
				goalType,
				strings.Join(
					supportedKeys,
					", ",
				),
			)
		}

		if _, duplicate := seenKeys[key]; duplicate {
			return fmt.Errorf("%s.key '%s' must be unique", prefix, key)
		}
		seenKeys[key] = struct{}{}

		if label == "" {
			return fmt.Errorf("%s.label is required for %s goal key '%s'", prefix, goalType, key)
		}
		if label != definition.Label {
			return fmt.Errorf(
				"%s.label '%s' is invalid for %s goal key '%s'; expected label is '%s'",
				prefix,
				label,
				goalType,
				key,
				definition.Label,
			)
		}

		if goal.Amount <= 0 {
			return fmt.Errorf("%s.amount must be greater than 0", prefix)
		}

		// Validate the human-readable timeline as well as the numeric years/months
		// fields. This prevents contradictory payloads such as timeline="13 Months"
		// with months=12 from being accepted.
		if err := validateGoalTimeline(prefix, goal, definition, goalType, key); err != nil {
			return err
		}

		switch goalType {
		case "instant":
			if goal.Months < definition.MinMonths || goal.Months > definition.MaxMonths {
				return fmt.Errorf(
					"%s.months must be between %d and %d for instant goal key '%s'",
					prefix,
					definition.MinMonths,
					definition.MaxMonths,
					key,
				)
			}
			if goal.Years != 0 {
				return fmt.Errorf("%s.years must not be provided for instant goal key '%s'", prefix, key)
			}

		case "short":
			if goal.Years < definition.MinYears || goal.Years > definition.MaxYears {
				return fmt.Errorf(
					"%s.years must be between %d and %d for short goal key '%s'",
					prefix,
					definition.MinYears,
					definition.MaxYears,
					key,
				)
			}
			if goal.Months != 0 {
				return fmt.Errorf("%s.months must not be provided for short goal key '%s'", prefix, key)
			}

		case "long":
			if goal.Years < definition.MinYears || goal.Years > definition.MaxYears {
				return fmt.Errorf(
					"%s.years must be between %d and %d for long goal key '%s'",
					prefix,
					definition.MinYears,
					definition.MaxYears,
					key,
				)
			}
			if goal.Months != 0 {
				return fmt.Errorf("%s.months must not be provided for long goal key '%s'", prefix, key)
			}
		}

		if key == "others" && strings.TrimSpace(goal.CustomName) == "" {
			return fmt.Errorf("%s.customName is required for instant goal key 'others'", prefix)
		}
	}

	return nil
}

func validateAssumptionRanges(req FinancialPlannerRequest) error {
	inflationFields := map[string]string{
		"education_1_inflation":  req.Education1Inflation,
		"marriage_1_inflation":   req.Marriage1Inflation,
		"retirement_inflation":   req.RetirementInflation,
		"wealth_inflation":       req.WealthInflation,
		"generic_goal_inflation": req.GenericGoalInflation,
	}
	for field, value := range inflationFields {
		if err := validateOptionalRange(field, value, 0, 20, false); err != nil {
			return err
		}
	}

	returnFields := map[string]string{
		"education_1_lumpsum_return":  req.Education1LumpsumReturn,
		"education_1_sip_return":      req.Education1SIPReturn,
		"marriage_1_lumpsum_return":   req.Marriage1LumpsumReturn,
		"marriage_1_sip_return":       req.Marriage1SIPReturn,
		"retirement_pre_return":       req.RetirementPreReturn,
		"retirement_post_return":      req.RetirementPostReturn,
		"wealth_lumpsum_return":       req.WealthLumpsumReturn,
		"wealth_sip_return":           req.WealthSIPReturn,
		"generic_goal_lumpsum_return": req.GenericGoalLumpsumReturn,
		"generic_goal_sip_return":     req.GenericGoalSIPReturn,
	}
	for field, value := range returnFields {
		if err := validateOptionalRange(field, value, 0, 50, false); err != nil {
			return err
		}
	}

	targetAgeFields := map[string]string{
		"education_1_target_age": req.Education1TargetAge,
		"marriage_1_target_age":  req.Marriage1TargetAge,
	}
	for field, value := range targetAgeFields {
		if strings.TrimSpace(value) == "" || strings.TrimSpace(value) == "0" {
			continue
		}
		if err := validateOptionalRange(field, value, 1, 100, true); err != nil {
			return err
		}
	}

	return nil
}

func validateOptionalRange(field, value string, min, max float64, wholeNumber bool) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("%s must be numeric", field)
	}
	if wholeNumber && number != float64(int64(number)) {
		return fmt.Errorf("%s must be a whole number", field)
	}
	if number < min || number > max {
		if wholeNumber {
			return fmt.Errorf("%s must be between %.0f and %.0f", field, min, max)
		}
		return fmt.Errorf("%s must be between %.0f and %.0f", field, min, max)
	}
	return nil
}

func validateInstantInsurance(req FinancialPlannerRequest) error {
	validations := []struct {
		flagValue any
		cover     string
	}{
		{req.ExistingLifeTerm, req.ExistingLifeTermCover},
		{req.ExistingHealth, req.ExistingHealthCover},
		{req.ExistingCritical, req.ExistingCriticalCover},
		{req.ExistingPersonal, req.ExistingPersonalCover},
	}

	for _, item := range validations {
		selected, provided, err := parseOptionalBoolean(item.flagValue)
		if err != nil {
			return errors.New("insurance details are not applicable for instant goals; insurance selections must be no/false/0 and cover amounts must be 0")
		}
		coverText := strings.TrimSpace(item.cover)
		cover := 0.0
		if coverText != "" {
			cover, err = strconv.ParseFloat(coverText, 64)
			if err != nil {
				return errors.New("insurance details are not applicable for instant goals; insurance selections must be no/false/0 and cover amounts must be 0")
			}
		}
		if (provided && selected) || cover != 0 {
			return errors.New("insurance details are not applicable for instant goals; insurance selections must be no/false/0 and cover amounts must be 0")
		}
	}
	return nil
}

func validateExistingInsurance(req FinancialPlannerRequest) error {
	validations := []struct {
		flagName  string
		flagValue any
		coverName string
		cover     string
	}{
		{"existing_life_term", req.ExistingLifeTerm, "existing_life_term_cover", req.ExistingLifeTermCover},
		{"existing_health", req.ExistingHealth, "existing_health_cover", req.ExistingHealthCover},
		{"existing_critical", req.ExistingCritical, "existing_critical_cover", req.ExistingCriticalCover},
		{"existing_personal", req.ExistingPersonal, "existing_personal_cover", req.ExistingPersonalCover},
	}

	for _, item := range validations {
		selected, provided, err := parseOptionalBoolean(item.flagValue)
		if err != nil {
			return fmt.Errorf("%s must be one of: yes, no, true, false, 1, 0", item.flagName)
		}

		coverText := strings.TrimSpace(item.cover)
		cover := 0.0
		if coverText != "" {
			cover, err = strconv.ParseFloat(coverText, 64)
			if err != nil {
				return fmt.Errorf("%s must be numeric", item.coverName)
			}
		}

		if !provided {
			if cover > 0 {
				return fmt.Errorf("%s is required when %s is greater than 0", item.flagName, item.coverName)
			}
			continue
		}

		if selected && cover <= 0 {
			return fmt.Errorf("%s must be greater than 0 when %s is selected", item.coverName, item.flagName)
		}
		if !selected && cover != 0 {
			return fmt.Errorf("%s must be 0 when %s is not selected", item.coverName, item.flagName)
		}
	}

	return nil
}

func parseOptionalBoolean(value any) (bool, bool, error) {
	if value == nil {
		return false, false, nil
	}

	switch typed := value.(type) {
	case bool:
		return typed, true, nil
	case float64:
		if typed == 1 {
			return true, true, nil
		}
		if typed == 0 {
			return false, true, nil
		}
		return false, true, errors.New("invalid boolean")
	case string:
		normalized := strings.ToLower(strings.TrimSpace(typed))
		if normalized == "" {
			return false, false, nil
		}
		switch normalized {
		case "true", "yes", "y", "1":
			return true, true, nil
		case "false", "no", "n", "0":
			return false, true, nil
		default:
			return false, true, errors.New("invalid boolean")
		}
	default:
		normalized := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
		if normalized == "" || normalized == "<nil>" {
			return false, false, nil
		}
		return false, true, errors.New("invalid boolean")
	}
}

func floatDefault(value string) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}

func supportedGoalKeysByType(

	goalType string,

) []string {

	keys := make([]string, 0)
	for key, definition := range goalMaster {
		if definition.Type == goalType {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys

}

func calculateAge(dob, now time.Time) int {
	age := now.Year() - dob.Year()
	if now.YearDay() < dob.YearDay() {
		age--
	}
	return age
}

func validateChildren(
	children []ChildDetail,
	childrenCount int,
	customerAge int,
	maritalStatus string,
	spouseAge int,
) error {

	seen := make(map[int]struct{}, len(children))

	providedChildren := 0
	for _, child := range children {
		if child.Index < 1 || child.Index > childrenCount {
			return fmt.Errorf(
				"child_%d details cannot be provided when children_count is %d",
				child.Index,
				childrenCount,
			)
		}
		if strings.TrimSpace(child.Name) != "" || strings.TrimSpace(child.DOB) != "" || child.Age != 0 {
			providedChildren++
		}
	}

	// Child details are optional when no child fields are supplied. Once any
	// child detail is supplied, complete details must match children_count.
	if providedChildren == 0 {
		return nil
	}

	if providedChildren != childrenCount {
		return fmt.Errorf(
			"children_count is %d but %d child detail(s) were provided; provide exactly one name and DOB for each child",
			childrenCount,
			providedChildren,
		)
	}

	for _, child := range children {

		if child.Index < 1 || child.Index > childrenCount {
			return fmt.Errorf(
				"child_%d details cannot be provided when children_count is %d",
				child.Index,
				childrenCount,
			)
		}

		if _, exists := seen[child.Index]; exists {
			return fmt.Errorf(
				"child_%d details are duplicated",
				child.Index,
			)
		}

		seen[child.Index] = struct{}{}

		childName := strings.TrimSpace(child.Name)
		childDOBText := strings.TrimSpace(child.DOB)

		if childName == "" || childDOBText == "" {
			return fmt.Errorf(
				"child_%d_name and child_%d_dob are both required when children_count is %d",
				child.Index,
				child.Index,
				childrenCount,
			)
		}

		if childName != "" {
			if len(childName) > 100 || !nameRegex.MatchString(childName) {
				return fmt.Errorf(
					"child_%d_name must be a valid name up to 100 characters",
					child.Index,
				)
			}
		}

		childAge := child.Age
		if childDOBText != "" {
			childDOB, err := time.Parse("02/01/2006", childDOBText)
			if err != nil {
				return fmt.Errorf("child_%d_dob must be in DD/MM/YYYY format", child.Index)
			}
			now := time.Now()
			if childDOB.After(now) {
				return fmt.Errorf("child_%d_dob cannot be a future date", child.Index)
			}
			childAge = calculateAge(childDOB, now)
		}
		if childAge < 0 || childAge > 25 {
			return fmt.Errorf("child age calculated from child_%d_dob must be between 0 and 25 years", child.Index)
		}

		// Customer must be at least 18 years older than the child.
		if customerAge-childAge < 18 {
			return fmt.Errorf(
				"child_%d_dob is invalid; customer age must be at least 18 years greater than child age",
				child.Index,
			)
		}

		// For married customers, spouse must also be at least 18 years older than the child.
		if strings.EqualFold(strings.TrimSpace(maritalStatus), "married") && spouseAge-childAge < 18 {
			return fmt.Errorf(
				"child_%d_dob is invalid; spouse age must be at least 18 years greater than child age",
				child.Index,
			)
		}
	}

	return nil
}
