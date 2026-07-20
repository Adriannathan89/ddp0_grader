// Command migrate-users imports users from a JSON file into the grader database.
//
// The input must be a JSON array matching models.User. Fields not used by the
// user table (such as progresses) are ignored. Example:
// [
//
//	{"id":"user-1","email":"user@example.com","score":0}
//
// ]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"ddp0_grader/app/config"
	"ddp0_grader/app/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func loadUsers(path string, now time.Time) ([]models.User, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open users JSON: %w", err)
	}
	defer file.Close()

	var users []models.User
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&users); err != nil {
		return nil, fmt.Errorf("decode users JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("users JSON must contain exactly one array")
		}
		return nil, fmt.Errorf("read users JSON: %w", err)
	}

	ids := make(map[string]struct{}, len(users))
	emails := make(map[string]struct{}, len(users))
	for index := range users {
		user := &users[index]
		// Progresses belong to a different import and must never be inserted as
		// a side effect of migrating users.
		user.Progresses = nil
		user.ID = strings.TrimSpace(user.ID)
		user.Email = strings.TrimSpace(user.Email)
		if user.ID == "" {
			return nil, fmt.Errorf("user at index %d has an empty id", index)
		}
		if user.Email == "" {
			return nil, fmt.Errorf("user at index %d has an empty email", index)
		}
		if _, exists := ids[user.ID]; exists {
			return nil, fmt.Errorf("duplicate user id %q", user.ID)
		}
		if _, exists := emails[user.Email]; exists {
			return nil, fmt.Errorf("duplicate user email %q", user.Email)
		}
		ids[user.ID] = struct{}{}
		emails[user.Email] = struct{}{}
		if user.CreatedAt.IsZero() {
			user.CreatedAt = now
		}
		if user.UpdatedAt.IsZero() {
			user.UpdatedAt = now
		}
	}

	return users, nil
}

func migrateUsers(db *gorm.DB, users []models.User) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"email", "score", "created_at", "updated_at"}),
		}).Create(&users).Error
	})
}

func main() {
	filePath := flag.String("file", "users.json", "path to a JSON array of users")
	flag.Parse()

	users, err := loadUsers(*filePath, time.Now().UTC())
	if err != nil {
		log.Fatal(err)
	}

	config.InitDatabase()
	if err := migrateUsers(config.DB, users); err != nil {
		log.Fatalf("migrate users: %v", err)
	}
	log.Printf("migrated %d users from %s", len(users), *filePath)
}
