package config_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ddp0_grader/app/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testSigningKey = "test-signing-key"

func TestJWTAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	middleware, err := config.NewJWTAuthMiddleware(config.JWTAuthConfig{SigningKey: testSigningKey})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/protected", middleware, func(c *gin.Context) {
		userID, ok := config.AuthenticatedUserID(c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.String(http.StatusOK, userID)
	})

	valid := signedToken(t, jwt.MapClaims{"user_id": 42, "token_type": "access", "exp": time.Now().Add(time.Minute).Unix()})
	if response := authenticatedRequest(router, valid); response.Code != http.StatusOK || response.Body.String() != "42" {
		t.Fatalf("valid access token = (%d, %q), want (200, 42)", response.Code, response.Body.String())
	}

	refresh := signedToken(t, jwt.MapClaims{"user_id": "user-1", "token_type": "refresh", "exp": time.Now().Add(time.Minute).Unix()})
	if response := authenticatedRequest(router, refresh); response.Code != http.StatusUnauthorized {
		t.Fatalf("refresh token status = %d, want 401", response.Code)
	}

	expired := signedToken(t, jwt.MapClaims{"user_id": "user-1", "token_type": "access", "exp": time.Now().Add(-time.Minute).Unix()})
	if response := authenticatedRequest(router, expired); response.Code != http.StatusUnauthorized {
		t.Fatalf("expired token status = %d, want 401", response.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want 401", response.Code)
	}
}

func TestJWTAuthMiddlewareRejectsMissingSigningKey(t *testing.T) {
	if _, err := config.NewJWTAuthMiddleware(config.JWTAuthConfig{}); err == nil {
		t.Fatal("NewJWTAuthMiddleware() error = nil, want missing signing key error")
	}
}

func signedToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSigningKey))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func authenticatedRequest(router http.Handler, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
