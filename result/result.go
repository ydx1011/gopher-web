package result

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
)

type Result struct {
	Code int64       `json:"code"`
	Msg  string      `json:"message"`
	Data interface{} `json:"data,omitempty"`

	Err error `json:"error,omitempty"`

	HttpStatus int `json:"-"`
}

func Ok(data interface{}) Result {
	return Result{Code: OK.Code, Msg: OK.Msg, Data: data, HttpStatus: OK.HttpStatus}
}

func (result *Result) WriteJson(ctx *gin.Context) {
	ctx.JSON(result.HttpStatus, *result)
}

func (result *Result) SetCode(code int64) *Result {
	result.Code = code
	return result
}

func (result *Result) SetMessage(v string) *Result {
	result.Msg = v
	return result
}

func (result *Result) SetData(v interface{}) *Result {
	result.Data = v
	return result
}

func (result *Result) SetError(v error) *Result {
	result.Err = v
	return result
}

func (result *Result) SetHttpStatus(v int) *Result {
	result.HttpStatus = v
	return result
}

func (result *Result) Clone() *Result {
	ret := *result
	return &ret
}

func (result *Result) Error() string {
	return result.String()
}

func (result *Result) String() string {
	b, _ := json.Marshal(result)
	return string(b)
}
