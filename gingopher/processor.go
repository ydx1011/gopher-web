package gingopher

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/xfali/xlog"
	"github.com/ydx1011/gopher-core/bean"
	"github.com/ydx1011/gopher-web/gingopher/common/loghttp"
	"github.com/ydx1011/gopher-web/gingopher/common/recovery"
	"github.com/ydx1011/gopher-web/result"
	"github.com/ydx1011/yfig"
	"net/http"
	"time"
)

type serverConf struct {
	ContextPath  string
	Host         string
	Port         int
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int

	Tls tlsConf
}

type tlsConf struct {
	Cert string
	Key  string
}

type Processor struct {
	conf   yfig.Properties
	logger xlog.Logger
	server *http.Server

	compList []Component

	filters gin.HandlersChain

	panicHandler recovery.PanicHandler
	httpLogger   loghttp.HttpLogger
	logAll       bool
}

type Opt func(p *Processor)

func NewProcessor(opts ...Opt) *Processor {
	ret := &Processor{
		logger: xlog.GetLogger(),
		panicHandler: func(ctx *gin.Context, err interface{}) {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, result.InternalError)
		},
	}
	for _, v := range opts {
		v(ret)
	}
	return ret
}

func (p *Processor) Init(conf yfig.Properties, container bean.Container) error {
	p.conf = conf
	if p.httpLogger == nil {
		p.httpLogger = loghttp.NewFromConfig(conf, p.logger)
	}
	container.Register(p.httpLogger)
	return nil
}

func (p *Processor) Classify(o interface{}) (bool, error) {
	switch v := o.(type) {
	case Component:
		return true, p.parseBean(v)
	case Filter:
		return true, p.parseFilter(v)
	}
	return false, nil
}

func (p *Processor) parseBean(comp Component) error {
	p.compList = append(p.compList, comp)
	return nil
}

func (p *Processor) parseFilter(filter Filter) error {
	p.filters = append(p.filters, filter.FilterHandler)
	return nil
}

func (p *Processor) Process() error {
	return p.start(p.conf)
}

func (p *Processor) start(conf yfig.Properties) error {
	r := gin.New()
	//r.Use(gin.Logger())
	//r.Use(gin.Recovery())

	if p.panicHandler != nil {
		panicU := &recovery.RecoveryUtil{
			Logger:       p.logger,
			PanicHandler: p.panicHandler,
		}
		r.Use(panicU.Recovery())
	}
	if p.logAll {
		r.Use(p.httpLogger.LogHttp())
	}

	if len(p.filters) > 0 {
		r.Use(p.filters...)
	}

	servConf := serverConf{}
	err := conf.GetValue("gopher.web.server", &servConf)
	if err != nil {
		return err
	}

	if servConf.Port == 0 {
		servConf.Port = 8080
	}
	if servConf.ReadTimeout == 0 {
		servConf.ReadTimeout = 15
	}
	if servConf.WriteTimeout == 0 {
		servConf.WriteTimeout = 15
	}
	if servConf.IdleTimeout == 0 {
		servConf.IdleTimeout = 15
	}

	var router gin.IRouter = r
	if servConf.ContextPath != "" {
		router = router.Group(servConf.ContextPath)
	}
	for _, v := range p.compList {
		v.HttpRoutes(router)
	}

	addr := getServeAddr(servConf)
	s := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    time.Duration(servConf.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(servConf.WriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(servConf.IdleTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if servConf.Tls.Cert == "" {
			err := s.ListenAndServe()
			if err != nil {
				p.logger.Errorln(err)
			}
		} else {
			err := s.ListenAndServeTLS(servConf.Tls.Cert, servConf.Tls.Key)
			if err != nil {
				p.logger.Errorln(err)
			}
		}
	}()

	p.server = s

	return nil
}

func (p *Processor) BeanDestroy() error {
	if p.server != nil {
		return p.server.Close()
	}
	return nil
}

func getServeAddr(servConf serverConf) string {
	return fmt.Sprintf("%s:%d", servConf.Host, servConf.Port)
}
