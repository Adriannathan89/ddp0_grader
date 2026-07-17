package config_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ddp0_grader/app/config"

	"github.com/gin-gonic/gin"
)

func TestDjangoAdminMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       int
	}{
		{name: "django staff user", statusCode: http.StatusOK, body: `{"is_staff":true}`, want: http.StatusOK},
		{name: "non staff user", statusCode: http.StatusOK, body: `{"is_staff":false}`, want: http.StatusForbidden},
		{name: "django denies token", statusCode: http.StatusForbidden, body: `{}`, want: http.StatusForbidden},
		{name: "django unavailable", statusCode: http.StatusBadGateway, body: `{}`, want: http.StatusServiceUnavailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
				if got := r.Header.Get("Authorization"); got != "Bearer verified-token" {
					t.Fatalf("Authorization = %q", got)
				}
				return &http.Response{StatusCode: test.statusCode, Body: io.NopCloser(strings.NewReader(test.body)), Header: make(http.Header)}, nil
			})}
			middleware, err := config.NewDjangoAdminMiddleware(config.DjangoAdminAuthConfig{MeURL: "http://django.test/api/admin/me/", Client: client})
			if err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			router.GET("/write", func(c *gin.Context) {
				c.Set(config.AuthAccessTokenContextKey, "verified-token")
			}, middleware, func(c *gin.Context) { c.Status(http.StatusOK) })

			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/write", nil))
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.want, response.Body.String())
			}
		})
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (fn roundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestDjangoAdminMiddlewareRequiresURL(t *testing.T) {
	if _, err := config.NewDjangoAdminMiddleware(config.DjangoAdminAuthConfig{}); err == nil {
		t.Fatal("NewDjangoAdminMiddleware() error = nil, want missing URL error")
	}
}
