package nfc_relay

import "github.com/flipped-aurora/gin-vue-admin/server/global"

type PairingStatus string

const (
	PairingStatusPending   PairingStatus = "pending"
	PairingStatusCompleted PairingStatus = "completed"
	PairingStatusFailed    PairingStatus = "failed"
	PairingStatusExpired   PairingStatus = "expired"
)

// Pairing 结构体
type Pairing struct {
	global.GVA_MODEL
	InitiatorClientID string        `json:"initiatorClientId" gorm:"index;comment:发起配对的客户端ID"`
	ResponderClientID string        `json:"responderClientId" gorm:"index;comment:响应配对的客户端ID"`
	Status            PairingStatus `json:"status" gorm:"comment:配对状态"`
}

// TableName Pairing 表名
func (Pairing) TableName() string {
	return "nfc_pairings"
}
