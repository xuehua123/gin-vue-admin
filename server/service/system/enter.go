package system

var configManagerServiceInstance *ConfigManagerService

type ServiceGroup struct {
	JwtService
	ApiService
	MenuService
	UserService
	CasbinService
	InitDBService
	AutoCodeService
	BaseMenuService
	AuthorityService
	DictionaryService
	SystemConfigService
	OperationRecordService
	DictionaryDetailService
	AuthorityBtnService
	SysExportTemplateService
	SysParamsService
	AutoCodePlugin   autoCodePlugin
	AutoCodePackage  autoCodePackage
	AutoCodeHistory  autoCodeHistory
	AutoCodeTemplate autoCodeTemplate
}

// GetConfigManagerService 获取配置管理服务实例
func (s *ServiceGroup) GetConfigManagerService() *ConfigManagerService {
	if configManagerServiceInstance == nil {
		configManagerServiceInstance = NewConfigManagerService()
	}
	return configManagerServiceInstance
}
