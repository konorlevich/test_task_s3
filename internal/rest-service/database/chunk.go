package database

import uuid "github.com/google/uuid"

type Chunk struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:(gen_random_uuid())"`
	FileID   uuid.UUID `gorm:"index:,unique,composite:file_chunk"`
	File     File
	ServerID uuid.UUID
	Server   *Server
	Number   uint `gorm:"index:,unique,composite:file_chunk"`
}
