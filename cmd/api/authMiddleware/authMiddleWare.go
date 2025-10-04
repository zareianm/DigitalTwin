package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UnsafeClaims struct {
	TokenType string `json:"token_type"`
	Exp       int64  `json:"exp"`
	Iat       int64  `json:"iat"`
	Jti       string `json:"jti"`
	UserID    int    `json:"user_id"`
}

func decodePayloadSegment(seg string) ([]byte, error) {
	// JWT uses base64url without padding
	return base64.RawURLEncoding.DecodeString(seg)
}

func parseUnsafeClaims(jwt string) (*UnsafeClaims, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}
	payloadBytes, err := decodePayloadSegment(parts[1])
	if err != nil {
		return nil, err
	}
	var c UnsafeClaims
	if err := json.Unmarshal(payloadBytes, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// AuthUnsafe parses the token WITHOUT verifying the signature.
// It only checks exp and then puts user_id in Gin context.
func AuthUnsafe() gin.HandlerFunc {
	return func(c *gin.Context) {
		authz := c.GetHeader("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(authz, "Bearer ")

		claims, err := parseUnsafeClaims(token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token payload"})
			return
		}

		// Minimal safety: reject expired tokens
		now := time.Now().Unix()
		if claims.Exp != 0 && now >= claims.Exp {
			c.AbortWithStatusJSON(401, gin.H{"error": "token expired"})
			return
		}

		if claims.UserID == 0 {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token payload"})
		}

		// DEV ONLY: we did not verify the signature!
		// Expose user_id to handlers
		c.Set("user_id", claims.UserID)
		c.Set("access_token", token)
		c.Next()
	}
}
