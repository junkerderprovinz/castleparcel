// Package upload serves the resumable tus upload endpoint and its optional
// upload password gate.
package upload

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/junkerderprovinz/castleparcel/server-go/internal/config"
)

// UploadTokenHeader carries the upload password, as the client sends it from
// src/upload.ts.
const UploadTokenHeader = "x-fd-upload-token"

// RequestHeader names the file request a tus upload is for, and
// RequestKeyHeader carries the request's public key from the request link as
// the uploader's proof. Together they stand in for the upload password.
const (
	RequestHeader    = "x-fd-request"
	RequestKeyHeader = "x-fd-request-key"
)

// uploadTokenMatches compares provided and secret in constant time.
// subtle.ConstantTimeCompare returns at once on a length mismatch, which would
// leak the secret's length, so a mismatch still runs a dummy compare.
func uploadTokenMatches(provided, secret string) bool {
	a := []byte(provided)
	b := []byte(secret)
	if len(a) != len(b) {
		subtle.ConstantTimeCompare(a, make([]byte, len(a)))
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

// IsUploadAuthorized reports whether a write request passes the upload gate.
// Without an UPLOAD_PASSWORD every request passes; with one, token has to be a
// non-empty match.
func IsUploadAuthorized(token string, protected bool, secret string) bool {
	if !protected {
		return true
	}
	if len(token) == 0 {
		return false
	}
	return uploadTokenMatches(token, secret)
}

type admittedKey struct{}

// Admit marks r as an upload the owner's sign-in let through, which counts in
// place of the upload password.
func Admit(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), admittedKey{}, true))
}

// Allowed reports whether r may write an upload: Admit let it through, or it
// passes the upload password gate.
func Allowed(r *http.Request, cfg config.Config) bool {
	if admitted, _ := r.Context().Value(admittedKey{}).(bool); admitted {
		return true
	}
	return IsUploadAuthorized(r.Header.Get(UploadTokenHeader), cfg.UploadProtected, cfg.UploadPassword)
}
