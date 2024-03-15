package loghttp

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/xfali/xlog"
	"github.com/ydx1011/gopher-web/buffer"
	"github.com/ydx1011/yfig"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	LogReqHeaderKey  = "neve.web.log.requestHeader"
	LogReqBodyKey    = "neve.web.log.requestBody"
	LogRespHeaderKey = "neve.web.log.responseHeader"
	LogRespBodyKey   = "neve.web.log.responseBody"
	LogLevelKey      = "neve.web.log.level"

	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelPanic = "panic"
	LogLevelFatal = "fatal"
)

type LogOpt func(setter Setter)

type logFunc func(fmt string, args ...interface{})

type HttpLogger interface {
	// 获得按配置初始化的日志handler
	LogHttp() gin.HandlerFunc
	// 按参数配置日志handler
	OptLogHttp(opts ...LogOpt) gin.HandlerFunc
	// 根据参数配置Clone出新的HttpLogger
	Clone(opts ...LogOpt) HttpLogger
}

type Setter interface {
	Set(key string, value interface{})
}

func (util *LogHttpUtil) output(fmt string, args ...interface{}) {
	if util.logFunc == nil {
		util.Logger.Infof(fmt, args...)
	} else {
		util.logFunc(fmt, args...)
	}
}

type hLogger struct {
	LogHttpUtil
	pool buffer.Pool
}

func NewFromConfig(conf yfig.Properties, logger xlog.Logger) *hLogger {
	ret := &hLogger{
		LogHttpUtil: *NewLogHttpUtil(conf, logger),
		pool:        buffer.NewPool(),
	}
	return ret
}

func (util *hLogger) LogHttp() gin.HandlerFunc {
	return util.log
}

func (util *hLogger) OptLogHttp(opts ...LogOpt) gin.HandlerFunc {
	return util.clone(opts...).log
}

func (util *hLogger) Clone(opts ...LogOpt) HttpLogger {
	return util.clone(opts...)
}

func (util *hLogger) clone(opts ...LogOpt) *hLogger {
	ret := &hLogger{}
	ret.Logger = util.Logger
	ret.LogReqHeader = util.LogReqHeader
	ret.LogReqBody = util.LogReqBody
	ret.LogRespHeader = util.LogRespHeader
	ret.LogRespBody = util.LogRespBody
	ret.Level = util.Level
	ret.pool = util.pool

	for _, opt := range opts {
		opt(ret)
	}

	ret.initLog()
	return ret
}

func (util *hLogger) log(c *gin.Context) {
	start := time.Now()

	path := c.Request.URL.Path
	clientIP := c.ClientIP()
	method := c.Request.Method
	requestId := RandomId(16)
	params := c.Params
	querys := c.Request.URL.RawQuery
	reqHeaderBuf := util.pool.Get()
	defer util.pool.Put(reqHeaderBuf)
	if util.LogReqHeader {
		getHeaderBuffer(reqHeaderBuf, c.Request.Header)
	}

	//c.Set(REQEUST_ID, requestId)

	reqBody := ""
	if util.LogReqBody {
		reqBodyWrapper := buffer.NewReadWriteCloser(util.pool)
		io.Copy(reqBodyWrapper, c.Request.Body)
		c.Request.Body.Close()
		reqBody = string(reqBodyWrapper.Bytes())
		c.Request.Body = reqBodyWrapper
		// Must close here to release buffer.
		defer reqBodyWrapper.Close()
	}

	var blw *responseBodyWriter
	if util.LogRespBody {
		blw = newResponseBodyWriter(c.Writer, buffer.NewReadWriteCloser(util.pool))
		c.Writer = blw
		defer blw.Close()
	}

	if util.LogReqBody {
		util.output("[Request  %s] [path]: %s , [method]: %s , [client ip]: %s %s, [params]: %v , [query]: %s , [data]: %s\n",
			requestId, path, method, clientIP, reqHeaderBuf.String(), params, querys, reqBody)
	} else {
		util.output("[Request  %s] [path]: %s , [method]: %s , [client ip]: %s %s, [params]: %v , [query]: %s\n",
			requestId, path, method, clientIP, reqHeaderBuf.String(), params, querys)
	}

	// 处理请求
	c.Next()

	// 结束时间
	end := time.Now()
	//执行时间
	latency := end.Sub(start)

	statusCode := c.Writer.Status()

	var data string
	if util.LogRespBody {
		data = string(blw.getBody())
	}
	respHeaderBuf := util.pool.Get()
	defer util.pool.Put(respHeaderBuf)
	if util.LogRespHeader {
		rh := c.Writer.Header()
		if rh != nil {
			getHeaderBuffer(respHeaderBuf, rh.Clone())
		}
	}
	util.output("[Response %s] [path]: %s , [method]: %s , [latency]: %d ms, [status]: %d %s%s\n",
		requestId, path, method, latency/time.Millisecond, statusCode, respHeaderBuf.String(), data)
}

type LogHttpUtil struct {
	Logger xlog.Logger

	ignore struct{} `figPx:"gopher.web"`
	// with request header log
	LogReqHeader bool `fig:"log.requestHeader"`
	// with request body log
	LogReqBody bool `fig:"log.requestBody"`
	// with response header log
	LogRespHeader bool `fig:"log.responseHeader"`
	// with response body log
	LogRespBody bool `fig:"log.responseBody"`
	// log level
	Level string `fig:"log.level"`

	logFunc logFunc
}

type DefaultHttpLogger LogHttpUtil

func NewLogHttpUtil(conf yfig.Properties, logger xlog.Logger) *LogHttpUtil {
	ret := &LogHttpUtil{
		Logger: logger,
	}
	yfig.Fill(conf, ret)
	ret.initLog()
	return ret
}

func (util *LogHttpUtil) initLog() {
	lv := strings.ToLower(util.Level)
	switch lv {
	case LogLevelDebug:
		util.logFunc = util.Logger.Debugf
	case LogLevelInfo:
		util.logFunc = util.Logger.Infof
	case LogLevelWarn:
		util.logFunc = util.Logger.Warnf
	case LogLevelError:
		util.logFunc = util.Logger.Errorf
	case LogLevelPanic:
		util.logFunc = util.Logger.Panicf
	case LogLevelFatal:
		util.logFunc = util.Logger.Fatalf
	default:
		util.logFunc = util.Logger.Infof
	}
}

func (util *LogHttpUtil) Set(key string, value interface{}) {
	switch key {
	case LogReqHeaderKey:
		if v, ok := value.(bool); ok {
			util.LogReqHeader = v
		}
		break
	case LogReqBodyKey:
		if v, ok := value.(bool); ok {
			util.LogReqBody = v
		}
		break
	case LogRespHeaderKey:
		if v, ok := value.(bool); ok {
			util.LogRespHeader = v
		}
		break
	case LogRespBodyKey:
		if v, ok := value.(bool); ok {
			util.LogRespBody = v
		}
		break
	case LogLevelKey:
		if v, ok := value.(string); ok {
			util.Level = v
		}
		break
	}
}

type responseBodyWriter struct {
	gin.ResponseWriter
	body *buffer.ReadWriteCloser
}

func newResponseBodyWriter(w gin.ResponseWriter, rwc *buffer.ReadWriteCloser) *responseBodyWriter {
	ret := &responseBodyWriter{
		ResponseWriter: w,
		body:           rwc,
	}
	ret.body.Write([]byte(" , [data]: "))
	return ret
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseBodyWriter) WriteString(s string) (int, error) {
	w.body.Write([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

func (w *responseBodyWriter) Close() error {
	return w.body.Close()
}

func (w *responseBodyWriter) getBody() []byte {
	return w.body.Bytes()
}

func getHeaderStr(header http.Header) string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(", [header]: ")
	if len(header) > 0 {
		for k, vs := range header {
			buf.WriteString(k)
			buf.WriteString("=")
			for i := range vs {
				buf.WriteString(vs[i])
				if i < len(vs)-1 {
					buf.WriteString(",")
				}
			}
			buf.WriteString(" ")
		}
	}
	buf.WriteString(" ,")
	return buf.String()
}

func getHeaderBuffer(buf *bytes.Buffer, header http.Header) {
	buf.WriteString(", [header]: ")
	if len(header) > 0 {
		for k, vs := range header {
			buf.WriteString(k)
			buf.WriteString("=")
			for i := range vs {
				buf.WriteString(vs[i])
				if i < len(vs)-1 {
					buf.WriteString(",")
				}
			}
			buf.WriteString(" ")
		}
		buf.WriteString(" ")
	}
}

func RandomId(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}
