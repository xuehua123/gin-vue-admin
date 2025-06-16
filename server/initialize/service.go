package initialize

import "github.com/flipped-aurora/gin-vue-admin/server/service"

func Service() {
	service.ServiceGroupApp.SystemServiceGroup.NotificationService.Start()
}
