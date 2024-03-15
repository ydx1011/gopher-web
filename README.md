# gopher-web


gopher-web是gopher的WEB扩展组件，用于集成WEB相关服务。

内置WEB中间件为[gin](https://github.com/gin-gonic/gin)

## 安装
```
go get github.com/ydx1011/gopher-web
```

## 使用

### 1. gopher集成（依赖[gopher-core](https://github.com/ydx1011/gopher-core)）
```
app := gopher.NewFileConfigApplication("assets/config-test.yaml")
app.RegisterBean(gopherweb.NewGinProcessor())
// 或者
// app.RegisterBean(gingopher.NewProcessor())
//注册值注入处理器，用于根据配置注入值（非必须）
app.RegisterBean(processor.NewValueProcessor())
//注册其他对象
app.RegisterBean(&testProcess{})
app.RegisterBean(&webBean{})
app.Run()
```

### 2. 配置
在config-example.yaml中配置示例如下：
```
gopher:
  web:
    log:
      requestHeader: true
      requestBody: true
      responseHeader: true
      responseBody: true
      level: "warn"

    server:
      contextPath: ""
      host: ""
      port: 8080
      tls:
        cert: 
        key:
      readTimeout: 15
      writeTimeout: 15
      idleTimeout: 15
```
* 【gopher.web.log】配置rest的日志输出，包含request header、body，response header、body以及配置日志级别，根据项目需要进行配置。
* 【gopher.web.server】配置WEB服务的端口、读写超时等配置，contextPath配置总的根路由路径，如contextPath: "/order"
* 【gopher.web.server.tls】https tls相关配置

### 3. 注册路由
注册的bean实现 HttpRoutes(engine gin.IRouter)方法
```
//webBean通过app.RegisterBean(&webBean{})注册，并实现下列方法：

func (b *webBean) HttpRoutes(engine gin.IRouter) {
	engine.GET("test", b.HttpLogger.LogHttp(), func(context *gin.Context) {
		context.JSON(http.StatusOK, result.Ok(b.V))
	})

	engine.POST("test", b.HttpLogger.LogHttp(), func(context *gin.Context) {
		d, err := context.GetRawData()
		if err != nil {
			context.AbortWithStatus(http.StatusBadRequest)
			return
		}
		context.JSON(http.StatusOK, result.Ok(string(d)))
	})
}
```

### 4. 输出日志配置
注入loghttp.HttpLogger，在gin.IRouter中添加该handler
```
type webBean struct {
	V          string //`fig:"Log.Level"`
	//注入
	HttpLogger loghttp.HttpLogger `inject:""`
}
func (b *webBean) HttpRoutes(engine gin.IRouter) {
    //使用“b.HttpLogger.LogHttp()”配置，作为首个handler
	engine.GET("test", b.HttpLogger.LogHttp(), func(context *gin.Context) {
		context.JSON(http.StatusOK, result.Ok(b.V))
	})
}
```

### 5. 注册全局过滤器
1. 注册的bean实现 FilterHandler(ctx *gin.Context) 方法
```
type filter struct{}

func (f *filter) FilterHandler(context *gin.Context) {
    if f.pass() {
        // 继续执行
        context.Next()
    } else {
        // 过滤并阻断
        context.Abort()
    }
}
```
过滤器执行的顺序遵循bean注册的先后顺序

2. 通过NewProcessor时添加过滤器
```
app.RegisterBean(gigopher.NewProcessor(gigopher.OptAddFilters(func(context *gin.Context) {
		context.Set("hello", "world")
		context.Next()
	})))
```