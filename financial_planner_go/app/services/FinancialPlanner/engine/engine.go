package engine

import (
	"FinancialPlannerGo/app/utils"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	CalculationEngineVersion = "1.0.0"
	AssumptionVersion        = "2026.01"
	ResponseSchemaVersion    = "boltpe-json-v1"
)

type assumptions struct{ inflation, sipReturn, lumpReturn float64 }

func Calculate(req utils.FinancialPlannerRequest) (*utils.BoltPePlanningResponse, error) {
	income := parse(req.MonthlyIncome)
	expenses := parse(req.MonthlyExpenses)
	savings := parse(req.MonthlySavings)
	if income <= 0 {
		return nil, fmt.Errorf("FP_VALIDATION_001: monthly income must be greater than zero")
	}
	if len(req.SelectedGoals) == 0 {
		return nil, fmt.Errorf("FP_GOAL_001: at least one goal is required")
	}

	reports := make([]utils.GoalReport, 0, len(req.SelectedGoals))
	priority := make([]string, 0, len(req.SelectedGoals))
	totalRequired := 0.0
	feasibleCount := 0
	for i, goal := range req.SelectedGoals {
		if goal.Amount <= 0 {
			return nil, fmt.Errorf("FP_GOAL_001: selected_goals_json[%d].amount must be greater than zero", i)
		}
		a := goalAssumptions(req, goal)
		months := goal.Years*12 + goal.Months
		if months <= 0 {
			months = defaultMonths(goal.Type)
		}
		years := float64(months) / 12.0
		future := goal.Amount * math.Pow(1+a.inflation/100, years)
		lump := future / math.Pow(1+a.lumpReturn/100, years)
		sip := recurringPaymentForFutureValue(future, a.sipReturn, months)
		totalRequired += sip
		feasible := savings >= sip
		if feasible {
			feasibleCount++
		}
		achievableMonths := monthsForFutureValue(future, a.sipReturn, savings)
		if feasible {
			achievableMonths = months
		}
		extra := achievableMonths - months
		if extra < 0 {
			extra = 0
		}
		requiredIncome := expenses + sip
		goalName := strings.TrimSpace(goal.CustomName)
		if goalName == "" {
			goalName = strings.TrimSpace(goal.Label)
		}
		if goalName == "" {
			goalName = strings.ReplaceAll(goal.Key, "_", " ")
		}
		goalType := displayGoalType(goal.Type)
		selectedTimeline := formatGoalTimeline(goal.Type, months)

		// Advisory text is driven by feasibility. When the selected timeline is
		// achievable with current savings, keep the positive guidance. When
		// additional time is required, explain how to keep the fixed timeline by
		// increasing investible surplus/income. This same feasibility rule is used
		// for Instant, Short-Term and Long-Term goals.
		advisory := "Congratulations! Your current monthly savings are sufficient to achieve this goal within the selected timeline. Maintain your current savings plan to stay on track toward your goal."
		if !feasible {
			advisory = "To achieve this goal within the fixed timeline, increase monthly investible surplus by the funding gap amount. This assumes monthly expenses remain unchanged and the additional income/surplus is invested toward this goal."
		}
		recommendations := goalRecommendations(feasible, goalType, a)
		report := utils.GoalReport{
			Type: strings.ToLower(strings.TrimSpace(goal.Type)), Title: goalName, Priority: priorityLabel(i), SubTitle: goalName,
			Recommendations: recommendations, SummaryTitle: "Goal Summary",
			Summary:     utils.GoalSummary{GoalName: goalName, GoalType: goalType, TargetAmount: formatINR(goal.Amount), TargetHorizon: selectedTimeline, Timeline: selectedTimeline},
			Assumptions: utils.GoalAssumptions{ExpectedReturn: pct(a.sipReturn), InflationUsed: pct(a.inflation), PlanningStyle: "Goal based SIP planning"},
			Feasibility: utils.GoalFeasibility{Title: "Boltpe Planning & Income Gap Analysis", AIAdvisoryNote: advisory, AchievableTimelineWithSavings: achievableText(achievableMonths, savings), CurrentMonthlySavings: formatINR(savings), ExtraTimeRequired: extraTimeText(extra), RequiredMonthlyIncome: formatINR(requiredIncome), SelectedTimeline: selectedTimeline},
			Table:       []utils.GoalTable{{Goal: goalName, GoalType: goalType, FutureCorpusNeeded: formatINR(future), LumpsumNeededToday: formatINR(lump), MonthlySavingRequired: formatINR(sip), YearsMonthsLeft: selectedTimeline}},
		}
		reports = append(reports, report)
		priority = append(priority, fmt.Sprintf("%s: target %s, required monthly contribution %s", goalName, selectedTimeline, formatINR(sip)))
	}

	spouseIncome := parse(req.SpouseIncome)
	familyIncome := income
	if strings.EqualFold(strings.TrimSpace(req.MaritalStatus), "married") && spouseIncome > 0 {
		familyIncome += spouseIncome
	}
	investibleSurplus := math.Max(familyIncome-expenses, 0)

	insurance := insuranceAnalysis(req, familyIncome)
	score := calculateScore(req, income, expenses, savings, feasibleCount, len(req.SelectedGoals), insurance)
	status := utils.FinancialStatusFromScore(score)
	allocation := assetAllocation(req.RiskAppetite)
	common := commonRecommendations(req, savings, totalRequired, insurance)

	data := utils.BoltPePlanningData{
		CommonRecommendations:         common,
		CommonRecommendationsTitle:    "Boltpe Priority Recommendations",
		CommonRecommendationsSubtitle: "Action points based on your current financial position and selected goals",
		CompanyContact:                map[string]interface{}{"brand": "BoltPe"},
		Customer:                      map[string]interface{}{"name": req.FullName, "full_name": req.FullName, "contact": req.Contact, "email": req.Email, "age": ageFromDOB(req.DOB), "city": req.City, "monthly_income": formatINR(income), "monthly_expenses": formatINR(expenses), "monthly_savings": formatINR(savings)},
		DatabaseStatus:                utils.BoltPeDBStatus{Saved: false, Message: "Calculated by BoltPe Planning Engine; persistence follows API processing"},
		EmailStatus:                   map[string]interface{}{"sent": false, "provider": "boltpe_mailer", "message": "Email delivery is handled after report persistence/PDF generation"},
		GeneratedAt:                   time.Now().UTC().Format(time.RFC3339), GoalTypeEngine: true,
		InsuranceInvestment: map[string]interface{}{
			"interpretation":     familyCashflowInterpretation(req, familyIncome, investibleSurplus, insurance),
			"gap_summary":        insurance["gap_summary"],
			"insurance_gap_rows": insurance["insurance_gap_rows"],
			"investment_rows":    investmentRows(investibleSurplus, allocation),
			"asset_allocation":   allocation,
		},
		PDFStatus: utils.BoltPePDFStatus{Saved: false}, PriorityRoadmap: priority, Reports: reports,
		Score: utils.BoltPeScore{Value: score, Label: status},
		Summary: map[string]interface{}{
			"title": "Customer Financial Snapshot",
			"points": []string{
				fmt.Sprintf("Total monthly income is %s, giving this report its main cashflow base.", formatINR(income)),
				fmt.Sprintf("Monthly expenses are %s and monthly savings are %s, and remaining funds are %s.", formatINR(expenses), formatINR(savings), formatINR(math.Max(income-expenses-savings, 0))),
				fmt.Sprintf("Current position is assessed as %s.", status),
				fmt.Sprintf("%d selected goal(s) are calculated by the BoltPe Planning Engine using the applicable goal assumptions.", len(req.SelectedGoals)),
				fmt.Sprintf("Savings ratio is %.1f%% and expense ratio is %.1f%% of monthly income.", ratio(savings, income), ratio(expenses, income)),
			},
		},
	}
	return &utils.BoltPePlanningResponse{Success: true, Data: data, Message: "Financial plan calculated successfully by BoltPe Planning Engine"}, nil
}

func goalAssumptions(req utils.FinancialPlannerRequest, g utils.SelectedGoal) assumptions {
	// BoltPe category assumption matrix (product-approved):
	// Instant Goal:    6% inflation, 10% SIP return, 10% lump-sum return.
	// Short-Term Goal: 6% inflation, 11% SIP return, 11% lump-sum return.
	// Long-Term Goal:  6% inflation, 12% SIP return, 12% lump-sum return.
	//
	// Retirement and Wealth Creation are allowed to use their dedicated request
	// assumptions, as required by the FSD. A missing or non-positive dedicated
	// value falls back to the approved long-term category rate.
	kind := strings.ToLower(strings.TrimSpace(g.Type))
	base := assumptions{inflation: 6, sipReturn: 11, lumpReturn: 11}
	switch kind {
	case "instant":
		base = assumptions{inflation: 6, sipReturn: 10, lumpReturn: 10}
	case "long", "long-term", "long_term":
		base = assumptions{inflation: 6, sipReturn: 12, lumpReturn: 12}
	}

	switch strings.ToLower(strings.TrimSpace(g.Key)) {
	case "retirement":
		// retirement_pre_return is the accumulation-stage return and is used
		// for both SIP accumulation and present lump-sum discounting. The
		// retirement_post_return field is reserved for future post-retirement
		// cashflow modelling, which is outside this corpus-only calculation.
		return assumptions{
			inflation:  firstPositive(parse(req.RetirementInflation), base.inflation),
			sipReturn:  firstPositive(parse(req.RetirementPreReturn), base.sipReturn),
			lumpReturn: firstPositive(parse(req.RetirementPreReturn), base.lumpReturn),
		}
	case "wealth":
		return assumptions{
			inflation:  firstPositive(parse(req.WealthInflation), base.inflation),
			sipReturn:  firstPositive(parse(req.WealthSIPReturn), base.sipReturn),
			lumpReturn: firstPositive(parse(req.WealthLumpsumReturn), base.lumpReturn),
		}
	default:
		return base
	}
}

// recurringPaymentForFutureValue calculates a beginning-of-month SIP
// (annuity due), exactly as specified in FSD v1.2:
// SIP = FV / [(((1+r)^N - 1) / r) * (1+r)].
func recurringPaymentForFutureValue(fv, annual float64, months int) float64 {
	if months <= 0 || fv <= 0 {
		return 0
	}
	r := annual / 1200
	if r == 0 {
		return fv / float64(months)
	}
	factor := ((math.Pow(1+r, float64(months)) - 1) / r) * (1 + r)
	if factor <= 0 || math.IsNaN(factor) || math.IsInf(factor, 0) {
		return 0
	}
	return fv / factor
}

// monthsForFutureValue is the inverse of the annuity-due SIP formula. It is
// used by the feasibility engine to calculate the achievable timeline with
// the customer's current monthly savings.
func monthsForFutureValue(fv, annual, payment float64) int {
	if payment <= 0 || fv <= 0 {
		return 0
	}
	r := annual / 1200
	if r == 0 {
		return int(math.Ceil(fv / payment))
	}
	inside := 1 + (fv*r)/(payment*(1+r))
	if inside <= 0 {
		return 0
	}
	n := math.Log(inside) / math.Log(1+r)
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return 0
	}
	return int(math.Ceil(n))
}
func defaultMonths(t string) int {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "instant":
		return 12
	case "long", "long-term", "long_term":
		return 300
	default:
		return 36
	}
}
func displayGoalType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "instant":
		return "Instant Goal"
	case "long", "long-term", "long_term":
		return "Long-Term Goal"
	default:
		return "Short-Term Goal"
	}
}
func priorityLabel(i int) string {
	if i == 0 {
		return "High"
	}
	if i < 3 {
		return "Medium"
	}
	return "Normal"
}
func pct(v float64) string {
	if math.Abs(v-math.Round(v)) < 0.000001 {
		return fmt.Sprintf("%.0f%% p.a.", v)
	}
	return fmt.Sprintf("%.2f%% p.a.", v)
}
func parse(s string) float64 { v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64); return v }
func firstPositive(v, d float64) float64 {
	if v > 0 {
		return v
	}
	return d
}
func ratio(a, b float64) float64 {
	if b <= 0 {
		return 0
	}
	return a / b * 100
}
func formatGoalTimeline(goalType string, months int) string {
	if months <= 0 {
		return "-"
	}

	normalizedType := strings.ToLower(strings.TrimSpace(goalType))
	if strings.Contains(normalizedType, "instant") {
		if months == 1 {
			return "1 Month"
		}
		return fmt.Sprintf("%d Months", months)
	}

	// Short-Term and Long-Term goals are displayed in years. Their configured
	// timelines are year based (for example 3, 5, 20 or 25 years). If a
	// non-whole-year value is ever supplied, keep it in years rather than
	// switching the PDF back to a month label.
	if months%12 == 0 {
		years := months / 12
		if years == 1 {
			return "1 Year"
		}
		return fmt.Sprintf("%d Years", years)
	}

	years := float64(months) / 12.0
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", years), "0"), ".") + " Years"
}

func formatDuration(months int) string {
	if months <= 0 {
		return "-"
	}
	y, m := months/12, months%12
	if y > 0 && m > 0 {
		return fmt.Sprintf("%d Years %d Months", y, m)
	}
	if y > 0 {
		if y == 1 {
			return "1 Year"
		}
		return fmt.Sprintf("%d Years", y)
	}
	if months == 1 {
		return "1 Month"
	}
	return fmt.Sprintf("%d Months", months)
}
func achievableText(months int, savings float64) string {
	if savings <= 0 {
		return "Not achievable with current savings"
	}
	return formatDuration(months)
}
func extraTimeText(months int) string {
	if months <= 0 {
		return "No extra time required"
	}
	return formatDuration(months)
}
func formatINR(v float64) string {
	n := int64(math.Round(v))
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return sign + "₹" + s
	}
	last := s[len(s)-3:]
	head := s[:len(s)-3]
	parts := []string{}
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return sign + "₹" + strings.Join(parts, ",") + "," + last
}
func ageFromDOB(dob string) int {
	t, err := time.Parse("02/01/2006", strings.TrimSpace(dob))
	if err != nil {
		return 0
	}
	now := time.Now()
	age := now.Year() - t.Year()
	if now.YearDay() < t.YearDay() {
		age--
	}
	return age
}
func goalRecommendations(feasible bool, goalType string, a assumptions) []string {
	out := []string{fmt.Sprintf("Review the goal amount and %.2f%% inflation assumption annually.", a.inflation)}
	if !feasible {
		out = append(out, "Increase monthly surplus or extend the selected timeline.")
	} else {
		out = append(out, "Maintain the required contribution and review progress at least annually.")
	}
	if goalType == "Instant Goal" {
		out = append(out, "Prefer low-volatility/liquid instruments because the goal horizon is short.")
	}
	return out
}
func familyCashflowInterpretation(req utils.FinancialPlannerRequest, familyIncome, investibleSurplus float64, insurance map[string]interface{}) []string {
	married := strings.EqualFold(strings.TrimSpace(req.MaritalStatus), "married")
	children, _ := strconv.Atoi(strings.TrimSpace(req.ChildrenCount))
	dependents, _ := strconv.Atoi(strings.TrimSpace(req.Dependents))
	spouseAge, _ := strconv.Atoi(strings.TrimSpace(req.SpouseAge))
	spouseIncome := parse(req.SpouseIncome)

	familyParts := []string{"Self"}
	totalMembers := 1
	if married {
		familyParts = append(familyParts, "Spouse")
		totalMembers++
	}
	if children > 0 {
		familyParts = append(familyParts, fmt.Sprintf("%d Child/Children", children))
		totalMembers += children
	}
	if dependents > 0 {
		familyParts = append(familyParts, fmt.Sprintf("%d Dependent(s)", dependents))
		totalMembers += dependents
	}

	gapSummary, _ := insurance["gap_summary"].(map[string]interface{})
	recommended := valueString(gapSummary, "Total Recommended Cover")
	existing := valueString(gapSummary, "Total Existing Cover")
	gap := valueString(gapSummary, "Total Protection Gap")

	risk := strings.TrimSpace(req.RiskAppetite)
	if risk == "" {
		risk = "Not specified"
	} else {
		risk = strings.ToUpper(risk[:1]) + strings.ToLower(risk[1:])
	}

	out := []string{
		fmt.Sprintf("Family definition considered: %s.", strings.Join(familyParts, " + ")),
		fmt.Sprintf("Total members considered for protection planning: %d.", totalMembers),
		fmt.Sprintf("Total family monthly income considered: %s.", formatINR(familyIncome)),
		fmt.Sprintf("Total recommended insurance cover across protection areas: %s.", recommended),
		fmt.Sprintf("Total existing insurance cover entered by customer: %s.", existing),
		fmt.Sprintf("Total BoltPe identified protection gap: %s.", gap),
		fmt.Sprintf("Monthly surplus considered for investment planning: %s.", formatINR(investibleSurplus)),
		fmt.Sprintf("Asset allocation is based on selected risk appetite: %s.", risk),
	}
	if married {
		if spouseAge > 0 {
			out = append(out, fmt.Sprintf("Spouse age considered: %d years.", spouseAge))
		}
		out = append(out, fmt.Sprintf("Spouse monthly income considered: %s.", formatINR(spouseIncome)))
	}
	return out
}

func valueString(m map[string]interface{}, key string) string {
	if m == nil {
		return "-"
	}
	v := strings.TrimSpace(fmt.Sprint(m[key]))
	if v == "" || v == "<nil>" {
		return "-"
	}
	return v
}

func assetAllocation(risk string) map[string]interface{} {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "low":
		return map[string]interface{}{"Equity": "30%", "Debt": "60%", "Gold / Alternatives": "10%"}
	case "high":
		return map[string]interface{}{"Equity": "75%", "Debt": "15%", "Gold / Alternatives": "10%"}
	default:
		return map[string]interface{}{"Equity": "60%", "Debt": "30%", "Gold / Alternatives": "10%"}
	}
}
func investmentRows(surplus float64, allocation map[string]interface{}) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"Investment Bucket":      "Emergency Fund",
			"Suggested Product Type": "Liquid fund / Sweep FD",
			"Monthly Allocation":     formatINR(surplus * 0.20),
			"Purpose":                "Safety and liquidity",
		},
		{
			"Investment Bucket":      "Goal SIP",
			"Suggested Product Type": "Diversified SIP portfolio",
			"Monthly Allocation":     formatINR(surplus * 0.50),
			"Purpose":                "Goal funding",
		},
		{
			"Investment Bucket":      "Long-term Wealth SIP",
			"Suggested Product Type": allocationSummary(allocation),
			"Monthly Allocation":     formatINR(surplus * 0.30),
			"Purpose":                "Long-term wealth creation",
		},
	}
}

func allocationSummary(allocation map[string]interface{}) string {
	return fmt.Sprintf("%s Equity, %s Debt, %s Gold",
		valueString(allocation, "Equity"),
		valueString(allocation, "Debt"),
		valueString(allocation, "Gold / Alternatives"),
	)
}

func insuranceAnalysis(req utils.FinancialPlannerRequest, monthlyIncome float64) map[string]interface{} {
	annual := monthlyIncome * 12
	dep, _ := strconv.Atoi(strings.TrimSpace(req.Dependents))
	life := annual * (10 + float64(dep))
	health := math.Max(500000, annual*0.5)
	critical := math.Max(500000, annual)
	accident := math.Max(1000000, annual*5)
	types := []struct {
		name                  string
		recommended, existing float64
	}{{"Life / Term Insurance", life, parse(req.ExistingLifeTermCover)}, {"Health Insurance", health, parse(req.ExistingHealthCover)}, {"Critical Illness Insurance", critical, parse(req.ExistingCriticalCover)}, {"Personal Accident Insurance", accident, parse(req.ExistingPersonalCover)}}
	rows := []map[string]interface{}{}
	totalR, totalE := 0.0, 0.0
	for _, x := range types {
		gap := math.Max(x.recommended-x.existing, 0)
		status := "Adequate"
		if gap > 0 {
			status = "Gap Identified"
		}
		rows = append(rows, map[string]interface{}{"Insurance Type": x.name, "Recommended Cover": formatINR(x.recommended), "Existing Cover": formatINR(x.existing), "Protection Gap": formatINR(gap), "Gap Status": status})
		totalR += x.recommended
		totalE += x.existing
	}
	totalGap := math.Max(totalR-totalE, 0)
	status := "Adequate"
	if totalGap > 0 {
		status = "Action Required"
	}
	return map[string]interface{}{"gap_summary": map[string]interface{}{"Total Recommended Cover": formatINR(totalR), "Total Existing Cover": formatINR(totalE), "Total Protection Gap": formatINR(totalGap), "Gap Status": status, "gap_value": totalGap}, "insurance_gap_rows": rows}
}
func calculateScore(req utils.FinancialPlannerRequest, income, expenses, savings float64, feasible, totalGoals int, insurance map[string]interface{}) int {
	score := 0.0
	sr := ratio(savings, income)
	if sr >= 30 {
		score += 25
	} else if sr >= 20 {
		score += 20
	} else if sr >= 10 {
		score += 14
	} else {
		score += 7
	}
	er := ratio(expenses, income)
	if er <= 50 {
		score += 15
	} else if er <= 70 {
		score += 10
	} else {
		score += 5
	}
	if savings >= expenses*3 {
		score += 15
	} else if savings > 0 {
		score += 8
	}
	gap := 0.0
	if gs, ok := insurance["gap_summary"].(map[string]interface{}); ok {
		gap, _ = gs["gap_value"].(float64)
	}
	if gap == 0 {
		score += 20
	} else if parse(req.ExistingLifeTermCover)+parse(req.ExistingHealthCover) > 0 {
		score += 10
	} else {
		score += 4
	}
	if totalGoals > 0 {
		score += 15 * float64(feasible) / float64(totalGoals)
	}
	switch strings.ToLower(req.InvestmentExperience) {
	case "advance", "advanced":
		score += 10
	case "intermediate":
		score += 8
	default:
		score += 5
	}
	if score > 100 {
		score = 100
	}
	return int(math.Round(score))
}
func commonRecommendations(req utils.FinancialPlannerRequest, savings, totalRequired float64, insurance map[string]interface{}) []string {
	// Advisor Action Plan is intentionally presented as a stable set of practical
	// customer actions. The first item explains what to do when the calculated SIP
	// exceeds available savings; the remaining items are applicable across plans.
	// Keep the wording here (the calculation layer) so JSON and PDF use the same
	// recommendation content.
	return []string{
		"If current savings are lower than the required SIP, either extend the timeline or increase monthly income/surplus as shown in the feasibility analysis.",
		"Review all goal values at least once every year and update them for inflation, income changes and family changes.",
		"Use annual step-up SIP whenever income increases instead of keeping the same contribution for many years.",
		"If any plan feels heavy on the current budget, you may review the Achievable Timeline, adjust it gradually, and continue the plan comfortably as your monthly surplus improves.",
		"Control non-essential expenses and redirect the saved amount toward your priority financial goals.",
		"Keep emergency fund and insurance protection separate from lifestyle purchases and long-term wealth goals.",
	}
}
