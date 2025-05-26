package router

import (
	nfcRouter "github.com/flipped-aurora/gin-vue-admin/server/nfc_r
	"github.com/flipped-aurora/gin-vue-admin/server/router/example"
	"github.com/flipped-aurora/gin-vue-admin/server/router/system"
)

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	System   system.RouterGroup
	Example  example.RouterGroup
	NFCRelay nfcRouter.RouterGroup
}
