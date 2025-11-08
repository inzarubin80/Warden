package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Simple in-memory store for state -> code_verifier (for PKCE)
type stateData struct {
	CodeVerifier string
	Expiry       time.Time
}

var (
	stateStore   = map[string]stateData{}
	stateStoreMu sync.Mutex
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/exchange", exchangeHandler)
	http.HandleFunc("/userinfo", userInfoHandler)

	log.Printf("TestServer starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("YANDEX_CLIENT_ID")
	if clientID == "" {
		http.Error(w, "YANDEX_CLIENT_ID not set", http.StatusInternalServerError)
		return
	}

	// redirect_uri may be passed as ?redirect or fallback to env REDIRECT_URI
	redirectURI := r.URL.Query().Get("redirect")
	if redirectURI == "" {
		redirectURI = os.Getenv("REDIRECT_URI")
	}
	if redirectURI == "" {
		http.Error(w, "redirect URI not set (pass ?redirect= or set REDIRECT_URI)", http.StatusBadRequest)
		return
	}

	pkce := r.URL.Query().Get("pkce") == "1"
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "login:info"
	}

	state := randomURLSafe(24)

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("scope", scope)

	// PKCE handling:
	// - if client provides code_challenge (mobile generates verifier locally), just forward it
	// - otherwise, if pkce=1, generate verifier on server and store it (legacy/server-side PKCE)
	clientChallenge := r.URL.Query().Get("code_challenge")
	if clientChallenge != "" {
		q.Set("code_challenge", clientChallenge)
		q.Set("code_challenge_method", "S256")
	} else if pkce {
		verifier := randomURLSafe(64)
		challenge := codeChallenge(verifier)
		q.Set("code_challenge", challenge)
		q.Set("code_challenge_method", "S256")

		// store verifier for state (server-side PKCE)
		stateStoreMu.Lock()
		stateStore[state] = stateData{
			CodeVerifier: verifier,
			Expiry:       time.Now().Add(5 * time.Minute),
		}
		stateStoreMu.Unlock()
	}

	// Authorization endpoint. Yandex supports oauth.yandex.com / oauth.yandex.ru.
	// We use oauth.yandex.com here.
	authURL := "https://oauth.yandex.com/authorize?" + q.Encode()
	http.Redirect(w, r, authURL, http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("YANDEX_CLIENT_ID")
	clientSecret := os.Getenv("YANDEX_CLIENT_SECRET")
	tokenEndpoint := os.Getenv("YANDEX_TOKEN_ENDPOINT")
	if tokenEndpoint == "" {
		tokenEndpoint = "https://oauth.yandex.com/token"
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	// look up code_verifier for PKCE, if stored
	var verifier string
	stateStoreMu.Lock()
	if d, ok := stateStore[state]; ok {
		verifier = d.CodeVerifier
		delete(stateStore, state)
	}
	stateStoreMu.Unlock()

	// Do token exchange
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", clientID)
	// include client_secret if available (web flow)
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	// include redirect_uri if provided by env (recommended)
	redirectURI := os.Getenv("REDIRECT_URI")
	if redirectURI != "" {
		form.Set("redirect_uri", redirectURI)
	}
	if verifier != "" {
		form.Set("code_verifier", verifier)
	}

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		http.Error(w, "creating request failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "token request failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		w.WriteHeader(http.StatusBadGateway)
		w.Write(body)
		return
	}

	// forward token response as-is
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// exchangeHandler allows mobile clients to POST { code, code_verifier, redirect_uri? }
// and lets the server perform the token exchange with Yandex on behalf of the mobile app.
func exchangeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Code         string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
		RedirectURI  string `json:"redirect_uri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Code == "" {
		http.Error(w, "code required", http.StatusBadRequest)
		return
	}

	clientID := os.Getenv("YANDEX_CLIENT_ID")
	clientSecret := os.Getenv("YANDEX_CLIENT_SECRET")
	tokenEndpoint := os.Getenv("YANDEX_TOKEN_ENDPOINT")
	if tokenEndpoint == "" {
		tokenEndpoint = "https://oauth.yandex.com/token"
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", req.Code)
	form.Set("client_id", clientID)
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	if req.RedirectURI != "" {
		form.Set("redirect_uri", req.RedirectURI)
	} else if envRedirect := os.Getenv("REDIRECT_URI"); envRedirect != "" {
		// use env redirect if provided
		form.Set("redirect_uri", envRedirect)
	}
	if req.CodeVerifier != "" {
		form.Set("code_verifier", req.CodeVerifier)
	}

	req2, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		http.Error(w, "creating request failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req2)
	if err != nil {
		http.Error(w, "token request failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		w.WriteHeader(http.StatusBadGateway)
		w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func userInfoHandler(w http.ResponseWriter, r *http.Request) {
	// Accept token either in Authorization header or ?access_token=
	var token string
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "oauth ") || strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		// preserve scheme exactly as provided
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 {
			token = parts[1]
		}
	}
	if token == "" {
		token = r.URL.Query().Get("access_token")
	}
	if token == "" {
		http.Error(w, "access token required", http.StatusBadRequest)
		return
	}

	infoURL := "https://login.yandex.ru/info?format=json"
	req, _ := http.NewRequest("GET", infoURL, nil)
	req.Header.Set("Authorization", "OAuth "+token)

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "failed to fetch user info: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		w.WriteHeader(http.StatusBadGateway)
		w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// helpers
func randomURLSafe(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	// trim to requested length if base64 expands
	if len(s) > n {
		return s[:n]
	}
	return s
}

func codeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// optional: pretty-print JSON for debugging
func prettyJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
