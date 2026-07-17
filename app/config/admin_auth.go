package config

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// DjangoAdminAuthConfig points to Django's existing admin identity endpoint.
// The grader never accepts an admin flag supplied by a browser or JWT claim.
type DjangoAdminAuthConfig struct {
	MeURL  string
	Client *http.Client
}

type djangoAdminMeResponse struct {
	IsStaff bool `json:"is_staff"`
}

func NewDjangoAdminMiddlewareFromEnv() (gin.HandlerFunc, error) {
	return NewDjangoAdminMiddleware(DjangoAdminAuthConfig{
		MeURL:  GetEnv("DJANGO_ADMIN_ME_URL"),
		Client: &http.Client{Timeout: 3 * time.Second},
	})
}

func NewDjangoAdminMiddleware(cfg DjangoAdminAuthConfig) (gin.HandlerFunc, error) {
	cfg.MeURL = strings.TrimSpace(cfg.MeURL)
	if cfg.MeURL == "" {
		return nil, errors.New("DJANGO_ADMIN_ME_URL is required")
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 3 * time.Second}
	}

	return func(c *gin.Context) {
		token, ok := AuthenticatedAccessToken(c)
		if !ok {
			abortUnauthorized(c)
			return
		}

		request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, cfg.MeURL, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "admin authorization service unavailable"})
			return
		}
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Accept", "application/json")

		response, err := cfg.Client.Do(request)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "admin authorization service unavailable"})
			return
		}
		defer response.Body.Close()

		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "django administrator access required"})
			return
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "admin authorization service unavailable"})
			return
		}

		var identity djangoAdminMeResponse
		if err := json.NewDecoder(response.Body).Decode(&identity); err != nil || !identity.IsStaff {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "django administrator access required"})
			return
		}

		c.Next()
	}, nil
}
