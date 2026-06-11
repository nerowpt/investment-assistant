// Package api 实现 H8 HTTP API 层（复用 internal/core，供 uni-app 前端调用）。
package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// Envelope 统一 JSON 响应信封（对齐 04 §19.5：{code, data, success}）。
type Envelope struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// 业务错误码（HTTP 状态与 code 分离，便于前端统一处理）。
const (
	CodeOK           = 0
	CodeBadRequest   = 40000
	CodeNotFound     = 40400
	CodeConflict     = 40900
	CodeDBError      = 50001
	CodeReadError    = 50002
	CodeInitFailed   = 50003
	CodeInvalidJSON  = 40001
	CodeDraftFailed  = 40010
	CodeSubmitFailed = 40011
	CodeApproveFailed = 40012
)

var reasonToCode = map[string]int{
	"invalid_account":   CodeBadRequest,
	"invalid_json":      CodeInvalidJSON,
	"missing_type":      CodeBadRequest,
	"missing_field":     CodeBadRequest,
	"invalid_type":      CodeBadRequest,
	"invalid_payload":   CodeBadRequest,
	"invalid_values":    CodeBadRequest,
	"not_found":         CodeNotFound,
	"db_error":          CodeDBError,
	"read_error":        CodeReadError,
	"init_failed":       CodeInitFailed,
	"draft_failed":      CodeDraftFailed,
	"preview_failed":    CodeSubmitFailed,
	"submit_failed":     CodeSubmitFailed,
	"plan_failed":       CodeApproveFailed,
	"approve_failed":    CodeApproveFailed,
	"reject_failed":     CodeBadRequest,
	"risk_check_failed": CodeBadRequest,
}

func bizCode(reason string, httpStatus int) int {
	if c, ok := reasonToCode[reason]; ok {
		return c
	}
	if httpStatus >= 400 {
		return httpStatus * 100
	}
	return CodeBadRequest
}

// WriteJSON 写入成功响应。
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Code:    CodeOK,
		Success: true,
		Data:    data,
	})
}

// WriteError 写入错误响应并记录服务端日志。
func WriteError(w http.ResponseWriter, httpStatus int, reason, message string) {
	code := bizCode(reason, httpStatus)
	log.Printf("[api] ERROR http=%d code=%d reason=%s msg=%s", httpStatus, code, reason, message)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(Envelope{
		Code:    code,
		Success: false,
		Message: message,
		Data:    nil,
	})
}

// DecodeJSON 解析请求体。
func DecodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
