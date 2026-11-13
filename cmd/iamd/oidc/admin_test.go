package oidc

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminRequiresVerifiedAdmin(t *testing.T) {
	s := testServer(t)
	for _, path := range []string{"/admin/resources", "/admin/clients", "/admin/identities", "/admin/keys/rotate"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader(`{}`)))
			if w.Code != 403 {
				t.Fatalf("status = %d, want forbidden", w.Code)
			}
		})
	}
}
