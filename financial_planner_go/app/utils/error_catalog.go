package utils

import (
	"errors"
	"net/http"
	"strings"
)

const (
	ErrCodeValidation          = "VAL_001"
	ErrCodeCalculation         = "FP_CALC_001"
	ErrCodeCalculationResponse = "FP_CALC_INVALID_RESPONSE"
	ErrCodeResponse            = "FP_RESPONSE_ERROR"
	ErrCodeStorage             = "FP_STORAGE_ERROR"
	ErrCodePDF                 = "FP_REPORT_001"
	ErrCodeS3                  = "FP_STORAGE_001"
	ErrCodeEmail               = "FP_EMAIL_001"
	ErrCodeList                = "FP_LIST_500"
	ErrCodeNotFound            = "FP_404"
)

type ErrorDetail struct {
	HTTPStatus int
	Code       string
	Message    string
	Module     string
}

func ClassifyFinancialPlannerError(err error) ErrorDetail {
	if err == nil {
		return ErrorDetail{HTTPStatus: http.StatusInternalServerError, Code: ErrCodeCalculation, Message: "Financial planner operation failed", Module: "boltpe_planning_engine"}
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		msg = "Financial planner operation failed"
	}
	lower := strings.ToLower(msg)

	detail := ErrorDetail{HTTPStatus: http.StatusInternalServerError, Code: ErrCodeCalculation, Message: msg, Module: "boltpe_planning_engine"}

	switch {
	case strings.Contains(lower, "validation") || strings.Contains(lower, "selected_goals_json") || strings.Contains(lower, "monthly_") || strings.Contains(lower, "marital_status") || strings.Contains(lower, "spouse_") || strings.Contains(lower, "children_count"):
		detail.HTTPStatus = http.StatusBadRequest
		detail.Code = ErrCodeValidation
		detail.Module = "input_validation"
	case strings.Contains(lower, "goal"):
		detail.HTTPStatus = http.StatusBadRequest
		detail.Code = ErrCodeValidation
		detail.Module = "goal_validation"
	case strings.Contains(lower, "pdf") || strings.Contains(lower, "render") || strings.Contains(lower, "report filename"):
		detail.Code = ErrCodePDF
		detail.Module = "pdf_generation"
	case strings.Contains(lower, "s3") || strings.Contains(lower, "upload") || strings.Contains(lower, "object storage"):
		detail.Code = ErrCodeS3
		detail.Module = "s3_storage"
	case strings.Contains(lower, "email") || strings.Contains(lower, "smtp"):
		detail.Code = ErrCodeEmail
		detail.Module = "email_delivery"
	case strings.Contains(lower, "serialize") || strings.Contains(lower, "marshal"):
		detail.Code = ErrCodeResponse
		detail.Module = "response_serialization"
	case strings.Contains(lower, "database") || strings.Contains(lower, "store") || strings.Contains(lower, "repository"):
		detail.Code = ErrCodeStorage
		detail.Module = "database"
	case strings.Contains(lower, "feasibility"):
		detail.Module = "feasibility_engine"
	case strings.Contains(lower, "score"):
		detail.Module = "score_engine"
	case strings.Contains(lower, "insurance"):
		detail.Module = "insurance_engine"
	case strings.Contains(lower, "allocation") || strings.Contains(lower, "investment"):
		detail.Module = "investment_allocation_engine"
	case strings.Contains(lower, "recommend"):
		detail.Module = "recommendation_engine"
	case strings.Contains(lower, "profile") || strings.Contains(lower, "cashflow"):
		detail.Module = "profile_cashflow_engine"
	}

	return detail
}

func IsValidationError(err error) bool {
	return ClassifyFinancialPlannerError(err).HTTPStatus == http.StatusBadRequest
}

func ErrorResponseFromError(err error) (int, APIResponse) {
	detail := ClassifyFinancialPlannerError(err)
	return detail.HTTPStatus, ErrorResponse(detail.Code, detail.Message)
}

func WrapError(prefix string, err error) error {
	if err == nil {
		return nil
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return err
	}
	return errors.New(prefix + ": " + err.Error())
}
