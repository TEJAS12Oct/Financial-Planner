package utils

type APIResponse struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
}

func SuccessResponse(message string, data any) APIResponse {
	return APIResponse{Message: message, Success: true, Data: data}
}

func ErrorResponse(code string, message string) APIResponse {
	return APIResponse{Code: code, Message: message, Success: false}
}
