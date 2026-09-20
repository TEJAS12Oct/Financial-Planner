package utils

import (
	dao "FinancialPlannerGo/app/domain/dao/financialPlanner"
	"strings"
)

type FinancialPlannerReportResponse struct {
	ID            uint   `json:"id"`
	UserID        uint   `json:"user_id"`
	FullName      string `json:"full_name"`
	Contact       string `json:"contact"`
	Email         string `json:"email"`
	DOB           string `json:"dob"`
	City          string `json:"city"`
	Status        string `json:"status"`
	ReportPDFURL  string `json:"report_pdf_url"`
	ReportPDFPath string `json:"report_pdf_path"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func ToFinancialPlannerReportResponse(report *dao.FinancialPlannerReport) FinancialPlannerReportResponse {
	return FinancialPlannerReportResponse{
		ID:            report.ID,
		UserID:        report.UserID,
		FullName:      report.FullName,
		Contact:       MaskPhone(report.Contact),
		Email:         MaskEmail(report.Email),
		DOB:           report.DOB,
		City:          report.City,
		Status:        report.Status,
		ReportPDFURL:  report.ReportPDFURL,
		ReportPDFPath: report.ReportPDFPath,
		CreatedAt:     report.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     report.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func MaskPhone(phone string) string {
	phone = strings.TrimSpace(phone)

	if len(phone) < 4 {
		return "****"
	}

	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}

func MaskEmail(email string) string {
	email = strings.TrimSpace(email)

	parts := strings.Split(email, "@")
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "****"
	}

	name := parts[0]

	if len(name) == 1 {
		return name + "***@" + parts[1]
	}

	if len(name) == 2 {
		return name[:1] + "***@" + parts[1]
	}

	return name[:2] + "***@" + parts[1]
}
