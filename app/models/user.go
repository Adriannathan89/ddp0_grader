package models

import "time"

type User struct {
	ID         string     `gorm:"primaryKey" json:"id"`
	Email      string     `gorm:"unique;not null" json:"email"`
	Score      int        `gorm:"default:0" json:"score"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Progresses []Progress `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"progresses"`
}
