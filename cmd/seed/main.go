package main

import (
	"log"

	"ddp0_grader/app/config"
	"ddp0_grader/app/models"

	"gorm.io/gorm"
)

const helloWorldProblemID = "hello-world"

func main() {
	config.InitDatabase()

	problem := models.Problem{
		ID:          helloWorldProblemID,
		Title:       "Hello World",
		Description: "Print exactly: Hello, World!",
		Author:      "system",
		Tag:         models.TagOperational,
		Difficulty:  models.DifficultyEasy,
		TimeLimit:   1000,
		MemoryLimit: 64,
	}

	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", problem.ID).Assign(problem).FirstOrCreate(&problem).Error; err != nil {
			return err
		}

		testCases := []models.TestCase{
			{ID: "hello-world-1", ProblemID: problem.ID, Input: "", Output: "Hello, World!\n", IsHidden: false},
			{ID: "hello-world-2", ProblemID: problem.ID, Input: "ignored input\n", Output: "Hello, World!\n", IsHidden: true},
		}
		for _, testCase := range testCases {
			if err := tx.Where("id = ?", testCase.ID).Assign(testCase).FirstOrCreate(&testCase).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Fatalf("seed hello world failed: %v", err)
	}

	log.Printf("seeded problem %q with 2 test cases", helloWorldProblemID)
}
