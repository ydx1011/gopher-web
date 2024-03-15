package gingopher

import "github.com/gin-gonic/gin"

type Component interface {
	HttpRoutes(engine gin.IRouter)
}

type Filter interface {
	FilterHandler(ctx *gin.Context)
}
