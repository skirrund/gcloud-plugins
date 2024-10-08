package gxxljob

type Registry struct {
	RegistryGroup string `json:"registryGroup"` // 固定值EXECUTOR
	RegistryKey   string `json:"registryKey"`   // 执行器AppName
	RegistryValue string `json:"registryValue"` // 执行器地址，内置服务跟地址
}
