package gxxljob

const (
	SuccessCode         = 200
	FailureCode         = 500
	serialExecution     = "SERIAL_EXECUTION" //单机串行
	discardLater        = "DISCARD_LATER"    //丢弃后续调度
	coverEarly          = "COVER_EARLY"      //覆盖之前调度
	DefaultExecutorPort = 9999

	DefaultRegistryGroup = "EXECUTOR"

	DefaultAppName = "go-xxljob"

	DefaultRegisterAddressHttp = "http://"
)
