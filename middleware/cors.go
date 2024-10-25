package middleware

import (
	"net/http"
	"strings"
)

const (
	corsAllowOrigin      = "Access-Control-Allow-Origin"
	corsAllowHeaders     = "Access-Control-Allow-Headers"
	corsAllowMethods     = "Access-Control-Allow-Methods"
	corsExposeHeaders    = "Access-Control-Expose-Headers"
	corsAllowCredentials = "Access-Control-Allow-Credentials"
)

func Cors(r http.Request, w http.Response) {
	w.Header.Set(corsAllowOrigin, "*")
	w.Header.Set(corsAllowHeaders, "Content-Type")
	w.Header.Set(corsAllowMethods, strings.Join([]string{http.MethodGet, http.MethodPost, http.MethodOptions,
		http.MethodPut, http.MethodDelete, "ws", "wss"}, ","))
	w.Header.Set(corsExposeHeaders, strings.Join([]string{"Content-Length", corsAllowOrigin, corsAllowHeaders, "Content-Type", "Content-Disposition"}, ","))
	w.Header.Set(corsAllowCredentials, "true")
	if r.Method == http.MethodOptions {
		w.StatusCode = http.StatusNoContent
	}
}
