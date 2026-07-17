package config

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const AuthUserIDContextKey = "auth_user_id"
const AuthAccessTokenContextKey = "auth_access_token"

type JWTAuthConfig struct {
	SigningKey      string
	Issuer          string
	Audience        string
	UserIDClaim     string
	TokenTypeClaim  string
	AccessTokenType string
}

func NewJWTAuthMiddlewareFromEnv() (gin.HandlerFunc, error) {
	return NewJWTAuthMiddleware(JWTAuthConfig{
		SigningKey:      GetEnv("AUTH_JWT_SECRET"),
		Issuer:          GetEnv("AUTH_JWT_ISSUER"),
		Audience:        GetEnv("AUTH_JWT_AUDIENCE"),
		UserIDClaim:     defaultString(GetEnv("AUTH_JWT_USER_ID_CLAIM"), "user_id"),
		TokenTypeClaim:  defaultString(GetEnv("AUTH_JWT_TOKEN_TYPE_CLAIM"), "token_type"),
		AccessTokenType: defaultString(GetEnv("AUTH_JWT_ACCESS_TOKEN_TYPE"), "access"),
	})
}

func NewJWTAuthMiddleware(config JWTAuthConfig) (gin.HandlerFunc, error) {
	config.SigningKey = strings.TrimSpace(config.SigningKey)
	if config.SigningKey == "" {
		return nil, errors.New("AUTH_JWT_SECRET is required")
	}
	config.UserIDClaim = defaultString(config.UserIDClaim, "user_id")
	config.TokenTypeClaim = defaultString(config.TokenTypeClaim, "token_type")
	config.AccessTokenType = defaultString(config.AccessTokenType, "access")

	parserOptions := []jwt.ParserOption{jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})}
	if config.Issuer != "" {
		parserOptions = append(parserOptions, jwt.WithIssuer(config.Issuer))
	}
	if config.Audience != "" {
		parserOptions = append(parserOptions, jwt.WithAudience(config.Audience))
	}
	parser := jwt.NewParser(parserOptions...)

	return func(c *gin.Context) {
		rawToken, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			abortUnauthorized(c)
			return
		}

		claims := jwt.MapClaims{}
		token, err := parser.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method %q", token.Method.Alg())
			}
			return []byte(config.SigningKey), nil
		})
		if err != nil || token == nil || !token.Valid || claims[config.TokenTypeClaim] != config.AccessTokenType {
			fmt.Println("Invalid token or claims:", err, claims)
			abortUnauthorized(c)
			return
		}

		userID, ok := claimString(claims[config.UserIDClaim])
		if !ok {
			fmt.Println("User ID claim not found or invalid:", claims[config.UserIDClaim])
			abortUnauthorized(c)
			return
		}
		c.Set(AuthUserIDContextKey, userID)
		// Keep the already verified token available for downstream services that
		// need to ask the identity provider for server-side authorization facts.
		c.Set(AuthAccessTokenContextKey, rawToken)
		c.Next()
	}, nil
}

func AuthenticatedAccessToken(c *gin.Context) (string, bool) {
	token, ok := c.Get(AuthAccessTokenContextKey)
	value, isString := token.(string)
	return value, ok && isString && value != ""
}

func AuthenticatedUserID(c *gin.Context) (string, bool) {
	userID, ok := c.Get(AuthUserIDContextKey)
	value, isString := userID.(string)
	return value, ok && isString && value != ""
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func claimString(value any) (string, bool) {
	switch value := value.(type) {
	case string:
		value = strings.TrimSpace(value)
		return value, value != ""
	case float64:
		if value == float64(int64(value)) {
			return fmt.Sprintf("%d", int64(value)), true
		}
	}
	return "", false
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing access token"})
}
