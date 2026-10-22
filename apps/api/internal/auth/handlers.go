package auth

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
)

const CookieName = "ft_session"

type Handlers struct {
	OTP      *OTP
	Sessions *Sessions
	Users    UserStore
	// SecureCookie must be true everywhere except local development over http.
	SecureCookie bool
	// TrustProxy makes the rightmost X-Forwarded-For entry the client IP.
	// Enable only when exactly one trusted reverse proxy (Traefik) is in front.
	TrustProxy bool
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/otp/request", h.request)
	mux.HandleFunc("POST /api/v1/auth/otp/verify", h.verify)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("GET /api/v1/auth/me", h.me)
}

func (h *Handlers) request(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Phone string `json:"phone"`
	}
	if !decode(w, r, &body) {
		return
	}
	phone, err := NormalizePhone(body.Phone)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_phone")
		return
	}
	switch err := h.OTP.Request(r.Context(), phone, h.clientIP(r)); {
	case err == nil:
		reply(w, http.StatusAccepted, map[string]any{"status": "sent", "expiresInSeconds": int(codeTTL.Seconds())})
	case errors.Is(err, ErrCooldown), errors.Is(err, ErrRateLimited):
		fail(w, http.StatusTooManyRequests, "rate_limited")
	default:
		fail(w, http.StatusBadGateway, "send_failed")
	}
}

func (h *Handlers) verify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if !decode(w, r, &body) {
		return
	}
	phone, err := NormalizePhone(body.Phone)
	if err != nil {
		fail(w, http.StatusBadRequest, "invalid_phone")
		return
	}
	switch err := h.OTP.Verify(r.Context(), phone, normalizeDigits(body.Code)); {
	case err == nil:
	case errors.Is(err, ErrTooManyAttempts):
		fail(w, http.StatusTooManyRequests, "too_many_attempts")
		return
	case errors.Is(err, ErrInvalidCode):
		fail(w, http.StatusUnauthorized, "invalid_code")
		return
	default:
		fail(w, http.StatusInternalServerError, "internal")
		return
	}

	user, err := h.Users.UpsertByPhone(r.Context(), phone)
	if err != nil {
		fail(w, http.StatusInternalServerError, "internal")
		return
	}
	token, err := h.Sessions.Create(r.Context(), user.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "internal")
		return
	}
	http.SetCookie(w, h.cookie(token, int(SessionTTL.Seconds())))
	reply(w, http.StatusOK, map[string]any{"user": user})
}

func (h *Handlers) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		_ = h.Sessions.Delete(r.Context(), c.Value)
	}
	http.SetCookie(w, h.cookie("", -1))
	reply(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) me(w http.ResponseWriter, r *http.Request) {
	user, ok := h.CurrentUser(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	reply(w, http.StatusOK, map[string]any{"user": user})
}

// CurrentUser resolves the session cookie, for use by other handlers.
func (h *Handlers) CurrentUser(r *http.Request) (User, bool) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return User{}, false
	}
	id, err := h.Sessions.Get(r.Context(), c.Value)
	if err != nil {
		return User{}, false
	}
	user, err := h.Users.ByID(r.Context(), id)
	return user, err == nil
}

func (h *Handlers) cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name: CookieName, Value: value, Path: "/", MaxAge: maxAge,
		HttpOnly: true, Secure: h.SecureCookie, SameSite: http.SameSiteLaxMode,
	}
}

func (h *Handlers) clientIP(r *http.Request) string {
	if h.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func normalizeDigits(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		fail(w, http.StatusBadRequest, "bad_request")
		return false
	}
	return true
}

func fail(w http.ResponseWriter, status int, code string) {
	reply(w, status, map[string]string{"error": code})
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
