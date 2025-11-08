package http

import (
	"encoding/json"
	"net/http"

	authinterface "github.com/inzarubin80/Server/internal/app/authinterface"
	"github.com/inzarubin80/Server/internal/app/uhttp"
)

type (
	GetProvadersHandler struct {
		name          string
		provadersConf authinterface.MapProviderOauthConf
	}
)

func NewProvadersHandler(provadersConf authinterface.MapProviderOauthConf, name string) *GetProvadersHandler {
	return &GetProvadersHandler{
		name:          name,
		provadersConf: provadersConf,
	}
}

func (h *GetProvadersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	providerOauthConfFrontend := []authinterface.ProviderOauthConfFrontend{}
	for key, value := range h.provadersConf {
		providerOauthConfFrontend = append(providerOauthConfFrontend,
			authinterface.ProviderOauthConfFrontend{
				Provider: key,
				IconSVG:  value.IconSVG,
			},
		)
	}

	jsonContent, err := json.Marshal(providerOauthConfFrontend)

	if err != nil {
		uhttp.SendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonContent)
	
}
