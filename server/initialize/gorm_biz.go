package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay"
)

func bizModel() error {
	db := global.GVA_DB
	err := db.AutoMigrate(
		nfc_relay.Pairing{},
	)
	if err != nil {
		return err
	}
	return nil
}
