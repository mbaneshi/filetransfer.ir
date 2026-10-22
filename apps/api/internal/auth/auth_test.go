package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type captureSender struct{ phone, code string }

func (c *captureSender) SendCode(_ context.Context, phone, code string) error {
	c.phone, c.code = phone, code
	return nil
}

type memUsers map[string]User

func (m memUsers) UpsertByPhone(_ context.Context, phone string) (User, error) {
	u, ok := m[phone]
	if !ok {
		u = User{ID: "u-" + phone, Phone: phone}
		m[phone] = u
	}
	return u, nil
}

func (m memUsers) ByID(_ context.Context, id string) (User, error) {
	for _, u := range m {
		if u.ID == id {
			return u, nil
		}
	}
	return User{}, errors.New("not found")
}

func setup(t *testing.T) (*miniredis.Miniredis, *OTP, *captureSender, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	s := &captureSender{}
	return mr, NewOTP(rdb, s, []byte("test-secret")), s, rdb
}

func TestNormalizePhone(t *testing.T) {
	good := map[string]string{
		"09121234567":    "+989121234567",
		"+989121234567":  "+989121234567",
		"00989121234567": "+989121234567",
		"989121234567":   "+989121234567",
		"9121234567":     "+989121234567",
		"۰۹۱۲۱۲۳۴۵۶۷":    "+989121234567",
		"0912 123-4567":  "+989121234567",
		"٠٩١٢١٢٣٤٥٦٧":    "+989121234567",
	}
	for in, want := range good {
		if got, err := NormalizePhone(in); err != nil || got != want {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "0212345678", "0912123456", "09121234567890", "abc", "08121234567"} {
		if _, err := NormalizePhone(in); err == nil {
			t.Errorf("NormalizePhone(%q) accepted", in)
		}
	}
	if got := LocalFormat("+989121234567"); got != "09121234567" {
		t.Errorf("LocalFormat = %q", got)
	}
}

func TestOTPHappyPathIsSingleUse(t *testing.T) {
	_, otp, s, _ := setup(t)
	ctx := context.Background()
	phone := "+989121234567"

	if err := otp.Request(ctx, phone, "1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	if len(s.code) != 6 || s.phone != phone {
		t.Fatalf("sender got %q %q", s.phone, s.code)
	}
	if err := otp.Verify(ctx, phone, s.code); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := otp.Verify(ctx, phone, s.code); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("code reused: %v", err)
	}
}

func TestOTPExpires(t *testing.T) {
	mr, otp, s, _ := setup(t)
	ctx := context.Background()
	_ = otp.Request(ctx, "+989121234567", "1.1.1.1")
	mr.FastForward(codeTTL + time.Second)
	if err := otp.Verify(ctx, "+989121234567", s.code); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expired code accepted: %v", err)
	}
}

func TestOTPWrongAttemptsLockOut(t *testing.T) {
	_, otp, s, _ := setup(t)
	ctx := context.Background()
	phone := "+989121234567"
	_ = otp.Request(ctx, phone, "1.1.1.1")
	wrong := "000000"
	if s.code == wrong {
		wrong = "111111"
	}
	var err error
	for i := 0; i < maxVerifyTries; i++ {
		err = otp.Verify(ctx, phone, wrong)
	}
	if !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("last wrong attempt: %v", err)
	}
	if err := otp.Verify(ctx, phone, s.code); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("correct code accepted after lockout: %v", err)
	}
}

func TestOTPCooldownAndRateLimits(t *testing.T) {
	mr, otp, _, _ := setup(t)
	ctx := context.Background()
	phone := "+989121234567"

	if err := otp.Request(ctx, phone, "1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	if err := otp.Request(ctx, phone, "1.1.1.1"); !errors.Is(err, ErrCooldown) {
		t.Fatalf("resend inside cooldown: %v", err)
	}
	for i := 1; i < phoneMaxPerWin; i++ {
		mr.FastForward(resendCooldown + time.Second)
		if err := otp.Request(ctx, phone, "1.1.1.1"); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}
	mr.FastForward(resendCooldown + time.Second)
	if err := otp.Request(ctx, phone, "1.1.1.1"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("per-phone limit not enforced: %v", err)
	}
}

func TestOTPPerIPLimit(t *testing.T) {
	_, otp, _, _ := setup(t)
	ctx := context.Background()
	var err error
	for i := 0; i <= ipMaxPerWin; i++ {
		phone := "+98912" + strings.Repeat("0", 6) + string(rune('0'+i/10)) + string(rune('0'+i%10))
		err = otp.Request(ctx, phone, "9.9.9.9")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("per-IP limit not enforced: %v", err)
	}
}

func TestHTTPLoginFlow(t *testing.T) {
	_, otp, s, rdb := setup(t)
	h := &Handlers{OTP: otp, Sessions: NewSessions(rdb), Users: memUsers{}, SecureCookie: true}
	mux := http.NewServeMux()
	h.Register(mux)

	post := func(path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	if rec := post("/api/v1/auth/otp/request", `{"phone":"nope"}`, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad phone: %d", rec.Code)
	}
	if rec := post("/api/v1/auth/otp/request", `{"phone":"۰۹۱۲۱۲۳۴۵۶۷"}`, nil); rec.Code != http.StatusAccepted {
		t.Fatalf("request: %d %s", rec.Code, rec.Body)
	}
	wrong := "000000"
	if s.code == wrong {
		wrong = "111111"
	}
	if rec := post("/api/v1/auth/otp/verify", `{"phone":"09121234567","code":"`+wrong+`"}`, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code: %d", rec.Code)
	}

	rec := post("/api/v1/auth/otp/verify", `{"phone":"09121234567","code":"`+s.code+`"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie flags: %+v", cookies)
	}

	me := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	me.AddCookie(cookies[0])
	mrec := httptest.NewRecorder()
	mux.ServeHTTP(mrec, me)
	var out struct{ User User }
	if mrec.Code != http.StatusOK || json.Unmarshal(mrec.Body.Bytes(), &out) != nil || out.User.Phone != "+989121234567" {
		t.Fatalf("me: %d %s", mrec.Code, mrec.Body)
	}

	post("/api/v1/auth/logout", ``, cookies[0])
	again := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	again.AddCookie(cookies[0])
	arec := httptest.NewRecorder()
	mux.ServeHTTP(arec, again)
	if arec.Code != http.StatusUnauthorized {
		t.Fatalf("session survived logout: %d", arec.Code)
	}
}
