package gateway

import (
	"net/http"

	bucktooth "github.com/scttfrdmn/bucktooth"
)

func webFileServer() http.Handler {
	return bucktooth.WebFileServer()
}
