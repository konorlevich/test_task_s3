package database

import "github.com/google/uuid"

type File struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:(gen_random_uuid())"`
	User   string    `gorm:"index:,unique,composite:user_path"`
	Dir    string    `gorm:"index:,unique,composite:user_path"`
	Name   string    `gorm:"index:,unique,composite:user_path"`
	Chunks []*Chunk  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
