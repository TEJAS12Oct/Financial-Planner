package utils

// FinancialStatusFromScore converts the numeric financial score into the
// customer-facing status used across API responses, database records and PDFs.
//
// Score bands:
//
//	75-100 -> High
//	50-74  -> Average
//	0-49   -> Low
func FinancialStatusFromScore(score int) string {
	switch {
	case score >= 75:
		return "High"
	case score >= 50:
		return "Average"
	default:
		return "Low"
	}
}
