package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	authinterface "github.com/inzarubin80/Server/internal/app/authinterface"
	"github.com/inzarubin80/Server/internal/app/uhttp"
	"golang.org/x/oauth2"
)

type LoginHandler struct {
	name              string
	provadersConf     authinterface.MapProviderOauthConf
	store             *sessions.CookieStore
	loginStateStore   map[string]time.Time
	loginStateStoreMu sync.Mutex
}

func NewLoginHandler(provadersConf authinterface.MapProviderOauthConf, name string, store *sessions.CookieStore) *LoginHandler {
	return &LoginHandler{
		name:              name,
		provadersConf:     provadersConf,
		store:             store,
		loginStateStore:   make(map[string]time.Time),
		loginStateStoreMu: sync.Mutex{},
	}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Provider      string `json:"provider"`
		CodeChallenge string `json:"code_challenge"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		uhttp.SendErrorResponse(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Provider == "" {
		uhttp.SendErrorResponse(w, http.StatusBadRequest, "provider required")
		return
	}

	cfg, ok := h.provadersConf[req.Provider]
	if !ok || cfg == nil || cfg.Oauth2Config == nil {
		uhttp.SendErrorResponse(w, http.StatusBadRequest, "unknown provider")
		return
	}

	state := randomURLSafe(24)

	// handle PKCE: prefer client-provided code_challenge; if not and pkce requested, generate verifier
	var challenge string
	// require client-provided code_challenge (mobile generates verifier)
	if req.CodeChallenge == "" {
		uhttp.SendErrorResponse(w, http.StatusBadRequest, "code_challenge required from client")
		return
	}
	challenge = req.CodeChallenge

	// save state server-side (one-time, short TTL) so we can validate it at exchange
	h.loginStateStoreMu.Lock()
	h.loginStateStore[state] = time.Now().Add(5 * time.Minute)
	h.loginStateStoreMu.Unlock()

	localCfg := *cfg.Oauth2Config
	localCfg.RedirectURL = cfg.Oauth2Config.RedirectURL
	opts := []oauth2.AuthCodeOption{}
	if challenge != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_challenge", challenge))
		opts = append(opts, oauth2.SetAuthURLParam("code_challenge_method", "S256"))
	}
	authURL := localCfg.AuthCodeURL(state, opts...)

	resp := map[string]string{
		"auth_url": authURL,
		"state":    state,
	}
	b, _ := json.Marshal(resp)
	uhttp.SendSuccessfulResponse(w, b)
}

func randomURLSafe(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > n {
		return s[:n]
	}
	return s
}
