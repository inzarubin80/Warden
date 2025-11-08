package http

import (
	"encoding/json"
	"net/http"

	authinterface "github.com/inzarubin80/Server/internal/app/authinterface"
	"github.com/inzarubin80/Server/internal/app/uhttp"
)

type (
	GetProvadersHandler struct {
		name                      string
		providerOauthConfFrontend []authinterface.ProviderOauthConfFrontend
	}
)

func NewProvadersHandler(providerOauthConfFrontend []authinterface.ProviderOauthConfFrontend, name string) *GetProvadersHandler {
	return &GetProvadersHandler{
		name:                      name,
		providerOauthConfFrontend: providerOauthConfFrontend,
	}
}

func (h *GetProvadersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	jsonContent, err := json.Marshal(h.providerOauthConfFrontend)

	if err != nil {
		uhttp.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonContent)

}
