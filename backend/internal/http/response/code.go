package response

const (
	// CodeSuccess 通用成功码。
	CodeSuccess = 0

	// CodeInvalidParams 通用请求错误。
	CodeInvalidParams = 40001

	// CodeInvalidUsername User / Auth。
	CodeInvalidUsername = 40002
	CodePasswordTooLong = 40003

	CodeUsernameExists = 40901
	CodeEmailExists    = 40902

	// CodeUnauthorized 鉴权相关，后面登录阶段会使用。
	CodeUnauthorized = 40101
	CodeForbidden    = 40301

	// CodeInternalError 服务端异常。
	CodeInternalError = 50000
)
