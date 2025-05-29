package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router/example"
	"github.com/flipped-aurora/gin-vue-admin/server/router/nfc_relay_admin"
	"github.com/flipped-aurora/gin-vue-admin/server/router/system"
)

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	System        system.RouterGroup
	Example       example.RouterGroup
	NfcRelayAdmin nfc_relay_admin.RouterGroup
}
