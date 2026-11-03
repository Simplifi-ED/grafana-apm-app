package queries

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestValidateRequestURL(t *testing.T) {
	t.Parallel()
	ok, err := url.Parse("http://localhost:3000/api/ds/query?foo=1")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateRequestURL(ok); err != nil {
		t.Fatalf("expected ok: %v", err)
	}

	bad := &url.URL{Scheme: "http", Host: "evil", Path: "/a/../b"}
	if err := ValidateRequestURL(bad); err == nil {
		t.Fatal("expected traversal reject")
	}
	if err := ValidateRequestURL(nil); err == nil {
		t.Fatal("expected nil reject")
	}
}

func TestParseRequestURLRejectsEmptyAndUserinfo(t *testing.T) {
	t.Parallel()
	if _, err := ParseRequestURL(""); err == nil {
		t.Fatal("expected empty reject")
	}
	if _, err := ParseRequestURL("http://user:pass@localhost/x"); err == nil {
		t.Fatal("expected userinfo reject")
	}
}

func TestNewSafeRequestAndDoValidated(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	u, err := ParseRequestURL(srv.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	req, err := NewSafeRequest(context.Background(), http.MethodPost, u, strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoValidated(srv.Client(), req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestNewSafeRequestRejectsInvalid(t *testing.T) {
	t.Parallel()
	_, err := NewSafeRequest(context.Background(), http.MethodGet, &url.URL{Scheme: "ftp", Host: "x"}, nil)
	if err == nil {
		t.Fatal("expected reject")
	}
}
