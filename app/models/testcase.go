package models

import (
	"ddp0_grader/app/config"
	"log"
)

type TestCase struct {
	ID        string `gorm:"primaryKey" json:"id"`
	ProblemID string `gorm:"not null" json:"problem_id"`
	Input     string `gorm:"not null" json:"input"`
	Output    string `gorm:"not null" json:"output"`
	IsHidden  bool   `gorm:"default:true" json:"is_hidden"`
	CreatedAt string `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt string `gorm:"autoUpdateTime" json:"updated_at"`
}

func (t *TestCase) Save() error {
	if err := config.DB.Save(t).Error; err != nil {
		log.Printf("Error saving test case: %v", err)
		return err
	}
	return nil
}

func (t *TestCase) Delete() error {
	if err := config.DB.Delete(t).Error; err != nil {
		log.Printf("Error deleting test case: %v", err)
		return err
	}
	return nil
}

func GetTestCasesByProblemID(problemID string) ([]TestCase, error) {
	var testCases []TestCase
	if err := config.DB.Where("problem_id = ?", problemID).Find(&testCases).Error; err != nil {
		log.Printf("Error retrieving test cases by problem ID: %v", err)
		return nil, err
	}
	return testCases, nil
}
