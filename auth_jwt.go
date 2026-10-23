package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	jwtV4 "github.com/golang-jwt/jwt/v4"
)

type ConsumerInfo struct {
	OrgName string
	OrgUid  string
	OrgId   int
	Plan    string
}

type ConsumerClaims struct {
	jwtV4.RegisteredClaims
	TokenType string
	ConsumerInfo
}

// NewAuthJWT decodes the claims of a Bearer token into the "consumer" context key, as
// *ConsumerClaims. Requests without a Bearer token get empty claims; tokens that are not
// structurally valid JWTs are rejected with 401.
//
// The token signature, algorithm and expiry are NOT verified: only use it behind a component
// that already authenticated the request.
//
// Example: router.Use(NewAuthJWT())
//
// Deprecated: this middleware only decodes claims and will be removed in a future major
// version. Verify tokens with github.com/golang-jwt/jwt/v4 (or v5) and a known key instead.
func NewAuthJWT() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("consumer", &ConsumerClaims{})
		header := ctx.Request.Header.Get("Authorization")
		if header == "" || !strings.Contains(header, "Bearer") {
			ctx.Next()
			return
		}

		claims, ok := decodeConsumerClaims(header)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		ctx.Set("consumer", claims)
		ctx.Next()
	}
}

// decodeConsumerClaims reads the token after "Bearer" and decodes its payload without any
// verification, matching v1.1.x: claims that do not fit ConsumerClaims are kept partially filled,
// an undecodable payload yields empty claims and extra header fields are ignored. Only input that
// previously crashed the request (missing token, wrong segment count, invalid header segment)
// returns false.
func decodeConsumerClaims(header string) (*ConsumerClaims, bool) {
	fields := strings.Fields(header)
	if len(fields) < 2 {
		return nil, false
	}
	segments := strings.Split(fields[1], ".")
	if len(segments) != 3 || !isJSONSegment(segments[0]) {
		return nil, false
	}

	claims := &ConsumerClaims{}
	if payload, err := jwtV4.DecodeSegment(segments[1]); err == nil {
		// Decode errors are ignored on purpose: fields decoded before the error stay set.
		_ = json.Unmarshal(payload, claims)
	}
	return claims, true
}

func isJSONSegment(segment string) bool {
	raw, err := jwtV4.DecodeSegment(segment)
	return err == nil && json.Valid(raw)
}
