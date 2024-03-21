package database

import "github.com/google/uuid"

type File struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:(gen_random_uuid())"`
	User   string    `gorm:"index:,unique,composite:user_file"`
	Dir    string    `gorm:"index:,unique,composite:user_file"`
	Name   string    `gorm:"index:,unique,composite:user_file"`
	Chunks []*Chunk  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
