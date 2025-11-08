# TestServer — Mobile PKCE flow (Yandex)

This TestServer provides helpers for Yandex OAuth flows and is intended as a lightweight authorization proxy for testing mobile and web clients.

Key endpoints
- `GET /login?redirect=<REDIRECT_URI>&pkce=1&scope=...`  
  - Starts authorization by redirecting to `https://oauth.yandex.com/authorize?...`.
  - If the request contains `code_challenge=<value>` the server will forward that challenge to Yandex (mobile-generated PKCE).  
  - If `pkce=1` and no `code_challenge` is supplied, the server will generate a `code_verifier`, store it (in-memory) associated with `state`, and send `code_challenge` to Yandex (server-side PKCE).

- `GET /callback?code=...&state=...`  
  - Used by web flows: TestServer exchanges the `code` for tokens at Yandex `token` endpoint and returns the token JSON.
  - If server-side PKCE was used, the server reads the stored `code_verifier` by `state`.

- `POST /exchange`  
  - Mobile-friendly endpoint. JSON body:
    ```json
    { "code": "...", "code_verifier": "...", "redirect_uri": "myapp://oauth/callback" }
    ```
  - The server will forward this to Yandex token endpoint and return the token JSON. Use HTTPS in production.

- `GET /userinfo?access_token=...` or `Authorization: OAuth <token>`  
  - Proxy to `https://login.yandex.ru/info?format=json`.

Mobile recommended flow (PKCE, server as helper)
1. Mobile generates `code_verifier` (RFC7636, 43-128 chars) and computes `code_challenge = BASE64URL(SHA256(code_verifier))`.
2. Mobile opens system browser to:
   ```
   https://<TESTSERVER_HOST>/login?redirect=myapp://oauth/callback&code_challenge=<CODE_CHALLENGE>&scope=login:info
   ```
3. Yandex authenticates user and redirects to:
   ```
   myapp://oauth/callback?code=AUTH_CODE&state=STATE
   ```
4. Mobile receives deep link, extracts `code` and `state`, then POSTs to:
   ```
   POST https://<TESTSERVER_HOST>/exchange
   Content-Type: application/json
   { "code": "AUTH_CODE", "code_verifier": "ORIGINAL_VERIFIER", "redirect_uri": "myapp://oauth/callback" }
   ```
5. Server exchanges code+verifier with Yandex and returns `{ access_token, refresh_token, expires_in, ... }`.
6. Mobile stores `access_token` in secure storage (Keychain / Keystore). Do not store `client_secret` in app.

Notes and security
- Use HTTPS. Do not pass tokens in query strings in production.  
- Prefer mobile generating `code_verifier` locally; server should only store verifiers for compatibility cases and TTL them (currently 5 minutes).  
- For production, replace in-memory stores with Redis/DB and add client authentication, rate-limiting, and proper logging.


