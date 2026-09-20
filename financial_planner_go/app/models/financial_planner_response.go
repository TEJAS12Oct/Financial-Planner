package models

type ReportCustomerResponse struct {
	FullName string `json:"full_name" example:"Ravi Sharma"`
	Email    string `json:"email" example:"ravi@example.com"`
	Age      int    `json:"age" example:"36"`
}

type ReportScoreResponse struct {
	Value int    `json:"value" example:"62"`
	Label string `json:"label" example:"Average"`
}

type ReportEmailStatusResponse struct {
	Sent    bool   `json:"sent" example:"true"`
	Message string `json:"message" example:"Report email sent successfully to ravi@example.com."`
}

type ReportPDFStatusResponse struct {
	Saved         bool   `json:"saved" example:"true"`
	ReportPDFURL  string `json:"report_pdf_url"`
	ReportPDFPath string `json:"report_pdf_path"`
}

type ReportDataResponse struct {
	ReportID      uint                      `json:"report_id" example:"101"`
	Status        string                    `json:"status" example:"report_generated"`
	Customer      ReportCustomerResponse    `json:"customer"`
	Score         ReportScoreResponse       `json:"score"`
	ReportPDFURL  string                    `json:"report_pdf_url"`
	ReportPDFPath string                    `json:"report_pdf_path"`
	EmailStatus   ReportEmailStatusResponse `json:"email_status"`
	PDFStatus     ReportPDFStatusResponse   `json:"pdf_status"`
}

type GenerateReportSuccessResponse struct {
	Success bool               `json:"success" example:"true"`
	Message string             `json:"message" example:"Financial planner report generated successfully"`
	Data    ReportDataResponse `json:"data"`
}

type APIErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Code    string `json:"code" example:"VAL_001"`
	Message string `json:"message" example:"invalid request payload"`
}
