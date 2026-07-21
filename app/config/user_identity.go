package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ddp0_grader/app/models"
)

// DjangoUserProvider retrieves the authenticated user's identity from Django.
type DjangoUserProvider struct {
	meURL  string
	client *http.Client
}

type djangoMeResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func NewDjangoUserProviderFromEnv() (*DjangoUserProvider, error) {
	meURL := strings.TrimSpace(GetEnv("DJANGO_ME_URL"))
	if meURL == "" {
		return nil, fmt.Errorf("DJANGO_ME_URL is required")
	}
	return &DjangoUserProvider{meURL: meURL, client: &http.Client{Timeout: 3 * time.Second}}, nil
}

func (provider *DjangoUserProvider) GetUser(ctx context.Context, accessToken string) (models.User, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.meURL, nil)
	if err != nil {
		return models.User{}, fmt.Errorf("create Django me request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")

	response, err := provider.client.Do(request)
	if err != nil {
		return models.User{}, fmt.Errorf("request Django me: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return models.User{}, fmt.Errorf("Django me returned status %d", response.StatusCode)
	}

	var payload djangoMeResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return models.User{}, fmt.Errorf("decode Django me response: %w", err)
	}
	payload.ID = strings.TrimSpace(payload.ID)
	payload.Email = strings.TrimSpace(payload.Email)
	if payload.ID == "" || payload.Email == "" {
		return models.User{}, fmt.Errorf("Django me response must include id and email")
	}
	return models.User{ID: payload.ID, Email: payload.Email}, nil
}
