package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	jwtV4 "github.com/golang-jwt/jwt/v4"
)

// authResult is what the protected handler saw: whether it ran and the consumer it received.
type authResult struct {
	called   bool
	consumer *ConsumerClaims
}

func runAuth(t *testing.T, authorization string) (int, authResult) {
	t.Helper()
	var result authResult
	router := gin.New()
	router.Use(NewAuthJWT())
	router.GET("/", func(c *gin.Context) {
		result.called = true
		result.consumer, _ = c.MustGet("consumer").(*ConsumerClaims)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder.Code, result
}

func signedToken(t *testing.T, claims *ConsumerClaims) string {
	t.Helper()
	token, err := jwtV4.NewWithClaims(jwtV4.SigningMethodHS256, claims).SignedString([]byte("test-key"))
	if err != nil {
		t.Fatalf("signing token: %v", err)
	}
	return token
}

func TestAuthJWTDecodesClaims(t *testing.T) {
	token := signedToken(t, &ConsumerClaims{TokenType: "client", ConsumerInfo: ConsumerInfo{OrgUid: "org-1", OrgId: 7}})

	code, result := runAuth(t, "Bearer "+token)

	if code != http.StatusOK || !result.called {
		t.Fatalf("status = %d, handler called = %v; want 200 and called", code, result.called)
	}
	if result.consumer.OrgUid != "org-1" || result.consumer.OrgId != 7 || result.consumer.TokenType != "client" {
		t.Fatalf("decoded claims = %+v, want OrgUid org-1, OrgId 7, TokenType client", result.consumer)
	}
}

func TestAuthJWTWithoutTokenSetsEmptyClaims(t *testing.T) {
	code, result := runAuth(t, "")

	if code != http.StatusOK || result.consumer == nil || result.consumer.OrgUid != "" {
		t.Fatalf("status = %d, consumer = %+v; want 200 and empty claims", code, result.consumer)
	}
}

func TestAuthJWTRejectsMalformedTokens(t *testing.T) {
	for _, header := range []string{"Bearer", "Bearer not-a-jwt", "Bearer a.b.c", "Bearer x y", "Bearer e30.e30"} {
		code, result := runAuth(t, header)
		if code != http.StatusUnauthorized || result.called {
			t.Fatalf("header %q: status = %d, handler called = %v; want 401 and not called", header, code, result.called)
		}
	}
}

// rawToken builds an unsigned token from literal JSON segments, to reproduce inputs that
// v1.1.x accepted.
func rawToken(header, payload string) string {
	enc := base64.RawURLEncoding
	return enc.EncodeToString([]byte(header)) + "." + enc.EncodeToString([]byte(payload)) + ".sig"
}

// Inputs accepted by v1.1.x must keep being accepted with the same (possibly partial) claims.
func TestAuthJWTKeepsLegacyLeniency(t *testing.T) {
	cases := []struct {
		name, header, wantOrgUid string
	}{
		{"claim with unexpected type", "Bearer " + rawToken(`{"alg":"HS256"}`, `{"OrgUid":"u","OrgId":"12"}`), "u"},
		{"header without alg", "Bearer " + rawToken(`{"typ":"JWT"}`, `{"OrgUid":"u"}`), "u"},
		{"unknown alg", "Bearer " + rawToken(`{"alg":"XYZ"}`, `{"OrgUid":"u"}`), "u"},
		{"extra header fields", "Bearer " + rawToken(`{"alg":"none"}`, `{"OrgUid":"u"}`) + " extra", "u"},
		{"undecodable payload", "Bearer " + base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + ".%%%.sig", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, result := runAuth(t, tc.header)
			if code != http.StatusOK || result.consumer == nil || result.consumer.OrgUid != tc.wantOrgUid {
				t.Fatalf("status = %d, consumer = %+v; want 200 and OrgUid %q", code, result.consumer, tc.wantOrgUid)
			}
		})
	}
}
