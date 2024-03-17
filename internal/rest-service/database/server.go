package database

import "github.com/google/uuid"

type Server struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:(gen_random_uuid())"`
	Name string
	Port string
}
