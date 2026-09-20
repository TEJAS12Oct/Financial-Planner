package engine

import (
	"FinancialPlannerGo/app/utils"
	"math"
	"testing"
)

func TestBoltPeGoalAssumptionMatrix(t *testing.T) {
	tests := []struct {
		name                            string
		amount, inflation, annualReturn float64
		months                          int
		future, lump, sip               float64
	}{
		{"instant iphone", 90000, 6, 10, 12, 95400, 86727, 7529},
		{"short car", 1000000, 6, 11, 36, 1191016, 871550, 27820},
		{"long retirement", 20000000, 6, 12, 300, 85837414, 5049241, 45234},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			years := float64(tc.months) / 12
			fv := tc.amount * math.Pow(1+tc.inflation/100, years)
			lump := fv / math.Pow(1+tc.annualReturn/100, years)
			sip := recurringPaymentForFutureValue(fv, tc.annualReturn, tc.months)
			if math.Abs(fv-tc.future) > 1 {
				t.Fatalf("future corpus %.2f; want %.2f", fv, tc.future)
			}
			if math.Abs(lump-tc.lump) > 1 {
				t.Fatalf("lump %.2f; want %.2f", lump, tc.lump)
			}
			if math.Abs(sip-tc.sip) > 1 {
				t.Fatalf("sip %.2f; want %.2f", sip, tc.sip)
			}
		})
	}
}

func TestCalculateUsesSameFrameworkForInstantShortLong(t *testing.T) {
	req := utils.FinancialPlannerRequest{FullName: "Test User", Contact: "9999999999", Email: "test@example.com", DOB: "01/01/1990", City: "Pune", MonthlyIncome: "100000", MonthlyExpenses: "40000", MonthlySavings: "30000", RiskAppetite: "medium", InvestmentExperience: "intermediate", SelectedGoals: []utils.SelectedGoal{
		{Key: "iphone", Label: "Phone", Type: "instant", Amount: 90000, Months: 12},
		{Key: "car_purchase", Label: "Car Purchase", Type: "short", Amount: 1000000, Years: 3},
		{Key: "retirement", Label: "Retirement Planning", Type: "long", Amount: 20000000, Years: 25},
	}}
	resp, err := Calculate(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data.Reports) != 3 {
		t.Fatalf("reports=%d, want 3", len(resp.Data.Reports))
	}
	for i, r := range resp.Data.Reports {
		if len(r.Table) != 1 || r.Table[0].FutureCorpusNeeded == "" || r.Table[0].LumpsumNeededToday == "" || r.Table[0].MonthlySavingRequired == "" {
			t.Fatalf("goal %d missing common calculation outputs", i)
		}
	}
}

func TestFormatGoalTimelineByCategory(t *testing.T) {
	tests := []struct {
		goalType string
		months   int
		want     string
	}{
		{goalType: "instant", months: 12, want: "12 Months"},
		{goalType: "instant", months: 6, want: "6 Months"},
		{goalType: "short", months: 36, want: "3 Years"},
		{goalType: "short", months: 60, want: "5 Years"},
		{goalType: "long", months: 240, want: "20 Years"},
		{goalType: "long", months: 300, want: "25 Years"},
	}

	for _, tt := range tests {
		if got := formatGoalTimeline(tt.goalType, tt.months); got != tt.want {
			t.Fatalf("formatGoalTimeline(%q, %d) = %q; want %q", tt.goalType, tt.months, got, tt.want)
		}
	}
}
func TestGoalSpecificRetirementAndWealthAssumptionsOverrideLongTermDefaults(t *testing.T) {
	req := utils.FinancialPlannerRequest{
		RetirementInflation:  "5",
		RetirementPreReturn:  "10",
		RetirementPostReturn: "7",
		WealthInflation:      "7",
		WealthLumpsumReturn:  "11",
		WealthSIPReturn:      "13",
	}
	retirement := goalAssumptions(req, utils.SelectedGoal{Key: "retirement", Type: "long"})
	if retirement.inflation != 5 || retirement.sipReturn != 10 || retirement.lumpReturn != 10 {
		t.Fatalf("retirement assumptions = %+v; want inflation=5, sip=10, lump=10", retirement)
	}
	wealth := goalAssumptions(req, utils.SelectedGoal{Key: "wealth", Type: "long"})
	if wealth.inflation != 7 || wealth.sipReturn != 13 || wealth.lumpReturn != 11 {
		t.Fatalf("wealth assumptions = %+v; want inflation=7, sip=13, lump=11", wealth)
	}
}
func TestAchievableTimelineUsesAnnuityDueInverse(t *testing.T) {
	fv := 95400.0
	annual := 10.0
	payment := recurringPaymentForFutureValue(fv, annual, 12)
	if got := monthsForFutureValue(fv, annual, payment); got != 12 {
		t.Fatalf("monthsForFutureValue = %d; want 12", got)
	}
}
