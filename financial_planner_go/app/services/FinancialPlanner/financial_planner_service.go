package FinancialPlanner

import (
	dao "FinancialPlannerGo/app/domain/dao/financialPlanner"
	repo "FinancialPlannerGo/app/repositories/FinancialPlanner"
	planningengine "FinancialPlannerGo/app/services/FinancialPlanner/engine"
	"FinancialPlannerGo/app/services/FinancialPlanner/mailer"
	pdfgen "FinancialPlannerGo/app/services/FinancialPlanner/pdf"
	"FinancialPlannerGo/app/services/storage"
	"FinancialPlannerGo/app/utils"
	envconfig "FinancialPlannerGo/config/env_config"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
)

type FinancialPlannerService interface {
	GenerateFinancialReport(
		ctx context.Context,
		userID uint,
		req utils.FinancialPlannerRequest,
	) (*dao.FinancialPlannerReport, *utils.BoltPePlanningResponse, error)

	GenerateStructuredJSONReport(
		ctx context.Context,
		req utils.FinancialPlannerRequest,
	) (*utils.BoltPePlanningResponse, []byte, error)

	GenerateBoltPePDFDirect(
		ctx context.Context,
		req utils.FinancialPlannerRequest,
	) ([]byte, string, []byte, error)

	CreateStructuredJSONReportRequest(
		userID uint,
		req utils.FinancialPlannerRequest,
		rawRequestBytes []byte,
	) (*dao.FinancialPlannerReport, error)

	CreatePDFDirectReportRequest(
		userID uint,
		req utils.FinancialPlannerRequest,
		rawRequestBytes []byte,
	) (*dao.FinancialPlannerReport, error)

	CompleteDirectPDFReport(
		ctx context.Context,
		reportID uint,
		rawCalculationBytes []byte,
		pdfData []byte,
		filename string,
	) (*dao.FinancialPlannerReport, error)

	MarkDirectPDFGenerationFailed(
		reportID uint,
		pdfErr error,
		rawCalculationBytes []byte,
	) error

	CompleteStructuredJSONReport(
		reportID uint,
		req utils.FinancialPlannerRequest,
		planning *utils.BoltPePlanningResponse,
		rawCalculationBytes []byte,
		responseBytes []byte,
	) (*dao.FinancialPlannerReport, error)

	MarkStructuredJSONReportFailed(
		reportID uint,
		status string,
		err error,
		rawCalculationBytes []byte,
	) error

	StoreStructuredJSONReport(
		userID uint,
		req utils.FinancialPlannerRequest,
		rawRequestBytes []byte,
		planning *utils.BoltPePlanningResponse,
		rawCalculationBytes []byte,
		responseBytes []byte,
	) (*dao.FinancialPlannerReport, error)

	ProcessStructuredReportPDF(
		ctx context.Context,
		reportID uint,
		req utils.FinancialPlannerRequest,
	) error
	GetReports(
		userID uint,
	) ([]dao.FinancialPlannerReport, error)

	GetGeneratedReportPDF(ctx context.Context, reportReference string) ([]byte, error)

	StoreBoltPeResponse(
		report *dao.FinancialPlannerReport,
		response any,
	) error

	GetSMTPStatus() SMTPStatus
	SendTestEmail(ctx context.Context, toEmail string) error
}

const (
	boltPeCalculationEngine  = "boltpe-planning-engine"
	calculationEngineVersion = "1.0.0"
	assumptionVersion        = "2026.01"
	responseSchemaVersion    = "boltpe-json-v1"
)

type SMTPStatus struct {
	Configured                      bool
	Host                            string
	Port                            int
	User                            string
	PasswordSet                     bool
	PasswordLengthAfterSpaceRemoval int
}

type financialPlannerService struct {
	repo          repo.FinancialPlannerRepository
	reportJSONURL string
	s3Storage     *storage.S3Storage
}

func NewFinancialPlannerService(
	financialPlannerRepo repo.FinancialPlannerRepository,
) FinancialPlannerService {
	if envconfig.AppConfig == nil {
		panic("application configuration is not initialized")
	}

	s3cfg := envconfig.AppConfig.S3
	s3Store, err := storage.NewS3Storage(context.Background(), storage.S3Config{Endpoint: s3cfg.Endpoint, Region: s3cfg.Region, AccessKey: s3cfg.AccessKey, SecretKey: s3cfg.SecretKey, Bucket: s3cfg.BucketName, Folder: s3cfg.PublicFolder})
	if err != nil {
		panic("initialize S3 storage: " + err.Error())
	}

	return &financialPlannerService{
		repo:          financialPlannerRepo,
		reportJSONURL: boltPeCalculationEngine,
		s3Storage:     s3Store,
	}
}

func (s *financialPlannerService) GetSMTPStatus() SMTPStatus {
	mailCfg := envconfig.AppConfig.Mail
	password := strings.ReplaceAll(strings.TrimSpace(mailCfg.Password), " ", "")
	configured := strings.TrimSpace(mailCfg.Host) != "" &&
		mailCfg.Port > 0 &&
		strings.TrimSpace(mailCfg.Username) != "" &&
		strings.TrimSpace(mailCfg.Password) != "" &&
		strings.TrimSpace(mailCfg.FromEmail) != ""

	return SMTPStatus{
		Configured:                      configured,
		Host:                            strings.TrimSpace(mailCfg.Host),
		Port:                            mailCfg.Port,
		User:                            strings.TrimSpace(mailCfg.Username),
		PasswordSet:                     strings.TrimSpace(mailCfg.Password) != "",
		PasswordLengthAfterSpaceRemoval: len(password),
	}
}

func (s *financialPlannerService) SendTestEmail(ctx context.Context, toEmail string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	mailCfg := envconfig.AppConfig.Mail
	sender := mailer.Mailer{
		Config: mailer.Config{
			Host:      mailCfg.Host,
			Port:      mailCfg.Port,
			Username:  mailCfg.Username,
			Password:  mailCfg.Password,
			FromEmail: mailCfg.FromEmail,
			FromName:  mailCfg.FromName,
			UseTLS:    mailCfg.UseTLS,
			UseSSL:    mailCfg.UseSSL,
		},
	}

	return sender.SendTestEmail(strings.TrimSpace(toEmail))
}

func (s *financialPlannerService) GenerateStructuredJSONReport(
	ctx context.Context,
	req utils.FinancialPlannerRequest,
) (*utils.BoltPePlanningResponse, []byte, error) {
	_ = ctx
	response, err := planningengine.Calculate(req)
	if err != nil {
		return nil, nil, fmt.Errorf("calculate BoltPe financial plan: %w", err)
	}
	raw, err := json.Marshal(response)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal BoltPe planning response: %w", err)
	}
	return response, raw, nil
}

func (s *financialPlannerService) GenerateBoltPePDFDirect(
	ctx context.Context,
	req utils.FinancialPlannerRequest,
) ([]byte, string, []byte, error) {
	structuredReport, rawCalculationResponse, err := s.GenerateStructuredJSONReport(ctx, req)
	if err != nil {
		return nil, "", rawCalculationResponse, fmt.Errorf("boltpe planning engine: %w", err)
	}
	// Keep the complete planning response for PDF generation so that table,
	// feasibility, assumptions and recommendation fields are not lost.
	// Only inject BoltPe's target-amount fallback into the raw response.
	structuredResponse, err := mergeTargetAmountsIntoRawResponse(
		rawCalculationResponse,
		structuredReport,
	)
	if err != nil {
		return nil, "", rawCalculationResponse, fmt.Errorf("pdf preparation: %w", err)
	}

	generator := pdfgen.Generator{
		LogoPath: envconfig.AppConfig.Brand.LogoPath,
	}
	pdfData, err := generator.GenerateBytes(structuredResponse)
	if err != nil {
		return nil, "", rawCalculationResponse, fmt.Errorf("pdf generation: %w", err)
	}

	return pdfData, financialPlanningReportFilename(req.FullName), rawCalculationResponse, nil
}
func mergeTargetAmountsIntoRawResponse(
	rawCalculationResponse []byte,
	response *utils.BoltPePlanningResponse,
) ([]byte, error) {
	if len(rawCalculationResponse) == 0 {
		return nil, errors.New("planning response is empty")
	}
	if response == nil {
		return nil, errors.New("normalized planning response is required")
	}
	var root map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(rawCalculationResponse))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode planning response: %w", err)
	}
	data := firstRawMap(root, "data", "result", "response", "payload")
	if data == nil {
		data = root
	}
	reports := firstRawSlice(data, "reports", "goal_reports", "goalReports")
	for index := range reports {
		if index >= len(response.Data.Reports) {
			break
		}
		targetAmount := strings.TrimSpace(
			response.Data.Reports[index].Summary.TargetAmount,
		)
		if targetAmount == "" {
			continue
		}
		report, ok := reports[index].(map[string]interface{})
		if !ok {
			continue
		}
		summary := firstRawMap(report, "summary", "goal_summary", "goalSummary")
		if summary == nil {
			summary = make(map[string]interface{})
			report["summary"] = summary
		}
		setExistingOrDefaultRawValue(
			summary,
			targetAmount,
			"Target Amount / Today's Cost",
			"target_amount",
			"targetAmount",
			"TargetAmount",
		)
	}
	result, err := json.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("encode updated planning response: %w", err)
	}
	return result, nil
}
func firstRawMap(
	data map[string]interface{},
	keys ...string,
) map[string]interface{} {
	for _, key := range keys {
		value, exists := data[key]
		if !exists {
			continue
		}
		mapped, ok := value.(map[string]interface{})
		if ok {
			return mapped
		}
	}
	return nil
}
func firstRawSlice(
	data map[string]interface{},
	keys ...string,
) []interface{} {
	for _, key := range keys {
		value, exists := data[key]
		if !exists {
			continue
		}
		items, ok := value.([]interface{})
		if ok {
			return items
		}
	}
	return nil
}
func setExistingOrDefaultRawValue(
	data map[string]interface{},
	value interface{},
	keys ...string,
) {
	for _, key := range keys {
		if _, exists := data[key]; exists {
			data[key] = value
			return
		}
	}
	if len(keys) > 0 {
		data[keys[0]] = value
	}
}

func (s *financialPlannerService) CreateStructuredJSONReportRequest(
	userID uint,
	req utils.FinancialPlannerRequest,
	rawRequestBytes []byte,
) (*dao.FinancialPlannerReport, error) {
	return s.createReportRequest(userID, req, rawRequestBytes, "calculation_in_progress", "report_json")
}

func (s *financialPlannerService) CreatePDFDirectReportRequest(
	userID uint,
	req utils.FinancialPlannerRequest,
	rawRequestBytes []byte,
) (*dao.FinancialPlannerReport, error) {
	return s.createReportRequest(userID, req, rawRequestBytes, "calculation_in_progress", "report_pdf_direct")
}

func (s *financialPlannerService) createReportRequest(
	userID uint,
	req utils.FinancialPlannerRequest,
	rawRequestBytes []byte,
	initialStatus string,
	requestType string,
) (*dao.FinancialPlannerReport, error) {
	if len(req.SelectedGoals) == 0 {
		return nil, errors.New("at least one selected goal is required")
	}
	if len(rawRequestBytes) == 0 || !json.Valid(rawRequestBytes) {
		return nil, errors.New("raw request payload is not valid JSON")
	}

	goalsBytes, err := json.Marshal(req.SelectedGoals)
	if err != nil {
		return nil, fmt.Errorf("marshal selected goals: %w", err)
	}
	childrenBytes, err := json.Marshal(req.Children)
	if err != nil {
		return nil, fmt.Errorf("marshal children: %w", err)
	}

	report := &dao.FinancialPlannerReport{
		UserID: userID, FullName: req.FullName, Contact: req.Contact, Email: req.Email, DOB: req.DOB, City: req.City,
		MaritalStatus: req.MaritalStatus, Dependents: parseInt(req.Dependents), ChildrenCount: parseInt(req.ChildrenCount),
		MonthlyIncome: parseFloat(req.MonthlyIncome), MonthlyExpenses: parseFloat(req.MonthlyExpenses), MonthlySavings: parseFloat(req.MonthlySavings),
		SpouseIncome: parseFloat(req.SpouseIncome), SpouseAge: parseInt(req.SpouseAge),
		RiskAppetite: req.RiskAppetite, InvestmentExperience: req.InvestmentExperience,
		ExistingLifeTerm: parseBoolean(req.ExistingLifeTerm), ExistingHealth: parseBoolean(req.ExistingHealth),
		ExistingCritical: parseBoolean(req.ExistingCritical), ExistingPersonal: parseBoolean(req.ExistingPersonal),
		ExistingLifeTermCover: parseFloat(req.ExistingLifeTermCover), ExistingHealthCover: parseFloat(req.ExistingHealthCover),
		ExistingCriticalCover: parseFloat(req.ExistingCriticalCover), ExistingPersonalCover: parseFloat(req.ExistingPersonalCover),
		Education1Inflation: parseFloat(req.Education1Inflation), Education1LumpsumReturn: parseFloat(req.Education1LumpsumReturn),
		Education1SIPReturn: parseFloat(req.Education1SIPReturn), Education1TargetAge: parseInt(req.Education1TargetAge),
		Marriage1Inflation: parseFloat(req.Marriage1Inflation), Marriage1LumpsumReturn: parseFloat(req.Marriage1LumpsumReturn),
		Marriage1SIPReturn: parseFloat(req.Marriage1SIPReturn), Marriage1TargetAge: parseInt(req.Marriage1TargetAge),
		RetirementInflation: parseFloat(req.RetirementInflation), RetirementPreReturn: parseFloat(req.RetirementPreReturn),
		RetirementPostReturn: parseFloat(req.RetirementPostReturn), WealthInflation: parseFloat(req.WealthInflation),
		WealthLumpsumReturn: parseFloat(req.WealthLumpsumReturn), WealthSIPReturn: parseFloat(req.WealthSIPReturn),
		GenericGoalInflation: parseFloat(req.GenericGoalInflation), GenericGoalLumpsumReturn: parseFloat(req.GenericGoalLumpsumReturn),
		GenericGoalSIPReturn:     parseFloat(req.GenericGoalSIPReturn),
		Status:                   initialStatus,
		RequestType:              requestType,
		CalculationEngine:        s.reportJSONURL,
		CalculationStatusCode:    0,
		CalculationErrorCode:     "",
		CalculationErrorMessage:  "",
		FailedModule:             "",
		CalculationEngineVersion: calculationEngineVersion,
		AssumptionVersion:        assumptionVersion,
		ResponseSchemaVersion:    responseSchemaVersion,
		PDFStatus:                "not_started",
		StorageStatus:            "request_saved",
		SelectedGoalsJSON:        datatypes.JSON(goalsBytes),
		ChildrenJSON:             datatypes.JSON(childrenBytes),
		RequestPayload:           datatypes.JSON(append([]byte(nil), rawRequestBytes...)),
		CalculationPayload:       datatypes.JSON([]byte(`{}`)), ResponsePayload: datatypes.JSON([]byte(`{}`)),
		EmailStatus:    toJSON(map[string]interface{}{"sent": false, "message": "Report email not attempted", "error": ""}),
		DatabaseStatus: toJSON(map[string]interface{}{"saved": true, "message": "Report request stored before planning processing"}),
		ErrorMessage:   "",
	}

	if err := s.repo.Save(report); err != nil {
		return nil, fmt.Errorf("save financial planner request before planning call: %w", err)
	}
	if report.ID == 0 {
		return nil, errors.New("financial planner request was saved without a generated report ID")
	}
	report.DatabaseStatus = toJSON(map[string]interface{}{
		"id": report.ID, "saved": true, "message": "Report request stored before planning processing",
		"report_pdf_url": "", "report_pdf_path": "",
	})
	if err := s.repo.Update(report); err != nil {
		return nil, fmt.Errorf("update initial database status for report %d: %w", report.ID, err)
	}
	return report, nil
}

func (s *financialPlannerService) CompleteStructuredJSONReport(
	reportID uint,
	req utils.FinancialPlannerRequest,
	planning *utils.BoltPePlanningResponse,
	rawCalculationBytes []byte,
	responseBytes []byte,
) (*dao.FinancialPlannerReport, error) {
	if planning == nil {
		return nil, errors.New("planning response is required")
	}
	if len(rawCalculationBytes) == 0 || !json.Valid(rawCalculationBytes) {
		return nil, errors.New("raw planning response payload is not valid JSON")
	}
	if len(responseBytes) == 0 || !json.Valid(responseBytes) {
		return nil, errors.New("BoltPe response payload is not valid JSON")
	}

	report, err := s.repo.GetByID(reportID)
	if err != nil {
		return nil, fmt.Errorf("load report %d for completion: %w", reportID, err)
	}

	report.ScoreValue = planning.Data.Score.Value
	report.ScoreLabel = utils.FinancialStatusFromScore(planning.Data.Score.Value)
	report.Status = "structured_json_generated"
	report.StorageStatus = "structured_response_saved"
	report.CalculationEngine = s.reportJSONURL
	report.CalculationEngineVersion = calculationEngineVersion
	report.AssumptionVersion = assumptionVersion
	report.ResponseSchemaVersion = responseSchemaVersion
	report.CalculationStatusCode = http.StatusOK
	report.CalculationErrorCode = ""
	report.CalculationErrorMessage = ""
	report.FailedModule = ""
	report.GoalTypeEngine = planning.Data.GoalTypeEngine
	report.GeneratedAt = parseGeneratedAt(planning.Data.GeneratedAt)
	report.CalculationPayload = datatypes.JSON(append([]byte(nil), rawCalculationBytes...))
	report.ResponsePayload = datatypes.JSON(append([]byte(nil), responseBytes...))
	report.CommonRecommendationsTitle = planning.Data.CommonRecommendationsTitle
	report.CommonRecommendationsSubtitle = planning.Data.CommonRecommendationsSubtitle
	report.CommonRecommendations = toJSON(planning.Data.CommonRecommendations)
	report.PriorityRoadmap = toJSON(planning.Data.PriorityRoadmap)
	report.CompanyContact = toJSON(planning.Data.CompanyContact)
	report.Customer = toJSON(planning.Data.Customer)
	report.InsuranceInvestment = toJSON(planning.Data.InsuranceInvestment)
	report.EmailStatus = toJSON(normalizeStoredEmailStatus(planning.Data.EmailStatus))
	report.DatabaseStatus = toJSON(normalizeStoredDatabaseStatus(planning.Data.DatabaseStatus, report.ID, true))
	report.ErrorMessage = ""

	if err := s.repo.Update(report); err != nil {
		return nil, fmt.Errorf("update structured JSON report %d: %w", report.ID, err)
	}

	selectedGoalRows := buildSelectedGoalRows(report.ID, req.SelectedGoals, planning.Data.Reports)
	if len(selectedGoalRows) == 0 {
		report.Status = "goal_save_failed"
		report.ErrorMessage = "selected goal rows could not be generated"
		_ = s.repo.Update(report)
		return nil, errors.New(report.ErrorMessage)
	}
	if err := s.repo.SaveSelectedGoals(selectedGoalRows); err != nil {
		report.Status = "goal_save_failed"
		report.ErrorMessage = err.Error()
		_ = s.repo.Update(report)
		return nil, fmt.Errorf("save financial planner selected goals for report %d: %w", report.ID, err)
	}

	return report, nil
}

func (s *financialPlannerService) CompleteDirectPDFReport(
	ctx context.Context,
	reportID uint,
	rawCalculationBytes []byte,
	pdfData []byte,
	filename string,
) (*dao.FinancialPlannerReport, error) {
	if len(pdfData) == 0 || !bytes.HasPrefix(pdfData, []byte("%PDF-")) {
		return nil, errors.New("generated direct PDF is empty or invalid")
	}

	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "" || filename == "." {
		filename = financialPlanningReportFilename("Customer")
	}

	report, err := s.repo.GetByID(reportID)
	if err != nil {
		return nil, fmt.Errorf("load report %d for PDF completion: %w", reportID, err)
	}

	pdfURL, s3Key, err := s.s3Storage.UploadPDF(ctx, filename, pdfData)
	if err != nil {
		report.Status = "pdf_upload_failed"
		report.PDFStatus = "pdf_generated"
		report.StorageStatus = "s3_upload_failed"
		report.ErrorMessage = err.Error()
		_ = s.repo.Update(report)
		return nil, fmt.Errorf("upload direct PDF to S3: %w", err)
	}

	report.ReportPDFURL = pdfURL
	report.ReportPDFPath = s3Key
	report.Status = "pdf_generated"
	report.PDFStatus = "pdf_generated"
	report.StorageStatus = "s3_uploaded"
	report.CalculationEngine = s.reportJSONURL
	report.CalculationEngineVersion = calculationEngineVersion
	report.AssumptionVersion = assumptionVersion
	report.ResponseSchemaVersion = responseSchemaVersion
	report.CalculationStatusCode = http.StatusOK
	report.CalculationErrorCode = ""
	report.CalculationErrorMessage = ""
	report.FailedModule = ""
	report.ErrorMessage = ""
	if len(rawCalculationBytes) > 0 && json.Valid(rawCalculationBytes) {
		report.CalculationPayload = datatypes.JSON(append([]byte(nil), rawCalculationBytes...))
		report.ResponsePayload = datatypes.JSON(append([]byte(nil), rawCalculationBytes...))
	}
	now := time.Now()
	report.GeneratedAt = &now
	report.DatabaseStatus = toJSON(map[string]interface{}{
		"id":              report.ID,
		"saved":           true,
		"message":         "Direct PDF generated and uploaded to S3 successfully",
		"report_pdf_url":  pdfURL,
		"report_pdf_path": s3Key,
	})
	if err := s.repo.Update(report); err != nil {
		return nil, fmt.Errorf("store direct PDF completion for report %d: %w", reportID, err)
	}
	return report, nil
}

func (s *financialPlannerService) MarkDirectPDFGenerationFailed(
	reportID uint,
	pdfErr error,
	rawCalculationBytes []byte,
) error {
	report, err := s.repo.GetByID(reportID)
	if err != nil {
		return fmt.Errorf("load report %d for PDF failure update: %w", reportID, err)
	}
	errMessage := "Failed to generate financial planning PDF"
	if pdfErr != nil && strings.TrimSpace(pdfErr.Error()) != "" {
		errMessage = pdfErr.Error()
	}
	report.Status = "pdf_generation_failed"
	report.PDFStatus = "pdf_generation_failed"
	report.FailedModule = "pdf_generation"
	report.CalculationEngine = s.reportJSONURL
	report.CalculationStatusCode = http.StatusOK
	report.CalculationErrorCode = "FP_REPORT_001"
	report.CalculationErrorMessage = errMessage
	report.ErrorMessage = errMessage
	if len(rawCalculationBytes) > 0 && json.Valid(rawCalculationBytes) {
		report.CalculationPayload = datatypes.JSON(append([]byte(nil), rawCalculationBytes...))
	}
	report.DatabaseStatus = toJSON(map[string]interface{}{
		"id": report.ID, "saved": true, "message": "PDF generation failure stored successfully",
		"report_pdf_url": report.ReportPDFURL, "report_pdf_path": report.ReportPDFPath,
	})
	if err := s.repo.Update(report); err != nil {
		return fmt.Errorf("store PDF generation failure for report %d: %w", reportID, err)
	}
	return nil
}

func planningFailureDetails(planningErr error) (int, string, string, string) {
	detail := utils.ClassifyFinancialPlannerError(planningErr)
	return detail.HTTPStatus, detail.Code, detail.Message, detail.Module
}

func (s *financialPlannerService) MarkStructuredJSONReportFailed(
	reportID uint,
	status string,
	planningErr error,
	rawCalculationBytes []byte,
) error {
	report, err := s.repo.GetByID(reportID)
	if err != nil {
		return fmt.Errorf("load report %d for failure update: %w", reportID, err)
	}
	if strings.TrimSpace(status) == "" {
		status = "planning_service_error"
	}
	planningHTTPStatus, planningErrorCode, errMessage, failedModule := planningFailureDetails(planningErr)

	payload := map[string]interface{}{
		"success":            false,
		"provider":           "boltpe",
		"operation":          report.RequestType,
		"calculation_engine": s.reportJSONURL,
		"calculation_status": planningHTTPStatus,
		"failed_module":      failedModule,
		"error_code":         planningErrorCode,
		"message":            errMessage,
	}
	if len(rawCalculationBytes) > 0 {
		payload["raw_response"] = string(rawCalculationBytes)
	}
	payloadBytes, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return fmt.Errorf("marshal planning failure payload for report %d: %w", reportID, marshalErr)
	}

	report.Status = status
	report.CalculationEngine = s.reportJSONURL
	report.CalculationStatusCode = planningHTTPStatus
	report.CalculationErrorCode = planningErrorCode
	report.CalculationErrorMessage = errMessage
	report.FailedModule = failedModule
	report.StorageStatus = "calculation_failure_saved"
	report.ErrorMessage = errMessage
	report.CalculationPayload = datatypes.JSON(payloadBytes)
	report.ResponsePayload = datatypes.JSON([]byte(`{}`))
	report.EmailStatus = toJSON(map[string]interface{}{
		"sent": false, "message": "Report email not attempted because report generation failed", "error": "",
	})
	report.DatabaseStatus = toJSON(map[string]interface{}{
		"id": report.ID, "saved": true, "message": "Calculation failure stored successfully",
		"report_pdf_url": report.ReportPDFURL, "report_pdf_path": report.ReportPDFPath,
	})

	if err := s.repo.Update(report); err != nil {
		return fmt.Errorf("store planning failure for report %d: %w", reportID, err)
	}
	return nil
}

func (s *financialPlannerService) StoreStructuredJSONReport(
	userID uint,
	req utils.FinancialPlannerRequest,
	rawRequestBytes []byte,
	planning *utils.BoltPePlanningResponse,
	rawCalculationBytes []byte,
	responseBytes []byte,
) (*dao.FinancialPlannerReport, error) {
	if planning == nil {
		return nil, errors.New("planning response is required")
	}

	if len(responseBytes) == 0 {
		return nil, errors.New("BoltPe response payload is required")
	}

	if !json.Valid(responseBytes) {
		return nil, errors.New("BoltPe response payload is not valid JSON")
	}

	if len(req.SelectedGoals) == 0 {
		return nil, errors.New("at least one selected goal is required")
	}

	if len(rawRequestBytes) == 0 || !json.Valid(rawRequestBytes) {
		return nil, errors.New("raw request payload is not valid JSON")
	}
	if len(rawCalculationBytes) == 0 || !json.Valid(rawCalculationBytes) {
		return nil, errors.New("raw planning response payload is not valid JSON")
	}

	goalsBytes, err := json.Marshal(req.SelectedGoals)
	if err != nil {
		return nil, fmt.Errorf("marshal selected goals: %w", err)
	}

	childrenBytes, err := json.Marshal(req.Children)
	if err != nil {
		return nil, fmt.Errorf("marshal children: %w", err)
	}

	storedResponseBytes := append(
		[]byte(nil),
		responseBytes...,
	)

	report := &dao.FinancialPlannerReport{
		UserID:   userID,
		FullName: req.FullName,
		Contact:  req.Contact,
		Email:    req.Email,
		DOB:      req.DOB,
		City:     req.City,

		MaritalStatus: req.MaritalStatus,
		Dependents:    parseInt(req.Dependents),
		ChildrenCount: parseInt(req.ChildrenCount),

		MonthlyIncome:   parseFloat(req.MonthlyIncome),
		MonthlyExpenses: parseFloat(req.MonthlyExpenses),
		MonthlySavings:  parseFloat(req.MonthlySavings),

		SpouseIncome: parseFloat(req.SpouseIncome),
		SpouseAge:    parseInt(req.SpouseAge),

		RiskAppetite:         req.RiskAppetite,
		InvestmentExperience: req.InvestmentExperience,

		ExistingLifeTerm: parseBoolean(
			req.ExistingLifeTerm,
		),
		ExistingHealth: parseBoolean(
			req.ExistingHealth,
		),
		ExistingCritical: parseBoolean(
			req.ExistingCritical,
		),
		ExistingPersonal: parseBoolean(
			req.ExistingPersonal,
		),

		ExistingLifeTermCover: parseFloat(
			req.ExistingLifeTermCover,
		),
		ExistingHealthCover: parseFloat(
			req.ExistingHealthCover,
		),
		ExistingCriticalCover: parseFloat(
			req.ExistingCriticalCover,
		),
		ExistingPersonalCover: parseFloat(
			req.ExistingPersonalCover,
		),

		Education1Inflation: parseFloat(
			req.Education1Inflation,
		),
		Education1LumpsumReturn: parseFloat(
			req.Education1LumpsumReturn,
		),
		Education1SIPReturn: parseFloat(
			req.Education1SIPReturn,
		),
		Education1TargetAge: parseInt(
			req.Education1TargetAge,
		),

		Marriage1Inflation: parseFloat(
			req.Marriage1Inflation,
		),
		Marriage1LumpsumReturn: parseFloat(
			req.Marriage1LumpsumReturn,
		),
		Marriage1SIPReturn: parseFloat(
			req.Marriage1SIPReturn,
		),
		Marriage1TargetAge: parseInt(
			req.Marriage1TargetAge,
		),

		RetirementInflation: parseFloat(
			req.RetirementInflation,
		),
		RetirementPreReturn: parseFloat(
			req.RetirementPreReturn,
		),
		RetirementPostReturn: parseFloat(
			req.RetirementPostReturn,
		),

		WealthInflation: parseFloat(
			req.WealthInflation,
		),
		WealthLumpsumReturn: parseFloat(
			req.WealthLumpsumReturn,
		),
		WealthSIPReturn: parseFloat(
			req.WealthSIPReturn,
		),

		GenericGoalInflation: parseFloat(
			req.GenericGoalInflation,
		),
		GenericGoalLumpsumReturn: parseFloat(
			req.GenericGoalLumpsumReturn,
		),
		GenericGoalSIPReturn: parseFloat(
			req.GenericGoalSIPReturn,
		),

		ScoreValue: planning.Data.Score.Value,
		ScoreLabel: utils.FinancialStatusFromScore(planning.Data.Score.Value),

		Status:         "structured_json_generated",
		GoalTypeEngine: planning.Data.GoalTypeEngine,
		GeneratedAt:    parseGeneratedAt(planning.Data.GeneratedAt),

		SelectedGoalsJSON: datatypes.JSON(goalsBytes),
		ChildrenJSON:      datatypes.JSON(childrenBytes),

		RequestPayload:     datatypes.JSON(append([]byte(nil), rawRequestBytes...)),
		CalculationPayload: datatypes.JSON(append([]byte(nil), rawCalculationBytes...)),
		ResponsePayload: datatypes.JSON(
			storedResponseBytes,
		),

		CommonRecommendationsTitle: planning.Data.
			CommonRecommendationsTitle,

		CommonRecommendationsSubtitle: planning.Data.
			CommonRecommendationsSubtitle,

		CommonRecommendations: toJSON(
			planning.Data.CommonRecommendations,
		),

		PriorityRoadmap: toJSON(
			planning.Data.PriorityRoadmap,
		),

		CompanyContact: toJSON(
			planning.Data.CompanyContact,
		),

		Customer: toJSON(
			planning.Data.Customer,
		),

		InsuranceInvestment: toJSON(
			planning.Data.InsuranceInvestment,
		),

		// Persist planning email delivery information for the structured JSON endpoint.
		EmailStatus: toJSON(
			normalizeStoredEmailStatus(planning.Data.EmailStatus),
		),

		// This is replaced with the local database result after GORM generates the ID.
		DatabaseStatus: toJSON(
			normalizeStoredDatabaseStatus(planning.Data.DatabaseStatus, 0, false),
		),

		// A successful structured JSON response has no error.
		ErrorMessage: "",
	}

	// First save the main report.
	if err := s.repo.Save(report); err != nil {
		return nil, fmt.Errorf(
			"save structured JSON report: %w",
			err,
		)
	}

	// GORM populates report.ID after successful insertion.
	if report.ID == 0 {
		report.Status = "database_save_failed"
		report.ErrorMessage = "financial planner report was saved without a generated report ID"
		return nil, errors.New(report.ErrorMessage)
	}

	// Store the actual local database result after the primary key is generated.
	report.DatabaseStatus = toJSON(
		normalizeStoredDatabaseStatus(
			planning.Data.DatabaseStatus,
			report.ID,
			true,
		),
	)

	if err := s.repo.Update(report); err != nil {
		return nil, fmt.Errorf(
			"update structured JSON database status for report %d: %w",
			report.ID,
			err,
		)
	}

	// Build one child-table row for every request goal.
	selectedGoalRows := buildSelectedGoalRows(
		report.ID,
		req.SelectedGoals,
		planning.Data.Reports,
	)

	if len(selectedGoalRows) == 0 {
		report.Status = "goal_save_failed"
		report.ErrorMessage = "selected goal rows could not be generated"
		_ = s.repo.Update(report)
		return nil, errors.New(report.ErrorMessage)
	}

	slog.Info(
		"saving financial planner selected goals",
		"report_id",
		report.ID,
		"request_goal_count",
		len(req.SelectedGoals),
		"planning_report_count",
		len(planning.Data.Reports),
		"database_row_count",
		len(selectedGoalRows),
	)

	if err := s.repo.SaveSelectedGoals(
		selectedGoalRows,
	); err != nil {
		report.Status = "goal_save_failed"
		report.ErrorMessage = err.Error()
		_ = s.repo.Update(report)
		return nil, fmt.Errorf(
			"save financial planner selected goals for report %d: %w",
			report.ID,
			err,
		)
	}

	// Final successful state after both parent and selected-goal rows are stored.
	report.Status = "structured_json_generated"
	report.ErrorMessage = ""
	report.DatabaseStatus = toJSON(
		normalizeStoredDatabaseStatus(
			planning.Data.DatabaseStatus,
			report.ID,
			true,
		),
	)
	if err := s.repo.Update(report); err != nil {
		return nil, fmt.Errorf(
			"finalize structured JSON report %d: %w",
			report.ID,
			err,
		)
	}

	slog.Info(
		"financial planner selected goals saved",
		"report_id",
		report.ID,
		"saved_goal_count",
		len(selectedGoalRows),
	)

	return report, nil
}
func normalizeStoredEmailStatus(value map[string]interface{}) map[string]interface{} {
	status := make(map[string]interface{})
	for key, item := range value {
		status[key] = item
	}

	if _, exists := status["sent"]; !exists {
		status["sent"] = false
	}
	if _, exists := status["message"]; !exists || strings.TrimSpace(fmt.Sprint(status["message"])) == "" {
		if parseFlexibleStoredBool(status["sent"]) {
			status["message"] = "Report email sent successfully"
		} else {
			status["message"] = "Report email was not sent by the structured JSON endpoint"
		}
	}
	if _, exists := status["error"]; !exists {
		status["error"] = ""
	}

	return status
}

func normalizeStoredDatabaseStatus(
	planningStatus utils.BoltPeDBStatus,
	reportID uint,
	saved bool,
) map[string]interface{} {
	status := map[string]interface{}{
		"id":              planningStatus.ID,
		"saved":           planningStatus.Saved,
		"report_pdf_url":  planningStatus.ReportPDFURL,
		"report_pdf_path": planningStatus.ReportPDFPath,
		"message":         planningStatus.Message,
	}

	if reportID > 0 {
		status["id"] = reportID
	}
	if saved {
		status["saved"] = true
		if strings.TrimSpace(fmt.Sprint(status["message"])) == "" {
			status["message"] = "Structured financial planner report saved successfully"
		}
	}

	return status
}

func parseFlexibleStoredBool(value interface{}) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		normalized := strings.ToLower(strings.TrimSpace(typed))
		return normalized == "true" || normalized == "yes" || normalized == "1" || normalized == "sent"
	case float64:
		return typed != 0
	case int:
		return typed != 0
	default:
		return false
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *financialPlannerService) ProcessStructuredReportPDF(
	ctx context.Context,
	reportID uint,
	req utils.FinancialPlannerRequest,
) error {

	report, err := s.repo.GetByID(reportID)
	if err != nil {
		return fmt.Errorf(
			"load report %d: %w",
			reportID,
			err,
		)
	}

	report.Status = "pdf_generation_in_progress"
	report.PDFStatus = "pdf_generation_in_progress"
	report.ErrorMessage = ""

	if err := s.repo.Update(report); err != nil {
		return fmt.Errorf(
			"update report %d status: %w",
			reportID,
			err,
		)
	}

	filename := financialPlanningReportFilename(req.FullName)

	generator := pdfgen.Generator{
		LogoPath: envconfig.AppConfig.Brand.LogoPath,
	}

	pdfData, err := generator.GenerateBytes(
		[]byte(report.ResponsePayload),
	)
	if err != nil {
		return s.failBackgroundPDF(
			report,
			"pdf_generation_failed",
			err,
		)
	}

	// ----------------------------------------
	// Upload PDF to S3
	// ----------------------------------------

	pdfURL, s3Key, err := s.s3Storage.UploadPDF(
		ctx,
		filename,
		pdfData,
	)
	if err != nil {
		return s.failBackgroundPDF(
			report,
			"pdf_upload_failed",
			err,
		)
	}

	// ----------------------------------------
	// Save PDF details immediately
	// ----------------------------------------

	report.ReportPDFPath = s3Key
	report.ReportPDFURL = pdfURL
	report.StorageStatus = "s3_uploaded"

	report.Status = "pdf_generated"
	report.PDFStatus = "pdf_generated"
	report.ErrorMessage = ""

	now := time.Now()
	report.GeneratedAt = &now

	// VERY IMPORTANT:
	// database_status is a separate JSON field.
	report.DatabaseStatus = toJSON(map[string]any{
		"id":      report.ID,
		"saved":   true,
		"message": "Structured financial planner report saved successfully",

		"report_pdf_url":  pdfURL,
		"report_pdf_path": s3Key,
	})

	if err := s.repo.Update(report); err != nil {
		return fmt.Errorf(
			"save generated PDF details for report %d: %w",
			report.ID,
			err,
		)
	}

	slog.Info(
		"financial planner PDF generated and uploaded",
		"report_id", report.ID,
		"report_pdf_path", s3Key,
		"report_pdf_url", pdfURL,
	)

	// ----------------------------------------
	// Email skipped
	// ----------------------------------------

	report.EmailStatus = toJSON(map[string]any{
		"sent":     false,
		"accepted": false,
		"provider": "boltpe_smtp",
		"to":       req.Email,
		"message":  "Email sending skipped",
	})

	report.Status = "pdf_generated"
	report.FailedModule = ""
	report.CalculationErrorMessage = ""
	report.ErrorMessage = ""

	report.DatabaseStatus = toJSON(map[string]any{
		"id":      report.ID,
		"saved":   true,
		"message": "Structured financial planner report saved successfully",
		"report_pdf_url":  report.ReportPDFURL,
		"report_pdf_path": report.ReportPDFPath,
	})

	return s.repo.Update(report)
}

func (s *financialPlannerService) failBackgroundPDF(
	report *dao.FinancialPlannerReport,
	status string,
	err error,
) error {
	report.Status = status
	report.PDFStatus = status
	report.FailedModule = "pdf_generation"
	report.CalculationErrorCode = "FP_REPORT_001"
	if status == "pdf_upload_failed" {
		report.FailedModule = "s3_storage"
		report.CalculationErrorCode = "FP_STORAGE_001"
		report.StorageStatus = "pdf_upload_failed"
	}
	report.CalculationErrorMessage = err.Error()
	report.ErrorMessage = err.Error()
	if updateErr := s.repo.Update(report); updateErr != nil {
		return fmt.Errorf("%v; additionally failed to update report status: %w", err, updateErr)
	}
	return err
}

func (s *financialPlannerService) GenerateFinancialReport(
	ctx context.Context,
	userID uint,
	req utils.FinancialPlannerRequest,
) (
	*dao.FinancialPlannerReport,
	*utils.BoltPePlanningResponse,
	error,
) {

	engineRequestBytes, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal BoltPe planning request: %w", err)
	}

	boltPeRequestBytes, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"marshal BoltPe request for audit: %w",
			err,
		)
	}

	selectedGoalsJSON, err := json.Marshal(req.SelectedGoals)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"marshal selected goals: %w",
			err,
		)
	}

	report := &dao.FinancialPlannerReport{
		UserID:   userID,
		FullName: req.FullName,
		Contact:  req.Contact,
		Email:    req.Email,
		DOB:      req.DOB,
		City:     req.City,

		MaritalStatus: req.MaritalStatus,
		Dependents:    parseInt(req.Dependents),
		ChildrenCount: parseInt(req.ChildrenCount),

		MonthlyIncome:   parseFloat(req.MonthlyIncome),
		MonthlyExpenses: parseFloat(req.MonthlyExpenses),
		MonthlySavings:  parseFloat(req.MonthlySavings),

		SpouseIncome: parseFloat(req.SpouseIncome),
		SpouseAge:    parseInt(req.SpouseAge),

		RiskAppetite:         req.RiskAppetite,
		InvestmentExperience: req.InvestmentExperience,

		ExistingLifeTerm: parseBoolean(req.ExistingLifeTerm),
		ExistingHealth:   parseBoolean(req.ExistingHealth),
		ExistingCritical: parseBoolean(req.ExistingCritical),
		ExistingPersonal: parseBoolean(req.ExistingPersonal),

		ExistingLifeTermCover: parseFloat(
			req.ExistingLifeTermCover,
		),
		ExistingHealthCover: parseFloat(
			req.ExistingHealthCover,
		),
		ExistingCriticalCover: parseFloat(
			req.ExistingCriticalCover,
		),
		ExistingPersonalCover: parseFloat(
			req.ExistingPersonalCover,
		),

		Education1Inflation: parseFloat(
			req.Education1Inflation,
		),
		Education1LumpsumReturn: parseFloat(
			req.Education1LumpsumReturn,
		),
		Education1SIPReturn: parseFloat(
			req.Education1SIPReturn,
		),
		Education1TargetAge: parseInt(
			req.Education1TargetAge,
		),

		Marriage1Inflation: parseFloat(
			req.Marriage1Inflation,
		),
		Marriage1LumpsumReturn: parseFloat(
			req.Marriage1LumpsumReturn,
		),
		Marriage1SIPReturn: parseFloat(
			req.Marriage1SIPReturn,
		),
		Marriage1TargetAge: parseInt(
			req.Marriage1TargetAge,
		),

		RetirementInflation: parseFloat(
			req.RetirementInflation,
		),
		RetirementPreReturn: parseFloat(
			req.RetirementPreReturn,
		),
		RetirementPostReturn: parseFloat(
			req.RetirementPostReturn,
		),

		WealthInflation: parseFloat(
			req.WealthInflation,
		),
		WealthLumpsumReturn: parseFloat(
			req.WealthLumpsumReturn,
		),
		WealthSIPReturn: parseFloat(
			req.WealthSIPReturn,
		),

		GenericGoalInflation: parseFloat(
			req.GenericGoalInflation,
		),
		GenericGoalLumpsumReturn: parseFloat(
			req.GenericGoalLumpsumReturn,
		),
		GenericGoalSIPReturn: parseFloat(
			req.GenericGoalSIPReturn,
		),

		Status: "report_requested",

		RequestPayload: datatypes.JSON(boltPeRequestBytes),

		SelectedGoalsJSON: datatypes.JSON(
			selectedGoalsJSON,
		),
	}

	if err := s.repo.Save(report); err != nil {
		return report, nil, fmt.Errorf(
			"save initial financial planner report: %w",
			err,
		)
	}

	apiResp, responseBytes, err := s.GenerateStructuredJSONReport(ctx, req)

	if err != nil {
		report.Status = "report_failed"
		report.ErrorMessage = err.Error()

		if len(responseBytes) > 0 {
			report.CalculationPayload =
				datatypes.JSON(responseBytes)
		}

		_ = s.repo.Update(report)

		return report, nil, err
	}

	report.CalculationPayload =
		datatypes.JSON(responseBytes)

	report.ReportPDFURL =
		apiResp.Data.ReportPDFURL

	report.ReportPDFPath =
		apiResp.Data.ReportPDFPath

	report.ScoreValue =
		apiResp.Data.Score.Value

	report.ScoreLabel =
		utils.FinancialStatusFromScore(apiResp.Data.Score.Value)

	report.GeneratedAt =
		parseGeneratedAt(
			apiResp.Data.GeneratedAt,
		)

	report.GoalTypeEngine =
		apiResp.Data.GoalTypeEngine

	report.CommonRecommendationsTitle =
		apiResp.Data.CommonRecommendationsTitle

	report.CommonRecommendationsSubtitle =
		apiResp.Data.CommonRecommendationsSubtitle

	report.CommonRecommendations =
		toJSON(
			apiResp.Data.CommonRecommendations,
		)

	report.PriorityRoadmap =
		toJSON(
			apiResp.Data.PriorityRoadmap,
		)

	report.CompanyContact =
		toJSON(
			apiResp.Data.CompanyContact,
		)

	report.Customer =
		toJSON(
			apiResp.Data.Customer,
		)

	report.InsuranceInvestment =
		toJSON(
			apiResp.Data.InsuranceInvestment,
		)

	report.EmailStatus =
		toJSON(
			apiResp.Data.EmailStatus,
		)

	report.DatabaseStatus =
		toJSON(
			apiResp.Data.DatabaseStatus,
		)

	if !apiResp.Success {
		report.Status = "report_failed"

		switch {
		case strings.TrimSpace(apiResp.Message) != "":
			report.ErrorMessage =
				apiResp.Message

		case strings.TrimSpace(apiResp.Error) != "":
			report.ErrorMessage =
				apiResp.Error

		default:
			report.ErrorMessage =
				"BoltPe Planning Engine returned unsuccessful response"
		}

		_ = s.repo.Update(report)

		return report,
			apiResp,
			errors.New(report.ErrorMessage)
	}

	s3URL, s3Key, pdfData, filename, err :=
		s.generateAndUploadPDF(
			ctx,
			engineRequestBytes,
			req.FullName,
		)

	if err != nil {
		report.Status = "pdf_upload_failed"
		report.PDFStatus = "pdf_upload_failed"
		report.StorageStatus = "s3_upload_failed"
		report.FailedModule = "s3_storage"
		report.ErrorMessage = err.Error()
		_ = s.repo.Update(report)
		return report, apiResp, err
	}

	applyS3PDFDetails(apiResp, s3URL, s3Key)
	report.ReportPDFURL = s3URL
	report.ReportPDFPath = s3Key
	report.PDFStatus = "pdf_generated"
	report.StorageStatus = "s3_uploaded"
	report.Status = "pdf_generated"
	report.ErrorMessage = ""

	mailCfg := envconfig.AppConfig.Mail
	sender := mailer.Mailer{Config: mailer.Config{
		Host: mailCfg.Host, Port: mailCfg.Port, Username: mailCfg.Username,
		Password: mailCfg.Password, FromEmail: mailCfg.FromEmail,
		FromName: mailCfg.FromName, UseTLS: mailCfg.UseTLS, UseSSL: mailCfg.UseSSL,
	}}
	if err := sender.SendReportData(req.Email, req.FullName, filename, pdfData); err != nil {
		report.Status = "email_failed"
		report.FailedModule = "email_delivery"
		report.ErrorMessage = err.Error()
		report.EmailStatus = toJSON(map[string]any{
			"sent": false, "accepted": false, "to": req.Email,
			"message": "PDF uploaded to S3, but email sending failed", "error": err.Error(),
		})
		apiResp.Data.EmailStatus = map[string]interface{}{
			"sent": false, "message": "PDF uploaded to S3, but email sending failed", "error": err.Error(),
		}
		_ = s.repo.Update(report)
		return report, apiResp, fmt.Errorf("send report email: %w", err)
	}

	report.Status = "email_sent"
	report.FailedModule = ""
	report.EmailStatus = toJSON(map[string]any{
		"sent": true, "accepted": true, "to": req.Email,
		"message": "Report email accepted by SMTP server",
	})
	apiResp.Data.EmailStatus = map[string]interface{}{
		"sent": true, "message": fmt.Sprintf("Report email sent successfully to %s.", req.Email),
	}

	if err := s.repo.Update(report); err != nil {
		return report,
			apiResp,
			fmt.Errorf(
				"update generated report: %w",
				err,
			)
	}

	selectedGoalRows :=
		buildSelectedGoalRows(
			report.ID,
			req.SelectedGoals,
			apiResp.Data.Reports,
		)

	if len(selectedGoalRows) > 0 {
		if err := s.repo.SaveSelectedGoals(
			selectedGoalRows,
		); err != nil {

			report.Status =
				"goal_save_failed"

			report.ErrorMessage =
				err.Error()

			_ = s.repo.Update(report)

			return report,
				apiResp,
				fmt.Errorf(
					"save selected goals: %w",
					err,
				)
		}
	}

	return report, apiResp, nil
}

func (s *financialPlannerService) StoreBoltPeResponse(
	report *dao.FinancialPlannerReport,
	response any,
) error {
	if report == nil {
		return errors.New("financial planner report is required")
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal BoltPe response for audit: %w", err)
	}

	report.ResponsePayload = datatypes.JSON(responseBytes)

	if err := s.repo.Update(report); err != nil {
		return fmt.Errorf("store BoltPe response: %w", err)
	}

	return nil
}

var unsafeFilenameChars = regexp.MustCompile(
	`[^a-zA-Z0-9._-]+`,
)

func (s *financialPlannerService) generateAndUploadPDF(
	ctx context.Context,
	requestBytes []byte,
	fullName string,
) (string, string, []byte, string, error) {
	var req utils.FinancialPlannerRequest
	if err := json.Unmarshal(requestBytes, &req); err != nil {
		return "", "", nil, "", fmt.Errorf("decode BoltPe planning request: %w", err)
	}
	apiResp, structuredResponse, err := s.GenerateStructuredJSONReport(ctx, req)
	if err != nil {
		return "", "", nil, "", fmt.Errorf("generate structured financial planner report: %w", err)
	}
	if apiResp == nil || !apiResp.Success {
		message := "BoltPe Planning Engine returned unsuccessful response"
		if apiResp != nil {
			message = firstNonBlank(apiResp.Message, apiResp.Error, message)
		}
		return "", "", nil, "", errors.New(message)
	}

	generator := pdfgen.Generator{LogoPath: envconfig.AppConfig.Brand.LogoPath}
	pdfData, err := generator.GenerateBytes(structuredResponse)
	if err != nil {
		return "", "", nil, "", fmt.Errorf("generate BoltPe financial planning PDF: %w", err)
	}
	if len(pdfData) == 0 || !bytes.HasPrefix(pdfData, []byte("%PDF-")) {
		return "", "", nil, "", errors.New("generated BoltPe PDF is invalid")
	}

	filename := financialPlanningReportFilename(fullName)
	s3URL, s3Key, err := s.s3Storage.UploadPDF(ctx, filename, pdfData)
	if err != nil {
		return "", "", nil, filename, err
	}
	return s3URL, s3Key, pdfData, filename, nil
}

func financialPlanningReportFilename(fullName string) string {
	name := strings.Trim(
		unsafeFilenameChars.ReplaceAllString(strings.TrimSpace(fullName), "_"),
		"._-",
	)
	if name == "" {
		name = "Customer"
	}

	return fmt.Sprintf(
		"%s_%s_Financial_Planning_Report.pdf",
		time.Now().Format("20060102_150405"),
		name,
	)
}

func applyS3PDFDetails(
	apiResp *utils.BoltPePlanningResponse,
	reportURL string,
	reportPath string,
) {

	apiResp.Data.ReportPDFURL =
		reportURL

	apiResp.Data.ReportPDFPath =
		reportPath

	apiResp.Data.PDFStatus.Saved =
		true

	apiResp.Data.PDFStatus.ReportPDFURL =
		reportURL

	apiResp.Data.PDFStatus.ReportPDFPath =
		reportPath

	if apiResp.Raw == nil {
		apiResp.Raw =
			make(
				map[string]interface{},
			)
	}

	data, ok :=
		apiResp.Raw["data"].(map[string]interface{})

	if !ok {
		data =
			make(
				map[string]interface{},
			)

		apiResp.Raw["data"] =
			data
	}

	data["report_pdf_url"] =
		reportURL

	data["report_pdf_path"] =
		reportPath

	pdfStatus, ok :=
		data["pdf_status"].(map[string]interface{})

	if !ok {
		pdfStatus =
			make(
				map[string]interface{},
			)

		data["pdf_status"] =
			pdfStatus
	}

	pdfStatus["saved"] =
		true

	pdfStatus["report_pdf_url"] =
		reportURL

	pdfStatus["report_pdf_path"] =
		reportPath
}

func (s *financialPlannerService) GetReports(
	userID uint,
) ([]dao.FinancialPlannerReport, error) {
	reports, err := s.repo.GetReports(userID)
	if err != nil {
		return nil, fmt.Errorf("fetch financial planner reports: %w", err)
	}

	return reports, nil
}

func (s *financialPlannerService) GetGeneratedReportPDF(
	ctx context.Context,
	reportReference string,
) ([]byte, error) {
	reportReference = strings.TrimPrefix(
		strings.TrimSpace(reportReference),
		"/",
	)

	if reportReference == "" ||
		!strings.HasSuffix(strings.ToLower(reportReference), ".pdf") {
		return nil, errors.New("invalid report path")
	}

	cleanReference := filepath.ToSlash(filepath.Clean(reportReference))
	if cleanReference == "." ||
		cleanReference == ".." ||
		strings.HasPrefix(cleanReference, "../") ||
		strings.Contains(cleanReference, "/../") {
		return nil, errors.New("invalid report path")
	}

	configuredFolder := strings.Trim(
		envconfig.AppConfig.S3.PublicFolder,
		"/",
	)
	if configuredFolder != "" &&
		!strings.HasPrefix(cleanReference, configuredFolder+"/") {
		return nil, errors.New("invalid report folder")
	}

	pdfData, err := s.s3Storage.DownloadPDF(ctx, cleanReference)
	if err != nil {
		return nil, err
	}

	if len(pdfData) == 0 {
		return nil, errors.New("empty PDF received from S3")
	}

	return pdfData, nil
}

func buildSelectedGoalRows(
	reportID uint,
	requestGoals []utils.SelectedGoal,
	apiReports []utils.GoalReport,
) []dao.FinancialPlannerSelectedGoal {
	if reportID == 0 || len(requestGoals) == 0 {
		return []dao.FinancialPlannerSelectedGoal{}
	}

	rows := make(
		[]dao.FinancialPlannerSelectedGoal,
		0,
		len(requestGoals),
	)

	usedPlanningReports := make(
		map[int]bool,
	)

	for goalIndex, requestGoal := range requestGoals {
		apiReport, reportFound := findMatchingPlanningReport(
			requestGoal,
			apiReports,
			usedPlanningReports,
		)

		row := dao.FinancialPlannerSelectedGoal{
			ReportID: reportID,

			GoalKey:   strings.TrimSpace(requestGoal.Key),
			GoalLabel: strings.TrimSpace(requestGoal.Label),
			GoalType: strings.ToLower(
				strings.TrimSpace(requestGoal.Type),
			),

			CategoryTitle: strings.TrimSpace(
				requestGoal.CategoryTitle,
			),
			CustomName: strings.TrimSpace(
				requestGoal.CustomName,
			),

			Amount: requestGoal.Amount,
			Years:  requestGoal.Years,
			Months: requestGoal.Months,

			IsRetirement: requestGoal.Retirement,
			IsAnnualBudget: requestGoal.
				AnnualBudget,

			GoalSequence: goalIndex + 1,

			// Always store valid JSON, even when planning report
			// matching is unavailable.
			Summary:         toJSON(map[string]interface{}{}),
			Recommendations: toJSON([]string{}),
			Assumptions:     toJSON(map[string]interface{}{}),
			Feasibility:     toJSON(map[string]interface{}{}),
			GoalTable:       toJSON([]interface{}{}),
		}

		// If a matching planning report is available, enrich the row.
		if reportFound {
			firstTable := utils.GoalTable{}

			if len(apiReport.Table) > 0 {
				firstTable = apiReport.Table[0]
			}

			row.Priority = apiReport.Priority
			row.SubTitle = apiReport.SubTitle
			row.Title = apiReport.Title
			row.Type = apiReport.Type

			row.SummaryTitle = apiReport.SummaryTitle

			row.Summary = toJSON(
				apiReport.Summary,
			)
			row.Recommendations = toJSON(
				apiReport.Recommendations,
			)
			row.Assumptions = toJSON(
				apiReport.Assumptions,
			)
			row.Feasibility = toJSON(
				apiReport.Feasibility,
			)
			row.GoalTable = toJSON(
				apiReport.Table,
			)

			row.Timeline = apiReport.Summary.Timeline
			row.TargetAmount = apiReport.Summary.TargetAmount
			row.TargetHorizon = apiReport.Summary.TargetHorizon

			row.FutureCorpusNeeded =
				firstTable.FutureCorpusNeeded

			row.LumpsumNeededToday =
				firstTable.LumpsumNeededToday

			row.MonthlySIPRequired =
				firstTable.MonthlySavingRequired

			row.YearsMonthsLeft =
				firstTable.YearsMonthsLeft

			row.SelectedTimeline =
				apiReport.Feasibility.SelectedTimeline

			row.AchievableTimeline =
				apiReport.Feasibility.
					AchievableTimelineWithSavings

			row.ExtraTimeRequired =
				apiReport.Feasibility.ExtraTimeRequired

			row.CurrentMonthlySavings =
				apiReport.Feasibility.CurrentMonthlySavings

			row.RequiredMonthlyIncome =
				apiReport.Feasibility.RequiredMonthlyIncome

			row.AIAdvisoryNote =
				apiReport.Feasibility.AIAdvisoryNote

			row.ExpectedReturn =
				apiReport.Assumptions.ExpectedReturn

			row.InflationUsed =
				apiReport.Assumptions.InflationUsed

			row.PlanningStyle =
				apiReport.Assumptions.PlanningStyle

			row.GoalName =
				apiReport.Summary.GoalName

			row.GoalTypeDisplay =
				apiReport.Summary.GoalType

			row.TargetAmountToday =
				apiReport.Summary.TargetAmount
		}

		// Request values are always used as fallback.
		if strings.TrimSpace(row.GoalLabel) == "" {
			row.GoalLabel = requestGoal.Key
		}

		if strings.TrimSpace(row.SubTitle) == "" {
			row.SubTitle = row.GoalLabel
		}

		if strings.TrimSpace(row.GoalName) == "" {
			row.GoalName = row.GoalLabel
		}

		if strings.TrimSpace(row.GoalTypeDisplay) == "" {
			row.GoalTypeDisplay = row.GoalType
		}

		if strings.TrimSpace(row.Type) == "" {

			row.Type = row.GoalKey

		}

		// Use the amount received in the BoltPe request when the planning

		// does not provide Target Amount / Today's Cost.

		if strings.TrimSpace(row.TargetAmount) == "" {

			row.TargetAmount = formatGoalAmount(requestGoal.Amount)

		}

		if strings.TrimSpace(row.TargetAmountToday) == "" {

			row.TargetAmountToday = row.TargetAmount

		}

		rows = append(rows, row)
	}

	return rows
}
func findMatchingPlanningReport(
	requestGoal utils.SelectedGoal,
	apiReports []utils.GoalReport,
	usedReports map[int]bool,
) (utils.GoalReport, bool) {
	requestKey := normalizeKey(
		requestGoal.Key,
	)

	requestLabel := normalizeKey(
		requestGoal.Label,
	)

	requestCustomName := normalizeKey(
		requestGoal.CustomName,
	)

	// First try exact or normalized matching.
	for index, report := range apiReports {
		if usedReports[index] {
			continue
		}

		reportType := normalizeKey(
			report.Type,
		)

		reportSubTitle := normalizeKey(
			report.SubTitle,
		)

		reportTitle := normalizeKey(
			report.Title,
		)

		reportGoalName := normalizeKey(
			report.Summary.GoalName,
		)

		if normalizedValuesMatch(
			requestKey,
			reportType,
			reportSubTitle,
			reportTitle,
			reportGoalName,
		) ||
			normalizedValuesMatch(
				requestLabel,
				reportType,
				reportSubTitle,
				reportTitle,
				reportGoalName,
			) ||
			normalizedValuesMatch(
				requestCustomName,
				reportType,
				reportSubTitle,
				reportTitle,
				reportGoalName,
			) {
			usedReports[index] = true
			return report, true
		}
	}

	// Fallback: use the first planning report that has not
	// already been assigned.
	for index, report := range apiReports {
		if usedReports[index] {
			continue
		}

		usedReports[index] = true
		return report, true
	}

	return utils.GoalReport{}, false
}

func normalizedValuesMatch(
	requestValue string,
	reportValues ...string,
) bool {
	if requestValue == "" {
		return false
	}

	for _, reportValue := range reportValues {
		if reportValue == "" {
			continue
		}

		if requestValue == reportValue {
			return true
		}

		if strings.Contains(
			requestValue,
			reportValue,
		) ||
			strings.Contains(
				reportValue,
				requestValue,
			) {
			return true
		}
	}

	return false
}
func findMatchingRequestGoal(
	subTitle string,
	title string,
	apiType string,
	byLabel map[string]utils.SelectedGoal,
	byKey map[string]utils.SelectedGoal,
) utils.SelectedGoal {

	keys :=
		[]string{
			normalizeKey(subTitle),
			normalizeKey(title),
			normalizeKey(apiType),

			strings.ReplaceAll(
				normalizeKey(apiType),
				"-",
				"_",
			),
		}

	for _, key := range keys {

		if key == "" {
			continue
		}

		if goal, ok :=
			byLabel[key]; ok {
			return goal
		}

		if goal, ok :=
			byKey[key]; ok {
			return goal
		}
	}

	return utils.SelectedGoal{}
}

func normalizeKey(
	value string,
) string {

	value =
		strings.ToLower(
			strings.TrimSpace(
				value,
			),
		)

	value =
		strings.ReplaceAll(
			value,
			"-",
			"_",
		)

	value =
		strings.ReplaceAll(
			value,
			" ",
			"_",
		)

	value =
		strings.ReplaceAll(
			value,
			"planning",
			"",
		)

	value =
		strings.ReplaceAll(
			value,
			"financial",
			"",
		)

	value =
		strings.ReplaceAll(
			value,
			"report",
			"",
		)

	value =
		strings.ReplaceAll(
			value,
			"chart",
			"",
		)

	for strings.Contains(
		value,
		"__",
	) {
		value =
			strings.ReplaceAll(
				value,
				"__",
				"_",
			)
	}

	return strings.Trim(
		value,
		"_",
	)
}

func toJSON(
	value interface{},
) datatypes.JSON {

	if value == nil {
		return datatypes.JSON(
			[]byte("null"),
		)
	}

	bytesValue, err :=
		json.Marshal(value)

	if err != nil {
		return datatypes.JSON(
			[]byte("null"),
		)
	}

	return datatypes.JSON(
		bytesValue,
	)
}

func parseInt(
	value string,
) int {

	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return 0
	}

	parsed, err :=
		strconv.Atoi(value)

	if err != nil {
		return 0
	}

	return parsed
}

func parseFloat(
	value string,
) float64 {

	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return 0
	}

	parsed, err :=
		strconv.ParseFloat(
			value,
			64,
		)

	if err != nil {
		return 0
	}

	return parsed
}

func parseBoolean(
	value interface{},
) bool {

	str := strings.TrimSpace(
		fmt.Sprint(value),
	)

	switch strings.ToLower(str) {

	case "true",
		"1",
		"yes",
		"y":

		return true

	default:
		return false
	}
}

func parseGeneratedAt(
	value string,
) *time.Time {

	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return nil
	}

	layouts :=
		[]string{
			"02/01/2006 03:04 PM",
			"02/01/2006 15:04:05",
			time.RFC3339,
			"2006-01-02 15:04:05",
		}

	for _, layout := range layouts {

		parsed, err :=
			time.ParseInLocation(
				layout,
				value,
				time.Local,
			)

		if err == nil {
			return &parsed
		}
	}

	return nil
}

func formatGoalAmount(amount float64) string {
	if amount <= 0 {
		return ""
	}

	rounded := math.Round(amount)
	number := strconv.FormatInt(int64(rounded), 10)

	if len(number) <= 3 {
		return "₹" + number
	}

	lastThree := number[len(number)-3:]
	remaining := number[:len(number)-3]

	var groups []string

	for len(remaining) > 2 {
		groups = append(
			[]string{remaining[len(remaining)-2:]},
			groups...,
		)
		remaining = remaining[:len(remaining)-2]
	}

	if remaining != "" {
		groups = append([]string{remaining}, groups...)
	}

	return "₹" + strings.Join(groups, ",") + "," + lastThree
}
