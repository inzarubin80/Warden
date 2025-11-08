package http

import (
	"net/http"

	"github.com/inzarubin80/Server/internal/app/uhttp"
)

type (
	PingHandler struct {
		name string
	}
)

func NewPingHandlerHandler(name string) *PingHandler {
	return &PingHandler{
		name: name,
	}
}

func (h *PingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uhttp.SendSuccessfulResponse(w, []byte("{}"))

}
