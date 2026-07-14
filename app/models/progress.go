package models

type Progress struct {
	ID          string       `gorm:"primaryKey" json:"id"`
	ProblemID   string       `gorm:"uniqueIndex:idx_problem_user, priority:1; not null" json:"problem_id"`
	UserID      string       `gorm:"uniqueIndex:idx_problem_user, priority:2; not null" json:"user_id"`
	BestScore   int          `gorm:"default:0" json:"best_score"`
	Problem     Problem      `gorm:"foreignKey:ProblemID;references:ID" json:"-"`
	User        User         `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Submissions []Submission `gorm:"foreignKey:ProgressID; constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"submissions"`
}
