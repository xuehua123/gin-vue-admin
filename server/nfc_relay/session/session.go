package session

import (
	"fmt"
	"sync"
	"time"
)

// ClientInfoProvider 定义了 Session 与 Client 交互所需的接口。
// 这样可以避免 session 包直接依赖 handler 包。
// handler.Client 类型将实现此接口。
type ClientInfoProvider interface {
	GetID() string
	GetRole() string
	GetSessionID() string
	GetUserID() string
	Send(message []byte) error // 假设 Client 有发送消息的方法
	// CloseSendChannel() // 可选，如果 Session 需要关闭 Client 的发送通道
}

// SessionStatus 定义了会话的可能状态
type SessionStatus string

// 定义客户端角色常量
const (
	RoleCardEnd = "card"
	RolePOSEnd  = "pos"
)

const (
	StatusWaitingForPairing SessionStatus = "waiting_for_pairing" // 等待配对
	StatusPaired            SessionStatus = "paired"              // 已配对
	StatusTerminated        SessionStatus = "terminated"          // 已终止
)

// Session 代表一个 NFC 中继会话，包含一个传卡端和一个收卡端客户端。
// TODO: 后续可以增加会话超时、更复杂的状态管理等。
type Session struct {
	SessionID string // 唯一的会话ID

	// 使用指针类型，因为客户端可能先有一个连接，另一个后加入
	CardEndClient ClientInfoProvider // 传卡端客户端 (NFC卡模拟端)
	POSEndClient  ClientInfoProvider // 收卡端客户端 (POS机模拟端)

	Status SessionStatus // 会话当前状态

	CreatedAt         time.Time // 会话创建时间
	LastActivityTime  time.Time // 跟踪最后一次APDU交换或重要活动的时间
	TerminatedAt      time.Time // 会话终止时间
	TerminationReason string    // 会话终止的原因

	// APDU交换计数
	upstreamAPDUCount   int64 // 从POS到卡的APDU数量
	downstreamAPDUCount int64 // 从卡到POS的APDU数量

	// Mutex 用于保护会话内部数据的并发访问，特别是客户端的分配和状态更新。
	mu sync.RWMutex
}

// NewSession 创建一个新的会话实例。
func NewSession(sessionID string) *Session {
	now := time.Now()
	return &Session{
		SessionID:        sessionID,
		Status:           StatusWaitingForPairing,
		CreatedAt:        now,
		LastActivityTime: now, // 初始化为当前时间
	}
}

// UpdateActivityTime 更新会话的最后活动时间。
func (s *Session) UpdateActivityTime() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastActivityTime = time.Now()
}

// IsInactive 检查会话是否在指定的持续时间后处于非活动状态。
func (s *Session) IsInactive(timeout time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Only consider paired sessions for inactivity timeout based on APDU exchange
	if s.Status != StatusPaired {
		return false // Or handle other statuses differently if needed
	}
	return time.Since(s.LastActivityTime) > timeout
}

// Terminate 标记会话为已终止。可以添加其他清理逻辑。
func (s *Session) Terminate() {
	s.TerminateWithReason("会话终止，无指定原因")
}

// TerminateWithReason 标记会话为已终止，并记录终止原因。
func (s *Session) TerminateWithReason(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusTerminated
	s.TerminatedAt = time.Now()
	s.TerminationReason = reason
	// 可选，如果不再需要，清除客户端引用
	// s.CardEndClient = nil
	// s.POSEndClient = nil
}

// IsTerminated 检查会话是否已终止
func (s *Session) IsTerminated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status == StatusTerminated
}

// SetClient 将客户端根据其角色分配到会话中。
// 返回配对是否成功以及是否有错误。
func (s *Session) SetClient(client ClientInfoProvider, role string) (paired bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == StatusTerminated {
		return false, &SessionError{Message: "会话已终止，无法加入"}
	}

	if client == nil {
		return false, &SessionError{Message: "客户端实例不能为nil"}
	}

	switch role {
	case RoleCardEnd:
		if s.CardEndClient != nil && s.CardEndClient != client {
			return false, &SessionError{Message: fmt.Sprintf("角色 '%s' 已被占用", RoleCardEnd)}
		}
		s.CardEndClient = client
	case RolePOSEnd:
		if s.POSEndClient != nil && s.POSEndClient != client {
			return false, &SessionError{Message: fmt.Sprintf("角色 '%s' 已被占用", RolePOSEnd)}
		}
		s.POSEndClient = client
	default:
		return false, &SessionError{Message: "无效的客户端角色: " + role}
	}

	if s.CardEndClient != nil && s.POSEndClient != nil && s.Status == StatusWaitingForPairing {
		s.Status = StatusPaired
		s.LastActivityTime = time.Now() // Update activity time when session becomes paired
		return true, nil
	}
	return false, nil
}

// RemoveClient 从会话中移除指定的客户端。
// 如果会话因此不再配对或变为空，则更新其状态。
func (s *Session) RemoveClient(client ClientInfoProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == StatusTerminated {
		return
	}

	peerDisconnected := false
	if s.CardEndClient == client {
		s.CardEndClient = nil
		peerDisconnected = true
	} else if s.POSEndClient == client {
		s.POSEndClient = nil
		peerDisconnected = true
	}

	if peerDisconnected && s.Status == StatusPaired {
		s.Status = StatusWaitingForPairing // 一方离开，回到等待状态
		// s.LastActivityTime = time.Now() // Optionally update activity time here too
	}

	// 如果会话变空，可以考虑是否将其标记为可清理或终止
	// if s.CardEndClient == nil && s.POSEndClient == nil {
	// 	s.Status = StatusTerminated // 或者一个 "empty" 状态
	// }
}

// GetPeer 获取会话中指定客户端的对端客户端。
// 如果没有对端或者会话未配对，则返回 nil。
func (s *Session) GetPeer(client ClientInfoProvider) ClientInfoProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Status != StatusPaired {
		return nil
	}

	if s.CardEndClient == client {
		return s.POSEndClient
	} else if s.POSEndClient == client {
		return s.CardEndClient
	}
	return nil // 给定客户端不在此会话中
}

// RecordUpstreamAPDU 记录一次从POS到卡的APDU交换
func (s *Session) RecordUpstreamAPDU() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.upstreamAPDUCount++
	s.LastActivityTime = time.Now()
}

// RecordDownstreamAPDU 记录一次从卡到POS的APDU交换
func (s *Session) RecordDownstreamAPDU() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.downstreamAPDUCount++
	s.LastActivityTime = time.Now()
}

// GetUpstreamAPDUCount 获取从POS到卡的APDU交换计数
func (s *Session) GetUpstreamAPDUCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.upstreamAPDUCount
}

// GetDownstreamAPDUCount 获取从卡到POS的APDU交换计数
func (s *Session) GetDownstreamAPDUCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.downstreamAPDUCount
}

// SessionError 定义了会话相关的错误类型
type SessionError struct {
	Message string
}

func (e *SessionError) Error() string {
	return e.Message
}
