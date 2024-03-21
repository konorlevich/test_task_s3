package database

import (
	"errors"

	"github.com/mattn/go-sqlite3"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrRecordNotFound = errors.New("record not found")
var ErrDuplicated = errors.New("record duplicated")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AddServer(name, port string) (uuid.UUID, error) {
	s := &Server{Name: name, Port: port}

	return s.ID, checkError(r.db.Create(s).Error)
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

	return f.ID, checkError(r.db.Save(f).Error)
}

func (r *Repository) GetFile(username, dir, name string) (*File, error) {
	c := &File{}
	err := r.db.
		Preload("Chunks").
		Preload("Chunks.Server").
		Preload("Chunks.File").
		First(c, &File{User: username, Dir: dir, Name: name}).Error

	return c, checkError(err)
}

func (r *Repository) RemoveFile(username, dir, name string) error {
	return checkError(r.db.Delete(&File{}, &File{User: username, Dir: dir, Name: name}).Error)
}

func (r *Repository) SaveChunk(file uuid.UUID, server uuid.UUID, number uint) (uuid.UUID, error) {
	c := &Chunk{
		Number:   number,
		ServerID: server,
		FileID:   file,
	}

	return c.ID, checkError(r.db.Save(c).Error)
}

func checkError(err error) error {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code {
		case sqlite3.ErrNotFound:
			return ErrRecordNotFound
		case sqlite3.ErrConstraint:
			return ErrDuplicated
		}
	}
	return err
}
