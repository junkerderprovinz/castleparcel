// Package headers sets the security headers every response carries.
package headers

import (
	"net/http"
	"strings"
)

// PermissionsPolicy turns off the browser features the app never uses, so an
// injected script could not reach for them either. Clipboard and fullscreen
// keep their same-origin default for the copy button and the video preview,
// and passkeys work on the page itself for the owner's sign-in.
const PermissionsPolicy = "accelerometer=(), autoplay=(), bluetooth=(), browsing-topics=(), " +
	"camera=(), display-capture=(), geolocation=(), gyroscope=(), hid=(), idle-detection=(), " +
	"magnetometer=(), microphone=(), midi=(), payment=(), publickey-credentials-create=(self), " +
	"publickey-credentials-get=(self), screen-wake-lock=(), serial=(), usb=(), xr-spatial-tracking=()"

// ContentSecurityPolicy returns the policy for the app. libsodium compiles its
// WebAssembly at runtime, Vue sets inline style attributes, Vite inlines the
// smallest font files as data: URLs, previews are blob: URLs, the PDF preview
// is an <embed> and the crypto worker starts from a blob: URL made from code
// inside the pinned main script. A non-empty logoOrigin, the origin of an
// external APP_LOGO, is allowed for images and nothing else.
func ContentSecurityPolicy(logoOrigin string) string {
	img := "'self' blob: data:"
	if logoOrigin != "" {
		img += " " + logoOrigin
	}
	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'wasm-unsafe-eval'",
		"style-src 'self' 'unsafe-inline'",
		"img-src " + img,
		"font-src 'self' data:",
		"media-src 'self' blob:",
		"object-src blob:",
		"worker-src 'self' blob:",
		"connect-src 'self'",
		"frame-ancestors 'none'",
		"base-uri 'none'",
		"form-action 'self'",
	}, "; ")
}

// Middleware sets the security headers before next runs, so a handler can
// still replace one for its own response. The paths under own set their
// headers themselves: the fortress's tabs frame another member's page below
// their bar, which frame-ancestors 'none' and DENY would forbid, and give it
// a sandbox of its own.
func Middleware(csp string, own ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, prefix := range own {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}
			h := w.Header()
			h.Set("Content-Security-Policy", csp)
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Permissions-Policy", PermissionsPolicy)
			next.ServeHTTP(w, r)
		})
	}
}
