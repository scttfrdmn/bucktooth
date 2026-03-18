// Package webhook provides reusable HTTP middleware for webhook security.
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

// HMACVerifier validates inbound webhook requests using HMAC-SHA256 signatures.
type HMACVerifier struct {
	// Secret is the shared secret used to compute the expected signature.
	Secret []byte
	// HeaderName is the HTTP header carrying the signature (e.g. "X-Hub-Signature-256").
	HeaderName string
	// StripPrefix is removed from the header value before hex-decoding (e.g. "sha256=").
	StripPrefix string
}

// Middleware returns an http.Handler that reads the full request body, verifies
// the HMAC-SHA256 signature from the configured header, and either passes the
// request to next or responds with 401 Unauthorized.
//
// The body is replaced with a new reader so downstream handlers can still read it.
func (v *HMACVerifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get(v.HeaderName)
		if sig == "" {
			http.Error(w, "missing signature header", http.StatusUnauthorized)
			return
		}
		sig = strings.TrimPrefix(sig, v.StripPrefix)

		sigBytes, err := hex.DecodeString(sig)
		if err != nil {
			http.Error(w, "invalid signature encoding", http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
			return
		}
		_ = r.Body.Close()

		if !VerifyHMACSHA256(v.Secret, body, sigBytes) {
			http.Error(w, "signature mismatch", http.StatusUnauthorized)
			return
		}

		// Restore body so downstream handlers can read it.
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		next.ServeHTTP(w, r)
	})
}

// VerifyHMACSHA256 returns true if the HMAC-SHA256 of message under secret equals
// the provided sig. The comparison is performed in constant time to prevent
// timing attacks.
func VerifyHMACSHA256(secret, message, sig []byte) bool {
	mac := hmac.New(sha256.New, secret)
	mac.Write(message)
	expected := mac.Sum(nil)
	return hmac.Equal(expected, sig)
}
