package repository

import (
	"ddp0_grader/app/models"
	"log"

	"gorm.io/gorm"
)

type UserRepository interface {
	GetUserByID(id string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	SaveUser(user *models.User) error
	DeleteUser(user *models.User) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db}
}

func (r *userRepository) GetUserByID(id string) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving user by ID: %v", err)
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, "email = ?", email).Error; err != nil {
		log.Printf("Error retrieving user by email: %v", err)
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) SaveUser(user *models.User) error {
	if err := r.db.Save(user).Error; err != nil {
		log.Printf("Error saving user: %v", err)
		return err
	}
	return nil
}

func (r *userRepository) DeleteUser(user *models.User) error {
	if err := r.db.Delete(user).Error; err != nil {
		log.Printf("Error deleting user: %v", err)
		return err
	}
	return nil
}
