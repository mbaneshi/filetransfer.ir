package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKavenegarSendsLookupRequest(t *testing.T) {
	var path, query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.RawQuery
	}))
	defer srv.Close()

	k := Kavenegar{APIKey: "KEY", Template: "verify", BaseURL: srv.URL}
	if err := k.SendCode(context.Background(), "+989121234567", "123456"); err != nil {
		t.Fatal(err)
	}
	if path != "/v1/KEY/verify/lookup.json" {
		t.Errorf("path = %s", path)
	}
	for _, want := range []string{"receptor=09121234567", "token=123456", "template=verify"} {
		if !strings.Contains(query, want) {
			t.Errorf("query %q missing %q", query, want)
		}
	}
}

func TestKavenegarErrorDoesNotLeakSecrets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	k := Kavenegar{APIKey: "SECRETKEY", Template: "verify", BaseURL: srv.URL}
	err := k.SendCode(context.Background(), "+989121234567", "654321")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "SECRETKEY") || strings.Contains(err.Error(), "654321") {
		t.Fatalf("error leaks secrets: %v", err)
	}
}
