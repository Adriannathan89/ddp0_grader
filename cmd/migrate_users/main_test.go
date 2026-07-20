package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadUsersDefaultsTimestamps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.json")
	if err := os.WriteFile(path, []byte(`[{"id":" user-1 ","email":" user@example.com ","score":12}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)

	users, err := loadUsers(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].ID != "user-1" || users[0].Email != "user@example.com" || users[0].Score != 12 {
		t.Fatalf("unexpected user: %#v", users)
	}
	if !users[0].CreatedAt.Equal(now) || !users[0].UpdatedAt.Equal(now) {
		t.Fatalf("timestamps were not defaulted: %#v", users[0])
	}
}

func TestLoadUsersRejectsDuplicateID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.json")
	if err := os.WriteFile(path, []byte(`[{"id":"one","email":"one@example.com"},{"id":"one","email":"two@example.com"}]`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := loadUsers(path, time.Now())
	if err == nil || !strings.Contains(err.Error(), "duplicate user id") {
		t.Fatalf("expected duplicate-id error, got %v", err)
	}
}
