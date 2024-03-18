package database

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setup() *Repository {
	db, err := NewDb("test.db")
	if err != nil {
		log.WithError(err).Fatalf("failed to connect database")
	}

	_ = db.AutoMigrate(&Server{}, &Chunk{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Server{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&File{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Chunk{})
	return NewRepository(db)
}

func TestRepository_AddServerGetServer(t *testing.T) {
	repo := setup()
	tests := []struct {
		name    string
		port    string
		wantErr bool
	}{
		{"TestServer1", "8080", false},
		{"TestServer2", "9090", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.AddServer(tt.name, tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddServer() error = %v, wantErr %v", err, tt.wantErr)
			}
			t.Run("check saved server", func(t *testing.T) {
				res := &Server{}
				tx := repo.db.Find(res, &Server{
					Name: tt.name,
					Port: tt.port,
				})
				if tx.Error != nil {
					t.Errorf("can't get saved server: %s", tx.Error)
				}
			})

		})
	}
}

func TestRepository_GetLeastLoadedServer(t *testing.T) {
	repo := setup()
	saved := make([]uuid.UUID, 0, 6)
	for i := 0; i < 8; i++ {
		server, err := repo.AddServer(fmt.Sprintf("GetLeastLoadedServer%d", i), "123")
		if err != nil {
			t.Fatalf("can't save server: %s", err)
		}
		saved = append(saved, server)
		file, err := repo.SaveFile("username_GetLeastLoadedServer", "dir_GetLeastLoadedServer", fmt.Sprintf("GetLeastLoadedServer_%d", i))
		if err != nil {
			t.Fatalf("can't save file: %s", err)
		}
		for ii := 0; ii < i; ii++ {
			if _, err := repo.SaveChunk(file, server, uint(ii)); err != nil {
				t.Fatalf("can't save chunk: %s", err)
			}
		}
	}

	tests := []struct {
		num  int
		want []*Server
	}{
		{num: 0, want: []*Server{}},
		{num: 1,
			want: []*Server{
				{ID: saved[0], Name: "GetLeastLoadedServer0", Port: "123"}}},
		{num: 2,
			want: []*Server{
				{ID: saved[0], Name: "GetLeastLoadedServer0", Port: "123"},
				{ID: saved[1], Name: "GetLeastLoadedServer1", Port: "123"},
			}},
		{num: 3,
			want: []*Server{
				{ID: saved[0], Name: "GetLeastLoadedServer0", Port: "123"},
				{ID: saved[1], Name: "GetLeastLoadedServer1", Port: "123"},
				{ID: saved[2], Name: "GetLeastLoadedServer2", Port: "123"},
			}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			servers, err := repo.GetLeastLoadedServers(tt.num)
			if err != nil {
				t.Errorf("GetLeastLoadedServer() error = %v", err)
			}
			if diff := cmp.Diff(tt.want, servers); diff != "" {
				t.Errorf("GetLeastLoadedServer():\n%s", diff)
			}
		})
	}
}

func TestRepository_SaveChunk(t *testing.T) {
	repo := setup()
	server := &Server{Name: "TestServer", Port: "8080"}
	if err := repo.db.Create(server).Error; err != nil {
		t.Fatalf("can't prepare test")
	}
	file1, file2 := &File{User: "user1", Dir: "dir1", Name: "file1"}, &File{User: "user1", Dir: "dir1", Name: "file2"}
	if err := repo.db.Create([]*File{file1, file2}).Error; err != nil {
		t.Fatalf("can't prepare test: %s", err)
	}

	tests := []struct {
		name    string
		number  uint
		file    uuid.UUID
		server  uuid.UUID
		wantErr bool
	}{
		{name: "file1", server: server.ID, file: file1.ID, number: 1},
		{name: "file1 duplicated", server: server.ID, file: file1.ID, number: 1, wantErr: true}, // duplicated chunk
		{name: "file2", server: server.ID, file: file2.ID, number: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := repo.SaveChunk(tt.file, tt.server, tt.number)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveChunk() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			t.Run("check saved", func(t *testing.T) {
				saved := &Chunk{}
				if err := repo.db.First(saved, id).Error; err != nil {
					t.Errorf("SaveChunk() can't find saved chunk")
				}
				expected := &Chunk{
					ID:       id,
					Number:   tt.number,
					FileID:   tt.file,
					ServerID: server.ID,
				}
				if diff := cmp.Diff(expected, saved); diff != "" {
					t.Errorf("SaveChunk()\n%s", diff)
				}
			})
		})
	}
}

func TestRepository_GetFiles(t *testing.T) {
	repo := setup()

	servers := make([]uuid.UUID, 0, 6)
	for i := 0; i < cap(servers); i++ {
		id, err := repo.AddServer(fmt.Sprintf("TestGetChunks%d", i), "123")
		if err != nil {
			t.Fatalf("can't save server: %s", err)
		}
		servers = append(servers, id)
	}
	files := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		fileId, err := repo.SaveFile("username3", "dir", fmt.Sprintf("GetFiles_%d", i))
		if err != nil {
			t.Fatalf("can't save file: %s", err)
		}
		files[i] = fileId
		for chunkNum, serverId := range servers {
			if _, err := repo.SaveChunk(fileId, serverId, uint(chunkNum)); err != nil {
				t.Fatalf("can't save chunk: %s", err)
			}
		}
	}

	tests := []struct {
		username string
		name     string
		dir      string
		filename string
		want     *File
		wantErr  bool
	}{
		{name: "regular", username: "username3", dir: "dir", filename: "GetFiles_0",
			want: &File{
				ID:   files[0],
				User: "username3",
				Dir:  "dir",
				Name: "GetFiles_0",
				Chunks: []*Chunk{
					{Number: 0, ServerID: servers[0], FileID: files[0]},
					{Number: 1, ServerID: servers[1], FileID: files[0]},
					{Number: 2, ServerID: servers[2], FileID: files[0]},
					{Number: 3, ServerID: servers[3], FileID: files[0]},
					{Number: 4, ServerID: servers[4], FileID: files[0]},
					{Number: 5, ServerID: servers[5], FileID: files[0]},
				}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetFile(tt.username, tt.dir, tt.filename)
			if err != nil {
				t.Errorf("GetFile() error: %v", err)
			}
			assert.Equal(t,
				tt.want.ID,
				got.ID)
			assert.Equal(t, tt.want.User, got.User)
			assert.Equal(t, tt.want.Dir, got.Dir)
			assert.Equal(t, tt.want.Name, got.Name)
			for i, chunk := range tt.want.Chunks {
				assert.Equal(t, chunk.Number, got.Chunks[i].Number)
				assert.Equal(t, chunk.ServerID, got.Chunks[i].ServerID)
				assert.Equal(t, chunk.ServerID, got.Chunks[i].Server.ID)
				assert.Equal(t, chunk.FileID, got.Chunks[i].FileID)
				assert.Equal(t, chunk.FileID, got.Chunks[i].File.ID)
			}
		})
	}
}

func TestRepository_RemoveFile(t *testing.T) {
	repo := setup()
	serverId, err := repo.AddServer("RemoveFile", "12")
	if err != nil {
		t.Fatalf("can't prepare test: %s", err)
	}
	fileId, err := repo.SaveFile("RemoveFile_user", "RemoveFile_dir", "RemoveFile_file")
	if err != nil {
		t.Fatalf("can't prepare test: %s", err)
	}

	for i := 0; i < 6; i++ {
		if _, err = repo.SaveChunk(fileId, serverId, uint(i)); err != nil {
			t.Fatalf("can't prepare test: %s", err)
		}
	}
	f, err := repo.GetFile("RemoveFile_user", "RemoveFile_dir", "RemoveFile_file")
	if err != nil {
		t.Fatalf("can't find saved file:  %s", err)
	}
	assert.Equal(t, f.ID, fileId)
	assert.Equal(t, f.User, "RemoveFile_user")
	assert.Equal(t, f.Name, "RemoveFile_file")
	assert.Equal(t, f.Dir, "RemoveFile_dir")
	assert.Equal(t, len(f.Chunks), 6)
	for _, chunk := range f.Chunks {
		assert.Equal(t, chunk.ServerID, serverId)
		assert.Equal(t, chunk.Server.ID, serverId)
		assert.Equal(t, chunk.FileID, fileId)
		assert.Equal(t, chunk.File.ID, fileId)
	}

	if err := repo.RemoveFile("RemoveFile_user", "RemoveFile_dir", "RemoveFile_file"); err != nil {
		t.Fatalf("can't remove file")
	}
	f, err = repo.GetFile("RemoveFile_user", "RemoveFile_dir", "RemoveFile_file")
	if assert.Error(t, err) {
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	}
}
