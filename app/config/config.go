package config

import (
	"context"
	"log"

	"ddp0_grader/app/models"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var RedisClient *redis.Client

func connectDatabase() {
	host := GetEnv("DB_HOST")
	port := GetEnv("DB_PORT")
	user := GetEnv("DB_USER")
	password := GetEnv("DB_PASSWORD")
	dbname := GetEnv("DB_NAME")

	dsn := "host=" + host + " user=" + user + " password=" + password + " dbname=" + dbname + " port=" + port + " sslmode=disable TimeZone=Asia/Shanghai"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	log.Println("Database connection established")
}

func connectRedis() {
	redisHost := GetEnv("REDIS_HOST")
	redisPort := GetEnv("REDIS_PORT")
	redisPassword := GetEnv("REDIS_PASSWORD")

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPassword,
		DB:       0,
	})

	_, err := RedisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}

	log.Println("Redis connection established")
}

func autoMigrate() {
	err := DB.AutoMigrate(
		&models.Problem{},
		&models.TestCase{},
		&models.Submission{},
		&models.TestCaseResult{},
	)
	if err != nil {
		log.Fatalf("Error during auto migration: %v", err)
	}

	log.Println("Database auto migration completed")
}

func InitConfig() {
	connectDatabase()
	autoMigrate()
	connectRedis()
}

// InitDatabase initializes only the database connection. It is useful for
// maintenance commands such as seeders that do not need Redis.
func InitDatabase() {
	connectDatabase()
	autoMigrate()
}
