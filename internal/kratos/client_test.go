// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package kratos

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExecuteIdentifierFirstUpdateLoginRequest(t *testing.T) {
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)

	var mux http.ServeMux
	server := httptest.NewServer(&mux)
	t.Cleanup(server.Close)

	redirectTo := server.URL + "/ui/login?flow=flow123"
	mux.HandleFunc("POST /self-service/login", func(w http.ResponseWriter, req *http.Request) {
		if ct := req.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Fatalf("expected form content type, got %q", ct)
		}
		if got := req.URL.Query().Get("flow"); got != "flow123" {
			t.Fatalf("expected flow query flow123, got %q", got)
		}
		if err := req.ParseForm(); err != nil {
			t.Fatalf("expected form body: %v", err)
		}
		if req.PostForm.Get("csrf_token") != "csrf_token_1234" || req.PostForm.Get("identifier") != "test@example.com" || req.PostForm.Get("method") != "identifier_first" {
			t.Fatalf("unexpected form values: %v", req.PostForm)
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, req, redirectTo, http.StatusSeeOther)
	})
	mux.HandleFunc("GET /ui/login", func(w http.ResponseWriter, req *http.Request) {
		t.Fatalf("redirect must not be followed")
	})

	client := NewClient(server.URL, false)

	resp, err := client.ExecuteIdentifierFirstUpdateLoginRequest(t.Context(), "flow123", "csrf_token_1234", "test@example.com", cookies)
	if err != nil {
		t.Fatalf("expected error to be nil not %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("expected status 303, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != redirectTo {
		t.Errorf("expected location %q, got %q", redirectTo, got)
	}
	if len(resp.Cookies()) != 1 || resp.Cookies()[0].Name != cookie.Name {
		t.Errorf("expected response cookie %q, got %v", cookie.Name, resp.Cookies())
	}
}
