package result

var (
	OK              = Result{Code: 0, Msg: "ok", HttpStatus: 200}
	InternalError   = Result{Code: -1, Msg: "internal error", HttpStatus: 500}
	ConnectError    = Result{Code: 1001, Msg: "connect error", HttpStatus: 500}
	SettingNilError = Result{Code: 1002, Msg: "setting is nil", HttpStatus: 500}
)
