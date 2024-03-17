package database

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AddServer(name, port string) (uuid.UUID, error) {
	s := &Server{Name: name, Port: port}
	tx := r.db.Create(s)
	return s.ID, tx.Error
}

func (r *Repository) GetLeastLoadedServers(num int) ([]*Server, error) {
	var res []*Server
	tx := r.db.
		Model(&Server{}).
		Select("servers.id,servers.name,servers.port, count(chunks.id) as chunk_count").
		Joins("left join chunks on servers.id = chunks.server_id").
		Group("servers.id").
		Order(clause.OrderByColumn{Column: clause.Column{Name: "chunk_count"}, Desc: false}).
		Limit(num).
		Find(&res)
	return res, tx.Error
}

func (r *Repository) SaveFile(user, dir, name string) (uuid.UUID, error) {
	f := &File{
		User: user,
		Dir:  dir,
		Name: name,
	}
	return f.ID, r.db.Save(f).Error
}

func (r *Repository) GetFile(username, dir, name string) (*File, error) {
	c := &File{}
	return c, r.db.
		Preload("Chunks").
		Preload("Chunks.Server").
		Preload("Chunks.File").
		First(c, &File{User: username, Dir: dir, Name: name}).Error
}

func (r *Repository) RemoveFile(username, dir, name string) error {
	return r.db.Delete(&File{}, &File{User: username, Dir: dir, Name: name}).Error
}

func (r *Repository) SaveChunk(file uuid.UUID, server uuid.UUID, number uint) (uuid.UUID, error) {
	c := &Chunk{
		Number:   number,
		ServerID: server,
		FileID:   file,
	}
	return c.ID, r.db.Save(c).Error
}
