package database

import (
	"database/sql"

	"github.com/google/uuid"
	sqliteGo "github.com/mattn/go-sqlite3"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const CustomDriverName = "sqlite3_extended"

const DefaultFile = "rest-service.db"

func init() {
	sql.Register(CustomDriverName,
		&sqliteGo.SQLiteDriver{
			ConnectHook: func(conn *sqliteGo.SQLiteConn) error {
				err := conn.RegisterFunc(
					"gen_random_uuid",
					func(arguments ...interface{}) (string, error) {
						u, err := uuid.NewRandom()
						if err != nil {
							return "", err
						}
						return u.String(), nil
					},
					true,
				)
				return err
			},
		},
	)
}

func NewDb(file string) (*gorm.DB, error) {

	conn, err := sql.Open(CustomDriverName, file)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: CustomDriverName,
		DSN:        file,
		Conn:       conn,
	}, &gorm.Config{
		Logger:                   logger.Default.LogMode(logger.Info),
		SkipDefaultTransaction:   true,
		DisableNestedTransaction: true,
	})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&Server{}, &Chunk{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Server{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Chunk{})
	return db, err
}
