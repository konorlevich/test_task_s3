package database

import (
	"fmt"

	"github.com/google/uuid"
)

type Server struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:(gen_random_uuid())"`
	Name string
	Port string
}

func (s *Server) GetID() uuid.UUID {
	return s.ID
}

func (s *Server) GetUrl() string {
	return fmt.Sprintf("http://%s:%s/", s.Name, s.Port)
}
