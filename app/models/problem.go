package models

import (
	"log"

	"ddp0_grader/app/config"
)

type Problem struct {
	ID          string     `gorm:"primaryKey" json:"id"`
	Title       string     `gorm:"not null" json:"title"`
	Description string     `gorm:"not null" json:"description"`
	TimeLimit   int        `gorm:"not null" json:"time_limit"`
	MemoryLimit int        `gorm:"not null" json:"memory_limit"`
	CreatedAt   string     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   string     `gorm:"autoUpdateTime" json:"updated_at"`
	TestCases   []TestCase `gorm:"foreignKey:ProblemID" json:"test_cases"`
}

func (p *Problem) Save() error {
	if err := config.DB.Save(p).Error; err != nil {
		log.Printf("Error saving problem: %v", err)
		return err
	}
	return nil
}

func (p *Problem) Delete() error {
	if err := config.DB.Delete(p).Error; err != nil {
		log.Printf("Error deleting problem: %v", err)
		return err
	}
	return nil
}

func GetProblemByID(id string) (*Problem, error) {
	var problem Problem
	if err := config.DB.First(&problem, "id = ?", id).Error; err != nil {
		log.Printf("Error retrieving problem by ID: %v", err)
		return nil, err
	}
	return &problem, nil
}

func GetAllProblems() ([]Problem, error) {
	var problems []Problem
	if err := config.DB.Find(&problems).Error; err != nil {
		log.Printf("Error retrieving all problems: %v", err)
		return nil, err
	}
	return problems, nil
}
