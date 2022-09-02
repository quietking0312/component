package mgin

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

const (
	AccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	AccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	AccessControlAllowMethods     = "Access-Control-Allow-Methods"
	AccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	AccessControlAllowCredentials = "Access-Control-Allow-Credentials"
)

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		c.Header(AccessControlAllowOrigin, "*")
		c.Header(AccessControlAllowHeaders, strings.Join([]string{"Content-Type"}, ","))
		c.Header(AccessControlAllowMethods, strings.Join([]string{http.MethodGet, http.MethodPost, http.MethodOptions,
			http.MethodPut, http.MethodDelete, "ws", "wss"}, ","))
		c.Header(AccessControlExposeHeaders, strings.Join([]string{"Content-Length", AccessControlAllowOrigin,
			AccessControlAllowHeaders, "Content-Type", "Content-Disposition"}, ","))
		c.Header(AccessControlAllowCredentials, "true")

		if method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}
