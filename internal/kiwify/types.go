package kiwify

// oauthTokenResponse is the body returned by POST /oauth/token.
// expires_in may be a JSON number or string depending on the API/version.
type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   any    `json:"expires_in"`
}

// apiErrorBody is a best-effort decode of Kiwify error JSON payloads.
type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}
