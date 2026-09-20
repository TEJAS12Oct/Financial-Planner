package FinancialPlanner

import (
	"FinancialPlannerGo/app/utils"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SMTPStatusData struct {
	Configured                      bool   `json:"configured"`
	Host                            string `json:"host"`
	Port                            int    `json:"port"`
	User                            string `json:"user"`
	PasswordSet                     bool   `json:"password_set"`
	PasswordLengthAfterSpaceRemoval int    `json:"password_length_after_space_removal"`
}

type SMTPStatusResponse struct {
	Success bool           `json:"success"`
	SMTP    SMTPStatusData `json:"smtp"`
}

type SimpleErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type TestEmailStatusResponse struct {
	Sent    bool   `json:"sent"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type TestEmailResponse struct {
	Success     bool                    `json:"success"`
	EmailStatus TestEmailStatusResponse `json:"email_status"`
}

type ReportListItem struct {
	ReportID     uint      `json:"report_id"`
	UserID       uint      `json:"user_id"`
	FullName     string    `json:"full_name"`
	Contact      string    `json:"contact"`
	Email        string    `json:"email"`
	Status       string    `json:"status"`
	ScoreValue   int       `json:"score_value"`
	ScoreLabel   string    `json:"score_label"`
	ReportPDFURL string    `json:"report_pdf_url"`
	CreatedAt    time.Time `json:"created_at"`
}

type ListReportsResponse struct {
	Success bool             `json:"success"`
	Message string           `json:"message"`
	Count   int              `json:"count"`
	Data    []ReportListItem `json:"data"`
}

type TestEmailRequest struct {
	ToEmail string `json:"to_email,omitempty" example:"sample.user@example.com"`
	Email   string `json:"email,omitempty" example:"sample.user@example.com"`
}

// Home godoc
// @Summary Render the Financial Planner home page
// @ID get_financial_planner_home
// @Description Returns the BoltPe Financial Planner application home page.
// @Tags Pages
// @Produce html
// @Success 200 {string} string "HTML page"
// @Router /boltpe-financial-planner/ [get]
func (ctrl *FinancialPlannerController) Home(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(
		http.StatusOK,
		`<!doctype html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width,initial-scale=1">
	<title>Boltpay Financial Planner API</title>
	<style>
		body {
			font-family: "HK Grotesk", Arial, sans-serif;
			max-width: 900px;
			margin: 48px auto;
			padding: 0 20px;
			color: #172033;
		}
		code {
			background: #f3f5f7;
			padding: 3px 6px;
			border-radius: 4px;
		}
		a {
			color: #0759c7;
		}
	</style>
</head>
<body>
	<h1>Dynamic Individual Financial Planner – Boltpe </h1>
	<p>The API service is running.</p>
	<p>Swagger documentation: <a href="/boltpe-financial-planner/apidocs/index.html">/apidocs/</a></p>
	<p>Health check: <a href="/boltpe-financial-planner/health">/health</a></p>
</body>
</html>`,
	)
}

// SMTPStatus godoc
// @Summary Get email-service status
// @ID get_financial_planner_smtp_status
// @Description Returns the configured Financial Planner email-service status.
// @Tags Email
// @Produce json
// // @Security ApiKeyAuth
// @Success 200 {object} SMTPStatusResponse
// @Failure 403 {object} models.APIErrorResponse "Client IP is not allowed"
// @Router /api/financial-planner/smtp-status [get]
func (ctrl *FinancialPlannerController) SMTPStatus(c *gin.Context) {
	status := ctrl.service.GetSMTPStatus()

	c.JSON(http.StatusOK, SMTPStatusResponse{
		Success: true,
		SMTP: SMTPStatusData{
			Configured:                      status.Configured,
			Host:                            status.Host,
			Port:                            status.Port,
			User:                            status.User,
			PasswordSet:                     status.PasswordSet,
			PasswordLengthAfterSpaceRemoval: status.PasswordLengthAfterSpaceRemoval,
		},
	})
}

// TestEmail godoc
// @Summary Send a test email
// @ID post_financial_planner_test_email
// @Description Sends a test email through the configured Financial Planner email service.
// @Tags Email
// @Accept json
// @Produce json
// // @Security ApiKeyAuth
// @Param request body TestEmailRequest true "Test email request"
// @Success 200 {object} TestEmailResponse
// @Failure 400 {object} models.APIErrorResponse
// @Failure 403 {object} models.APIErrorResponse "Client IP is not allowed"
// @Router /api/financial-planner/test-email [post]
func (ctrl *FinancialPlannerController) TestEmail(c *gin.Context) {
	var req TestEmailRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse("VAL_001", "invalid request payload"))
		return
	}

	toEmail := strings.TrimSpace(req.ToEmail)
	if toEmail == "" {
		toEmail = strings.TrimSpace(req.Email)
	}
	if toEmail == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse("VAL_001", "to_email or email is required"))
		return
	}

	if err := ctrl.service.SendTestEmail(c.Request.Context(), toEmail); err != nil {
		writeTestEmailFailure(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, TestEmailResponse{
		Success: true,
		EmailStatus: TestEmailStatusResponse{
			Sent:    true,
			Message: "Test email accepted by SMTP server",
		},
	})
}

func writeTestEmailFailure(c *gin.Context, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "SMTP email sending failed"
	}

	c.JSON(http.StatusInternalServerError, utils.APIResponse{
		Code:    utils.ErrCodeEmail,
		Message: "Test email could not be sent",
		Success: false,
		Data: map[string]interface{}{
			"email_status": TestEmailStatusResponse{Sent: false, Message: message, Error: message},
		},
	})
}

// GeneratePDFDirect godoc
// @Summary Generate a financial planner PDF and upload it to S3
// @ID post_financial_planner_report_pdf_direct
// @Description Validates the request, generates the PDF, uploads it to S3, stores the report details, and returns JSON metadata including report_pdf_url.
// @Tags Reports
// @Accept json
// @Produce json
// // @Security ApiKeyAuth
// // @Param Accept header string true "Accepted response content type" default(application/json)
// // @Param X-API-Key header string true "Financial Planner API key"
// // @Param X-User-ID header int true "Authenticated user ID" example(1)
// @Param request body models.FinancialPlannerRequest true "Financial planner request"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} models.APIErrorResponse
// @Failure 429 {object} models.APIErrorResponse
// @Failure 500 {object} models.APIErrorResponse "BoltPe calculation, persistence or PDF generation failure"
// @Router /api/boltpe-financial-planner/report-pdf-direct [post]
func (ctrl *FinancialPlannerController) GeneratePDFDirect(c *gin.Context) {
	var req utils.FinancialPlannerRequest

	rawRequestBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse("VAL_001", "Unable to read request payload"))
		return
	}
	if len(rawRequestBytes) == 0 || !json.Valid(rawRequestBytes) {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse("VAL_001", "Invalid financial planner request payload"))
		return
	}
	if err := json.Unmarshal(rawRequestBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse("VAL_001", err.Error()))
		return
	}

	utils.NormalizeFinancialPlannerRequest(&req)

	if err := utils.ValidateFinancialPlannerRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse("VAL_001", err.Error()))
		return
	}

	userID := uint(1)
	pendingReport, err := ctrl.service.CreatePDFDirectReportRequest(userID, req, rawRequestBytes)
	if err != nil {
		slog.Error("failed to store direct PDF request before planning call", "error", err, "email", req.Email)
		c.JSON(http.StatusInternalServerError, utils.ErrorResponse(
			"FP_STORAGE_ERROR",
			"Unable to store the financial planner request",
		))
		return
	}

	pdfData, filename, calculationBytes, err := ctrl.service.GenerateBoltPePDFDirect(
		c.Request.Context(),
		req,
	)
	if err != nil {
		status := classifyCalculationFailure(err)

		if status == "pdf_generation_failed" {
			if updateErr := ctrl.service.MarkDirectPDFGenerationFailed(
				pendingReport.ID,
				err,
				calculationBytes,
			); updateErr != nil {
				slog.Error("failed to persist direct PDF generation failure", "report_id", pendingReport.ID, "error", updateErr)
			}
			c.JSON(http.StatusInternalServerError, utils.APIResponse{
				Code:    utils.ErrCodePDF,
				Message: "Unable to generate the financial planning PDF.",
				Success: false,
				Data:    map[string]interface{}{"report_id": pendingReport.ID},
			})
			return
		}

		if updateErr := ctrl.service.MarkStructuredJSONReportFailed(
			pendingReport.ID,
			status,
			err,
			calculationBytes,
		); updateErr != nil {
			slog.Error("failed to persist BoltPe Planning Engine direct PDF failure", "report_id", pendingReport.ID, "error", updateErr)
		}
		detail := utils.ClassifyFinancialPlannerError(err)
		c.JSON(detail.HTTPStatus, utils.APIResponse{
			Code:    detail.Code,
			Message: detail.Message,
			Success: false,
			Data:    map[string]interface{}{"report_id": pendingReport.ID, "failed_module": detail.Module},
		})
		return
	}

	completedReport, err := ctrl.service.CompleteDirectPDFReport(
		c.Request.Context(),
		pendingReport.ID,
		calculationBytes,
		pdfData,
		filename,
	)
	if err != nil {
		slog.Error("failed to upload or finalize direct PDF report", "report_id", pendingReport.ID, "error", err)
		c.JSON(http.StatusInternalServerError, utils.APIResponse{
			Code:    utils.ErrCodeS3,
			Message: "PDF was generated but could not be uploaded to S3 or saved successfully.",
			Success: false,
			Data: map[string]interface{}{
				"report_id": pendingReport.ID,
				"error":     err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Financial planning PDF generated successfully",
		"data": gin.H{
			"report_id":        completedReport.ID,
			"full_name":        completedReport.FullName,
			"email":            completedReport.Email,
			"pdf_status":       completedReport.PDFStatus,
			"report_pdf_url":   completedReport.ReportPDFURL,
		},
	})
}

func writePDFFailure(c *gin.Context, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "PDF generation failed"
	}

	c.JSON(http.StatusInternalServerError, SimpleErrorResponse{
		Success: false,
		Error:   message,
	})
}

func extractUpstreamError(body []byte) string {
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &payload); err == nil {
		return firstNonEmpty(
			payload.Error,
			payload.Message,
			"PDF generation failed",
		)
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		return "PDF generation failed"
	}

	return message
}

// ServeReport godoc
// @Summary Download or preview a generated report PDF
// @ID get_financial_planner_report_file
// @Description Serves a generated report through a signed and expiring URL.
// @Tags Reports
// @Produce json
// @Param reportPath path string true "Generated report path, for example publictpap/report.pdf"
// @Param expires query int64 true "Signed URL expiry timestamp"
// @Param signature query string true "Signed URL signature"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} models.APIErrorResponse
// @Failure 401 {object} models.APIErrorResponse
// @Failure 404 {object} models.APIErrorResponse
// @Router /api/boltpe-financial-planner/reports/{reportPath} [get]
func (ctrl *FinancialPlannerController) ServeReport(c *gin.Context) {
	reportReference := strings.TrimPrefix(
		strings.TrimSpace(c.Param("reportPath")),
		"/",
	)

	if reportReference == "" ||
		!strings.HasSuffix(strings.ToLower(reportReference), ".pdf") {
		c.JSON(
			http.StatusBadRequest,
			utils.ErrorResponse("VAL_001", "invalid report path"),
		)
		return
	}

	pdfData, err := ctrl.service.GetGeneratedReportPDF(
		c.Request.Context(),
		reportReference,
	)
	if err != nil {
		c.JSON(
			http.StatusNotFound,
			utils.ErrorResponse("FP_404", "generated PDF report not found in S3"),
		)
		return
	}

	filename := filepath.Base(reportReference)
	c.Header("Cache-Control", "private, no-store")
	c.Header(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, filename),
	)
	c.Data(http.StatusOK, "application/pdf", pdfData)
}

// ListReports godoc
// @Summary List stored financial planner reports
// @ID list_financial_planner_reports
// @Description Returns report metadata stored in the BoltPe database in descending creation order. This endpoint does not call BoltPe Planning Engine.
// @Tags Financial Planner Reports
// @Produce json
// // @Security ApiKeyAuth
// // @Param X-User-ID header int true "Authenticated user ID" example(1)
// @Success 200 {object} ListReportsResponse "Reports fetched successfully"
// @Failure 500 {object} models.APIErrorResponse "Unable to fetch reports from the database"
// @Router /api/boltpe-financial-planner/reports [get]
func (ctrl *FinancialPlannerController) ListReports(c *gin.Context) {
	// X-User-ID header validation is currently disabled.
	// userID, err := getUserID(c)
	// if err != nil {
	// 	c.JSON(
	// 		http.StatusBadRequest,
	// 		utils.ErrorResponse(
	// 			"INVALID_USER_ID",
	// 			err.Error(),
	// 		),
	// 	)
	// 	return
	// }
	userID := uint(1)

	reports, err := ctrl.service.GetReports(userID)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			utils.ErrorResponse(
				"FP_LIST_500",
				"Unable to fetch financial planner reports",
			),
		)
		return
	}

	items := make([]ReportListItem, 0, len(reports))
	for _, report := range reports {
		// Prefer the permanent S3 object key. report_pdf_url may contain an
		// expired signature or an older API route stored by a previous build.
		reportReference := strings.TrimSpace(report.ReportPDFPath)
		if reportReference == "" {
			reportReference = strings.TrimSpace(report.ReportPDFURL)
		}

		reportPDFURL := normalizeBoltPeReportURL(reportReference)
		if reportPDFURL != "" && ctrl.reportSigner != nil {
			reportPDFURL = ctrl.reportSigner.SignURL(reportPDFURL)
		}
		items = append(items, ReportListItem{
			ReportID:     report.ID,
			UserID:       report.UserID,
			FullName:     report.FullName,
			Contact:      report.Contact,
			Email:        report.Email,
			Status:       report.Status,
			ScoreValue:   report.ScoreValue,
			ScoreLabel:   report.ScoreLabel,
			ReportPDFURL: reportPDFURL,
			CreatedAt:    report.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, ListReportsResponse{
		Success: true,
		Message: "Financial planner reports fetched successfully",
		Count:   len(items),
		Data:    items,
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
