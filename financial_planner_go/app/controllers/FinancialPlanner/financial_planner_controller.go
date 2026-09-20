package FinancialPlanner

import (
	financialPlanner "FinancialPlannerGo/app/domain/dao/financialPlanner"
	"FinancialPlannerGo/app/models"
	appsecurity "FinancialPlannerGo/app/security"
	service "FinancialPlannerGo/app/services/FinancialPlanner"
	"FinancialPlannerGo/app/utils"
	"context"

	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type FinancialPlannerController struct {
	service      service.FinancialPlannerService
	reportSigner *appsecurity.ReportSigner
}

func NewFinancialPlannerController(
	financialPlannerService service.FinancialPlannerService,
	reportSigner *appsecurity.ReportSigner,
) *FinancialPlannerController {
	return &FinancialPlannerController{
		service:      financialPlannerService,
		reportSigner: reportSigner,
	}
}

// GenerateStructuredJSONReport godoc
//
// @Summary Generate structured BoltPe financial planner JSON
// @ID generate_structured_financial_planner_json
// @Description Accepts a BoltPe request, runs the internal BoltPe Planning Engine, stores request/assumptions, calculation output and the structured BoltPe response.
// @Description Returns structured JSON immediately; PDF generation, S3 upload and email delivery continue in the background.
// @Description Goal timelines are validated strictly: Instant 6-12 Months, Short 1-5 Years, Long 20-25 Years; timeline text must match months/years.
// @Tags Financial Planner Reports
// @Accept json
// @Produce json
// // @Security ApiKeyAuth
// // @Param Accept header string true "Accepted response content type" default(application/json)
// // @Param X-API-Key header string true "Financial Planner API key"
// // @Param X-User-ID header int true "Authenticated user ID" example(1)
// @Param request body models.FinancialPlannerRequest true "Financial planner request"
// @Success 200 {object} models.StructuredJSONReportResponse "Structured report generated successfully"
// @Failure 400 {object} models.APIErrorResponse "Invalid request payload"
// @Failure 413 {object} models.APIErrorResponse "Request body exceeds the configured limit"
// @Failure 429 {object} models.APIErrorResponse "Rate limit exceeded"
// @Failure 500 {object} models.APIErrorResponse "BoltPe calculation, serialization or storage failure"
// @Router /api/boltpe-financial-planner/report-json [post]
func classifyCalculationFailure(err error) string {
	if err == nil {
		return "calculation_failed"
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "pdf generation") || strings.Contains(message, "pdf preparation") {
		return "pdf_generation_failed"
	}
	return "calculation_failed"
}

func (ctrl *FinancialPlannerController) GenerateStructuredJSONReport(
	c *gin.Context,
) {
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
		c.JSON(
			http.StatusBadRequest,
			utils.ErrorResponse(
				"VAL_001",
				fmt.Sprintf(
					"Invalid financial planner request payload: %v",
					err,
				),
			),
		)
		return
	}

	utils.NormalizeFinancialPlannerRequest(&req)

	if err := utils.ValidateFinancialPlannerRequest(req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			utils.ErrorResponse(
				"VAL_001",
				err.Error(),
			),
		)
		return
	}

	slog.Info(
		"generating structured financial planner report",
		"email",
		req.Email,
		"goals",
		len(req.SelectedGoals),
	)

	// X-User-ID header validation is currently disabled.
	userID := uint(1)

	// Persist the request before calling BoltPe Planning Engine so planning outages and
	// calculation errors are always auditable in the database.
	pendingReport, err := ctrl.service.CreateStructuredJSONReportRequest(
		userID,
		req,
		rawRequestBytes,
	)
	if err != nil {
		slog.Error("failed to store financial planner request before planning call", "error", err, "email", req.Email)
		c.JSON(http.StatusInternalServerError, utils.ErrorResponse(
			"FP_STORAGE_ERROR",
			"Unable to store the financial planner request",
		))
		return
	}

	planningResponse, calculationBytes, err := ctrl.service.GenerateStructuredJSONReport(
		c.Request.Context(),
		req,
	)
	if err != nil {
		status := classifyCalculationFailure(err)
		if updateErr := ctrl.service.MarkStructuredJSONReportFailed(
			pendingReport.ID,
			status,
			err,
			calculationBytes,
		); updateErr != nil {
			slog.Error("failed to persist BoltPe Planning Engine failure", "report_id", pendingReport.ID, "error", updateErr)
		}

		slog.Error(
			"BoltPe Planning Engine structured JSON request failed",
			"error", err,
			"email", req.Email,
			"report_id", pendingReport.ID,
		)

		detail := utils.ClassifyFinancialPlannerError(err)
		c.JSON(detail.HTTPStatus, utils.APIResponse{
			Code:    detail.Code,
			Message: detail.Message,
			Success: false,
			Data:    map[string]interface{}{"report_id": pendingReport.ID, "failed_module": detail.Module},
		})
		return
	}

	if planningResponse == nil {
		emptyErr := errors.New("BoltPe Planning Engine returned an empty response")
		_ = ctrl.service.MarkStructuredJSONReportFailed(
			pendingReport.ID,
			"calculation_invalid_response",
			emptyErr,
			calculationBytes,
		)
		c.JSON(http.StatusInternalServerError, utils.APIResponse{
			Code:    "FP_CALC_001",
			Message: "BoltPe Planning Engine returned an empty result.",
			Success: false,
			Data:    map[string]interface{}{"report_id": pendingReport.ID},
		})
		return
	}

	response := buildStructuredJSONResponse(planningResponse)

	responseBytes, err := json.Marshal(response)
	if err != nil {
		slog.Error(
			"failed to serialize structured financial planner response",
			"error",
			err,
			"email",
			req.Email,
		)
		_ = ctrl.service.MarkStructuredJSONReportFailed(
			pendingReport.ID,
			"response_serialization_failed",
			err,
			calculationBytes,
		)

		c.JSON(
			http.StatusInternalServerError,
			utils.ErrorResponse(
				"FP_RESPONSE_ERROR",
				"Structured report was generated but could not be serialized",
			),
		)
		return
	}

	// Finalize the same row that was created before the planning call.
	storedReport, err := ctrl.service.CompleteStructuredJSONReport(
		pendingReport.ID,
		req,
		planningResponse,
		calculationBytes,
		responseBytes,
	)
	if err != nil {
		slog.Error(
			"failed to store structured financial planner report",
			"error",
			err,
			"email",
			req.Email,
		)

		c.JSON(
			http.StatusInternalServerError,
			utils.ErrorResponse(
				"FP_STORAGE_ERROR",
				"Structured report was generated but could not be stored",
			),
		)
		return
	}

	slog.Info(
		"structured financial planner report generated and stored",
		"email",
		req.Email,
		"report_id",
		storedReport.ID,
		"response_bytes",
		len(responseBytes),
	)

	// Return the structured JSON immediately. PDF generation, S3 upload and
	// email delivery continue independently in the background. A fresh context
	// is used because Gin cancels the request context after this handler returns.
	go func(reportID uint, request utils.FinancialPlannerRequest) {
		timeout := 2 * time.Minute

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := ctrl.service.ProcessStructuredReportPDF(ctx, reportID, request); err != nil {
			slog.Error(
				"background PDF generation or email delivery failed",
				"report_id", reportID,
				"email", request.Email,
				"error", err,
			)
		}
	}(storedReport.ID, req)

	c.Data(
		http.StatusOK,
		"application/json; charset=utf-8",
		responseBytes,
	)
}

func buildStructuredJSONResponse(
	planning *utils.BoltPePlanningResponse,
) models.StructuredJSONReportResponse {
	reports := make(
		[]models.StructuredJSONGoalReport,
		0,
		len(planning.Data.Reports),
	)

	for _, report := range planning.Data.Reports {
		tables := make(
			[]models.StructuredJSONGoalTable,
			0,
			len(report.Table),
		)

		for _, row := range report.Table {
			tables = append(
				tables,
				models.StructuredJSONGoalTable{
					FutureCorpusNeeded: row.FutureCorpusNeeded,
					Goal:               row.Goal,
					GoalType:           row.GoalType,
					LumpsumNeededToday: row.LumpsumNeededToday,

					MonthlySavingRequired: row.
						MonthlySavingRequired,

					YearsMonthsLeft: row.YearsMonthsLeft,
				},
			)
		}

		reports = append(
			reports,
			models.StructuredJSONGoalReport{
				Assumptions: models.StructuredJSONGoalAssumptions{
					ExpectedReturn: report.Assumptions.
						ExpectedReturn,

					InflationUsed: report.Assumptions.
						InflationUsed,

					PlanningStyle: report.Assumptions.
						PlanningStyle,
				},

				Feasibility: models.StructuredJSONGoalFeasibility{
					AIAdvisoryNote: report.Feasibility.
						AIAdvisoryNote,

					AchievableTimeline: report.Feasibility.
						AchievableTimelineWithSavings,

					CurrentMonthlySavings: report.Feasibility.
						CurrentMonthlySavings,

					ExtraTimeRequired: report.Feasibility.
						ExtraTimeRequired,

					RequiredMonthlyIncome: report.Feasibility.
						RequiredMonthlyIncome,

					SelectedTimeline: report.Feasibility.
						SelectedTimeline,

					Title: report.Feasibility.Title,
				},

				Priority: report.Priority,

				Recommendations: nonNilStrings(
					report.Recommendations,
				),

				SubTitle: report.SubTitle,

				Summary: models.StructuredJSONGoalSummary{
					GoalName: report.Summary.GoalName,
					GoalType: report.Summary.GoalType,

					TargetAmount: report.Summary.
						TargetAmount,

					TargetHorizon: report.Summary.
						TargetHorizon,

					Timeline: report.Summary.Timeline,
				},

				SummaryTitle: report.SummaryTitle,

				Table: tables,

				Title: report.Title,

				Type: report.Type,
			},
		)
	}

	return models.StructuredJSONReportResponse{
		APIVersion: "boltpe-json-v1",

		Data: models.StructuredJSONReportData{
			CommonRecommendations: nonNilStrings(
				planning.Data.CommonRecommendations,
			),

			CommonRecommendationsSubtitle: planning.Data.
				CommonRecommendationsSubtitle,

			CommonRecommendationsTitle: planning.Data.
				CommonRecommendationsTitle,

			CompanyContact: nonNilMap(
				planning.Data.CompanyContact,
			),

			Customer: nonNilMap(
				planning.Data.Customer,
			),

			GeneratedAt: planning.Data.GeneratedAt,

			GoalTypeEngine: planning.Data.GoalTypeEngine,

			InsuranceInvestment: nonNilMap(
				planning.Data.InsuranceInvestment,
			),

			PriorityRoadmap: nonNilStrings(
				planning.Data.PriorityRoadmap,
			),

			Reports: reports,

			Score: models.StructuredJSONScore{
				Label: utils.FinancialStatusFromScore(planning.Data.Score.Value),
				Value: planning.Data.Score.Value,
			},

			Summary: nonNilMap(
				planning.Data.Summary,
			),
		},

		Message: "Structured financial planner JSON generated successfully. PDF generation and email delivery started in the background.",

		ResponseType: "structured_json",

		Success: true,
	}
}

func nonNilMap(
	value map[string]interface{},
) map[string]interface{} {
	if value == nil {
		return map[string]interface{}{}
	}

	return value
}

func nonNilStrings(
	value []string,
) []string {
	if value == nil {
		return []string{}
	}

	return value
}

// GeneratePublicReport godoc
//
// @Summary Generate BoltPe financial planner report
// @ID post_financial_planner_report
// @Description Validates the BoltPe request, runs the internal BoltPe Planning Engine, saves the report and returns the BoltPe report response. No Policy Planner/vendor call is used.
// @Tags Reports
// @Accept json
// @Produce json
// // @Security ApiKeyAuth
// @Param request body models.FinancialPlannerRequest true "Financial planner request"
// @Success 200 {object} models.GenerateReportSuccessResponse
// @Failure 400 {object} models.APIErrorResponse
// @Failure 429 {object} models.APIErrorResponse
// @Failure 500 {object} models.APIErrorResponse
// @Router /api/financial-planner/report [post]
func (ctrl *FinancialPlannerController) GeneratePublicReport(
	c *gin.Context,
) {
	var req utils.FinancialPlannerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			utils.ErrorResponse(
				"VAL_001",
				"Invalid financial planner request payload",
			),
		)
		return
	}

	utils.NormalizeFinancialPlannerRequest(&req)

	if err := utils.ValidateFinancialPlannerRequest(req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			utils.ErrorResponse(
				"VAL_001",
				err.Error(),
			),
		)
		return
	}

	// This endpoint does not require X-User-ID.
	// A zero value is passed because user-specific access is required only
	// for report-json creation and the reports listing endpoint.
	const userID uint = 0

	report, planningResponse, err :=
		ctrl.service.GenerateFinancialReport(
			c.Request.Context(),
			userID,
			req,
		)

	if err != nil {
		slog.Error(
			"failed to generate financial planner report",
			"error",
			err,
			"email",
			req.Email,
		)

		detail := utils.ClassifyFinancialPlannerError(err)
		c.JSON(detail.HTTPStatus, utils.ErrorResponse(detail.Code, detail.Message))
		return
	}

	if report == nil || planningResponse == nil {
		c.JSON(
			http.StatusInternalServerError,
			utils.ErrorResponse(
				"FP_CALC_INVALID_RESPONSE",
				"Unable to process the financial planner report",
			),
		)
		return
	}

	response := ctrl.buildGenerateReportResponse(
		req,
		report,
		planningResponse,
	)

	if err := ctrl.service.StoreBoltPeResponse(
		report,
		response,
	); err != nil {
		slog.Error(
			"failed to store BoltPe report response",
			"error",
			err,
			"report_id",
			report.ID,
		)

		c.JSON(
			http.StatusInternalServerError,
			utils.ErrorResponse(
				"FP_STORAGE_ERROR",
				"Financial planner report was generated, but the BoltPe response could not be stored",
			),
		)
		return
	}

	c.JSON(
		http.StatusOK,
		response,
	)
}

func (ctrl *FinancialPlannerController) buildGenerateReportResponse(
	req utils.FinancialPlannerRequest,
	report *financialPlanner.FinancialPlannerReport,
	planningResponse *utils.BoltPePlanningResponse,
) models.GenerateReportSuccessResponse {
	customer := planningResponse.Data.Customer

	fullName := stringFromMap(
		customer,
		"full_name",
	)

	if fullName == "" {
		fullName = strings.TrimSpace(req.FullName)
	}

	email := stringFromMap(
		customer,
		"email",
	)

	if email == "" {
		email = strings.TrimSpace(req.Email)
	}

	age := intFromMap(
		customer,
		"age",
	)

	emailSent := boolFromMap(
		planningResponse.Data.EmailStatus,
		"sent",
	)

	emailMessage := stringFromMap(
		planningResponse.Data.EmailStatus,
		"message",
	)

	if emailMessage == "" {
		if emailSent {
			emailMessage = fmt.Sprintf(
				"Report email sent successfully to %s.",
				email,
			)
		} else {
			emailMessage = "Report email was not sent"
		}
	}

	reportPDFURL := strings.TrimSpace(report.ReportPDFURL)
	if reportPDFURL == "" {
		reportPDFURL = strings.TrimSpace(planningResponse.Data.ReportPDFURL)
	}
	reportPDFPath := strings.TrimSpace(report.ReportPDFPath)
	if reportPDFPath == "" {
		reportPDFPath = strings.TrimSpace(planningResponse.Data.ReportPDFPath)
	}

	return models.GenerateReportSuccessResponse{
		Success: true,

		Message: "Financial planner report generated successfully",

		Data: models.ReportDataResponse{
			ReportID: report.ID,

			Status: normalizeReportStatus(
				report.Status,
			),

			Customer: models.ReportCustomerResponse{
				FullName: fullName,
				Email:    email,
				Age:      age,
			},

			Score: models.ReportScoreResponse{
				Value: planningResponse.Data.Score.Value,
				Label: utils.FinancialStatusFromScore(planningResponse.Data.Score.Value),
			},

			ReportPDFURL:  reportPDFURL,
			ReportPDFPath: reportPDFPath,

			EmailStatus: models.ReportEmailStatusResponse{
				Sent:    emailSent,
				Message: emailMessage,
			},

			PDFStatus: models.ReportPDFStatusResponse{
				Saved: planningResponse.Data.PDFStatus.Saved &&
					reportPDFURL != "",

				ReportPDFURL:  reportPDFURL,
				ReportPDFPath: reportPDFPath,
			},
		},
	}
}

func normalizeBoltPeReportURL(
	reportPDFURL string,
) string {
	reportPDFURL = strings.TrimSpace(reportPDFURL)
	if reportPDFURL == "" {
		return ""
	}

	parsedURL, err := url.Parse(reportPDFURL)
	if err == nil && parsedURL.Path != "" {
		reportPDFURL = parsedURL.Path
	}

	reportPDFURL = strings.TrimPrefix(
		strings.TrimSpace(reportPDFURL),
		"/",
	)

	downloadPrefixes := []string{
		"api/boltpe-financial-planner/reports/",
	}
	for _, downloadPrefix := range downloadPrefixes {
		if strings.HasPrefix(reportPDFURL, downloadPrefix) {
			reportPDFURL = strings.TrimPrefix(reportPDFURL, downloadPrefix)
			break
		}
	}

	segments := strings.Split(reportPDFURL, "/")
	cleanSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" || segment == "." || segment == ".." {
			continue
		}
		cleanSegments = append(cleanSegments, segment)
	}

	if len(cleanSegments) == 0 {
		return ""
	}

	filename := cleanSegments[len(cleanSegments)-1]
	if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		return ""
	}

	if len(cleanSegments) >= 2 &&
		strings.EqualFold(cleanSegments[len(cleanSegments)-2], "publictpap") {
		return "/api/boltpe-financial-planner/reports/publictpap/" +
			url.PathEscape(filename)
	}

	return "/api/boltpe-financial-planner/reports/" + url.PathEscape(filename)
}

func getUserID(
	c *gin.Context,
) (uint, error) {
	value := strings.TrimSpace(
		c.GetHeader("X-User-ID"),
	)

	if value == "" {
		return 0, errors.New(
			"X-User-ID header is required",
		)
	}

	id, err := strconv.ParseUint(
		value,
		10,
		64,
	)

	if err != nil || id == 0 {
		return 0, errors.New(
			"X-User-ID must be a valid positive integer",
		)
	}

	return uint(id), nil
}

func normalizeReportStatus(
	status string,
) string {
	normalized := strings.ToLower(
		strings.TrimSpace(status),
	)

	switch normalized {
	case "",
		"completed",
		"success",
		"successful",
		"generated",
		"report_completed":
		return "report_generated"

	case "processing",
		"requested",
		"report_requested":
		return "report_processing"

	case "failed",
		"error",
		"report_failed":
		return "report_failed"

	default:
		return normalized
	}
}

func stringFromMap(
	values map[string]interface{},
	key string,
) string {
	if values == nil {
		return ""
	}

	value, exists := values[key]
	if !exists || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)

	default:
		return strings.TrimSpace(
			fmt.Sprint(value),
		)
	}
}

func intFromMap(
	values map[string]interface{},
	key string,
) int {
	if values == nil {
		return 0
	}

	value, exists := values[key]
	if !exists || value == nil {
		return 0
	}

	switch typed := value.(type) {
	case int:
		return typed

	case int8:
		return int(typed)

	case int16:
		return int(typed)

	case int32:
		return int(typed)

	case int64:
		return int(typed)

	case uint:
		return int(typed)

	case uint8:
		return int(typed)

	case uint16:
		return int(typed)

	case uint32:
		return int(typed)

	case uint64:
		return int(typed)

	case float32:
		return int(typed)

	case float64:
		return int(typed)

	case json.Number:
		return parseFlexibleInt(
			typed.String(),
		)

	case string:
		return parseFlexibleInt(
			typed,
		)

	default:
		return parseFlexibleInt(
			fmt.Sprint(value),
		)
	}
}

func parseFlexibleInt(
	value string,
) int {
	cleaned := strings.TrimSpace(
		value,
	)

	if cleaned == "" {
		return 0
	}

	integerValue, err := strconv.Atoi(
		cleaned,
	)

	if err == nil {
		return integerValue
	}

	decimalValue, err := strconv.ParseFloat(
		cleaned,
		64,
	)

	if err == nil {
		return int(decimalValue)
	}

	return 0
}

func boolFromMap(
	values map[string]interface{},
	key string,
) bool {
	if values == nil {
		return false
	}

	value, exists := values[key]
	if !exists || value == nil {
		return false
	}

	switch typed := value.(type) {
	case bool:
		return typed

	case string:
		return parseFlexibleBool(
			typed,
		)

	case int:
		return typed != 0

	case int8:
		return typed != 0

	case int16:
		return typed != 0

	case int32:
		return typed != 0

	case int64:
		return typed != 0

	case uint:
		return typed != 0

	case uint8:
		return typed != 0

	case uint16:
		return typed != 0

	case uint32:
		return typed != 0

	case uint64:
		return typed != 0

	case float32:
		return typed != 0

	case float64:
		return typed != 0

	case json.Number:
		return parseFlexibleBool(
			typed.String(),
		)

	default:
		return parseFlexibleBool(
			fmt.Sprint(value),
		)
	}
}

func parseFlexibleBool(
	value string,
) bool {
	normalized := strings.ToLower(
		strings.TrimSpace(value),
	)

	switch normalized {
	case "true",
		"yes",
		"y",
		"1",
		"success",
		"successful",
		"sent",
		"saved",
		"completed",
		"generated",
		"ok":
		return true

	default:
		return false
	}
}
