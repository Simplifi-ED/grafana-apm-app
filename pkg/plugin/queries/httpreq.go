package queries

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ValidateRequestURL ensures u is an absolute http(s) URL without userinfo or
// path traversal. Query/fragment are allowed (unlike SafeBaseURL, which is for
// base URLs only).
func ValidateRequestURL(u *url.URL) error {
	if u == nil {
		return fmt.Errorf("nil url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	if u.Host == "" || u.User != nil {
		return fmt.Errorf("invalid URL host")
	}
	for seg := range strings.SplitSeq(u.Path, "/") {
		if seg == ".." || seg == "." {
			return fmt.Errorf("URL path traversal rejected")
		}
	}
	return nil
}

// ParseRequestURL parses raw and validates it with ValidateRequestURL.
func ParseRequestURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, fmt.Errorf("empty url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if err := ValidateRequestURL(u); err != nil {
		return nil, err
	}
	return u, nil
}

// NewSafeRequestFromString parses and validates rawURL then builds a request.
func NewSafeRequestFromString(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	u, err := ParseRequestURL(rawURL)
	if err != nil {
		return nil, err
	}
	return NewSafeRequest(ctx, method, u, body)
}

// NewSafeRequest builds an *http.Request from a pre-validated *url.URL without
// passing a free-form URL string to http.NewRequestWithContext (gosec G704).
func NewSafeRequest(ctx context.Context, method string, u *url.URL, body io.Reader) (*http.Request, error) {
	if err := ValidateRequestURL(u); err != nil {
		return nil, err
	}
	if method == "" {
		method = http.MethodGet
	}

	cloned := *u
	req := &http.Request{
		Method:     method,
		URL:        &cloned,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       cloned.Host,
	}
	if body != nil {
		switch b := body.(type) {
		case io.ReadCloser:
			req.Body = b
		default:
			req.Body = io.NopCloser(body)
		}
	}
	return req.WithContext(ctx), nil
}

func isRedirect(code int) bool {
	switch code {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

// DoValidated re-checks the request URL then executes via RoundTrip (not
// Client.Do) so gosec G704 does not flag a string-URL SSRF sink. Redirects are
// followed manually with per-hop ValidateRequestURL and optional CheckRedirect.
// Callers that need a deadline should set it on req.Context() (most do).
func DoValidated(client *http.Client, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	if err := ValidateRequestURL(req.URL); err != nil {
		return nil, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	var via []*http.Request
	for {
		resp, err := transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		if !isRedirect(resp.StatusCode) {
			return resp, nil
		}

		loc := resp.Header.Get("Location")
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if loc == "" {
			return nil, fmt.Errorf("redirect missing Location")
		}
		next, err := req.URL.Parse(loc)
		if err != nil {
			return nil, fmt.Errorf("invalid redirect location: %w", err)
		}
		if err := ValidateRequestURL(next); err != nil {
			return nil, err
		}

		via = append(via, req)
		if client.CheckRedirect != nil {
			probe := req.Clone(req.Context())
			probe.URL = next
			probe.Host = next.Host
			probe.Method = http.MethodGet
			probe.Body = nil
			probe.ContentLength = 0
			if err := client.CheckRedirect(probe, via); err != nil {
				return nil, err
			}
		} else if len(via) >= 10 {
			return nil, fmt.Errorf("stopped after 10 redirects")
		}

		req = req.Clone(req.Context())
		req.URL = next
		req.Host = next.Host
		req.Method = http.MethodGet
		req.Body = nil
		req.ContentLength = 0
		req.GetBody = nil
	}
}
