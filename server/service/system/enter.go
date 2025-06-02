package system

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
	UserOnlineService
	DeviceLogService
	AutoCodePlugin   autoCodePlugin
	AutoCodePackage  autoCodePackage
	AutoCodeHistory  autoCodeHistory
	AutoCodeTemplate autoCodeTemplate
}
