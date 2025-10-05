package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Claims struct {
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

func parseClaims(jwt string) (*Claims, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}
	payloadBytes, err := decodePayloadSegment(parts[1])
	if err != nil {
		return nil, err
	}
	var c Claims
	if err := json.Unmarshal(payloadBytes, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")

		if !strings.HasPrefix(token, "Bearer ") {
			token = "Bearer " + strings.Trim(token, " ")
		}

		tokenWithOutBearer := strings.TrimPrefix(token, "Bearer ")

		claims, err := parseClaims(tokenWithOutBearer)

		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token payload"})
			return
		}

		if claims.UserID == 0 {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token payload"})
		}

		if !isTokenValid(token) {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
		}

		c.Set("user_id", claims.UserID)
		c.Set("access_token", token)
		c.Next()
	}
}

func isTokenValid(token string) bool {

	url := "https://api.metable.ir/api/users/profile/"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return false
	}

	return true
}
