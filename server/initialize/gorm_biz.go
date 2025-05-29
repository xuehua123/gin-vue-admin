package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin"
)

func bizModel() error {
	db := global.GVA_DB
	err := db.AutoMigrate(
		// NFC中继管理系统表
		nfc_relay_admin.NfcAuditLog{},
		nfc_relay_admin.NfcClientBanRecord{},
		nfc_relay_admin.NfcUserSecurityProfile{},
	)
	if err != nil {
		return err
	}
	return nil
}
