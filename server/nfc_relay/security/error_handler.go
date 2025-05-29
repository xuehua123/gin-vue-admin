package security

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	"go.uber.org/zap"
)

// ErrorLevel 定义错误严重程度
type ErrorLevel int

const (
	ErrorLevelDebug ErrorLevel = iota
	ErrorLevelInfo
	ErrorLevelWarn
	ErrorLevelError
	ErrorLevelCritical
)

// SecurityError 安全相关错误结构
type SecurityError struct {
	Code       int        `json:"code"`
	Message    string     `json:"message"`
	UserMsg    string     `json:"userMessage"` // 用户友好的错误信息
	Level      ErrorLevel `json:"level"`
	Timestamp  time.Time  `json:"timestamp"`
	Component  string     `json:"component"`
	ClientID   string     `json:"clientId,omitempty"`
	UserID     string     `json:"userId,omitempty"`
	RemoteAddr string     `json:"remoteAddr,omitempty"`
	Cause      error      `json:"-"` // 原始错误，不序列化
}

// Error 实现error接口
func (se *SecurityError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", se.Component, se.Message, se.UserMsg)
}

// Unwrap 实现错误链
func (se *SecurityError) Unwrap() error {
	return se.Cause
}

// SecureErrorHandler 安全错误处理器
type SecureErrorHandler struct {
	enableDetailedErrors bool
	maxErrorLogSize      int
}

// NewSecureErrorHandler 创建安全错误处理器
func NewSecureErrorHandler() *SecureErrorHandler {
	return &SecureErrorHandler{
		enableDetailedErrors: global.GVA_CONFIG.Zap.Level == "debug", // 使用debug级别判断
		maxErrorLogSize:      1024,                                   // 限制错误日志大小
	}
}

// HandleError 处理错误并返回安全的错误信息
func (h *SecureErrorHandler) HandleError(err error, component string, clientContext ...map[string]string) *SecurityError {
	if err == nil {
		return nil
	}

	// 获取调用栈信息
	_, file, line, _ := runtime.Caller(1)
	location := fmt.Sprintf("%s:%d", file, line)

	// 创建安全错误
	secErr := &SecurityError{
		Timestamp: time.Now(),
		Component: component,
		Cause:     err,
	}

	// 从上下文中提取客户端信息
	if len(clientContext) > 0 {
		ctx := clientContext[0]
		secErr.ClientID = ctx["clientId"]
		secErr.UserID = ctx["userId"]
		secErr.RemoteAddr = ctx["remoteAddr"]
	}

	// 根据错误类型进行分类处理
	h.categorizeError(secErr, err)

	// 记录审计日志
	h.logSecurityEvent(secErr, location)

	return secErr
}

// categorizeError 对错误进行分类
func (h *SecureErrorHandler) categorizeError(secErr *SecurityError, err error) {
	errMsg := strings.ToLower(err.Error())

	switch {
	// 认证相关错误
	case strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "token"):
		secErr.Code = protocol.ErrorCodeAuthFailed
		secErr.Level = ErrorLevelWarn
		secErr.Message = "Authentication failed"
		secErr.UserMsg = "身份验证失败，请重新登录"

	// 权限相关错误
	case strings.Contains(errMsg, "permission") || strings.Contains(errMsg, "unauthorized"):
		secErr.Code = protocol.ErrorCodePermissionDenied
		secErr.Level = ErrorLevelWarn
		secErr.Message = "Permission denied"
		secErr.UserMsg = "权限不足，无法执行此操作"

	// 网络相关错误
	case strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "network"):
		secErr.Code = protocol.ErrorCodeServiceUnavailable
		secErr.Level = ErrorLevelError
		secErr.Message = "Network error"
		secErr.UserMsg = "网络连接异常，请检查网络设置"

	// 输入验证错误
	case strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "malformed"):
		secErr.Code = protocol.ErrorCodeBadRequest
		secErr.Level = ErrorLevelInfo
		secErr.Message = "Input validation failed"
		secErr.UserMsg = "输入格式不正确，请检查后重试"

	// 资源冲突错误
	case strings.Contains(errMsg, "busy") || strings.Contains(errMsg, "conflict"):
		secErr.Code = protocol.ErrorCodeConflict
		secErr.Level = ErrorLevelInfo
		secErr.Message = "Resource conflict"
		secErr.UserMsg = "资源正忙，请稍后重试"

	// 系统内部错误
	default:
		secErr.Code = protocol.ErrorCodeInternalError
		secErr.Level = ErrorLevelError
		secErr.Message = "Internal system error"
		if h.enableDetailedErrors {
			secErr.UserMsg = fmt.Sprintf("系统错误: %s", h.sanitizeErrorMessage(err.Error()))
		} else {
			secErr.UserMsg = "系统内部错误，请联系管理员"
		}
	}
}

// sanitizeErrorMessage 清理错误信息，移除敏感信息
func (h *SecureErrorHandler) sanitizeErrorMessage(msg string) string {
	// 限制错误信息长度
	if len(msg) > h.maxErrorLogSize {
		msg = msg[:h.maxErrorLogSize] + "..."
	}

	// 移除可能的敏感信息
	sensitivePatterns := []string{
		`password=\w+`,
		`token=[\w\-\.]+`,
		`key=[\w\-\.]+`,
		`secret=\w+`,
		`\b\d{16,19}\b`, // 信用卡号模式
	}

	for _, pattern := range sensitivePatterns {
		msg = strings.ReplaceAll(msg, pattern, "[REDACTED]")
	}

	return msg
}

// logSecurityEvent 记录安全事件日志
func (h *SecureErrorHandler) logSecurityEvent(secErr *SecurityError, location string) {
	fields := []zap.Field{
		zap.String("component", secErr.Component),
		zap.Int("errorCode", secErr.Code),
		zap.String("level", h.levelToString(secErr.Level)),
		zap.String("location", location),
		zap.String("userMessage", secErr.UserMsg),
	}

	// 添加客户端上下文信息
	if secErr.ClientID != "" {
		fields = append(fields, zap.String("clientId", secErr.ClientID))
	}
	if secErr.UserID != "" {
		fields = append(fields, zap.String("userId", secErr.UserID))
	}
	if secErr.RemoteAddr != "" {
		fields = append(fields, zap.String("remoteAddr", secErr.RemoteAddr))
	}

	// 根据错误级别选择日志级别
	switch secErr.Level {
	case ErrorLevelDebug:
		global.GVA_LOG.Debug(secErr.Message, fields...)
	case ErrorLevelInfo:
		global.GVA_LOG.Info(secErr.Message, fields...)
	case ErrorLevelWarn:
		global.GVA_LOG.Warn(secErr.Message, fields...)
	case ErrorLevelError:
		global.GVA_LOG.Error(secErr.Message, fields...)
	case ErrorLevelCritical:
		global.GVA_LOG.Error(secErr.Message, fields...)
		// 关键错误可以触发告警
		h.triggerAlert(secErr)
	}

	// 记录审计日志
	global.LogAuditEvent(
		"security_error",
		global.ErrorDetails{
			ErrorCode:    fmt.Sprintf("%d", secErr.Code),
			ErrorMessage: secErr.UserMsg,
			Component:    secErr.Component,
		},
		fields...,
	)
}

// levelToString 将错误级别转换为字符串
func (h *SecureErrorHandler) levelToString(level ErrorLevel) string {
	switch level {
	case ErrorLevelDebug:
		return "debug"
	case ErrorLevelInfo:
		return "info"
	case ErrorLevelWarn:
		return "warn"
	case ErrorLevelError:
		return "error"
	case ErrorLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// triggerAlert 触发告警通知
func (h *SecureErrorHandler) triggerAlert(secErr *SecurityError) {
	// 这里可以集成告警系统，如发送邮件、短信、钉钉等
	global.GVA_LOG.Error("🚨 关键安全错误告警",
		zap.String("component", secErr.Component),
		zap.String("message", secErr.Message),
		zap.String("clientId", secErr.ClientID),
		zap.String("userId", secErr.UserID),
	)
}

// 常用错误创建函数
func NewAuthenticationError(err error, clientContext map[string]string) *SecurityError {
	return NewSecureErrorHandler().HandleError(
		errors.New("authentication failed: "+err.Error()),
		"authentication",
		clientContext,
	)
}

func NewPermissionError(operation string, clientContext map[string]string) *SecurityError {
	return NewSecureErrorHandler().HandleError(
		fmt.Errorf("permission denied for operation: %s", operation),
		"authorization",
		clientContext,
	)
}

func NewValidationError(field string, value interface{}) *SecurityError {
	return NewSecureErrorHandler().HandleError(
		fmt.Errorf("validation failed for field %s with value %v", field, value),
		"input_validation",
	)
}
