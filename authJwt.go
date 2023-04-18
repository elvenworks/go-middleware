package middleware

import (
	"crypto/rsa"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
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

var (
	verifyKey *rsa.PublicKey
)

func NewAuthJWT() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("consumer", &ConsumerClaims{})
		clientToken := ctx.Request.Header.Get("Authorization")
		if clientToken != "" && strings.Contains(clientToken, "Bearer") {
			tokenString := strings.Split(clientToken, " ")

			token, _ := jwt.ParseWithClaims(tokenString[1], &ConsumerClaims{}, func(token *jwt.Token) (interface{}, error) {
				return verifyKey, nil
			})

			claims := token.Claims.(*ConsumerClaims)
			ctx.Set("consumer", claims)
		}

		ctx.Next()
	}
}
