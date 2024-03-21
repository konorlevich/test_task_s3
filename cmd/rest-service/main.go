package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/konorlevich/test_task_s3/internal/rest-service/database"
	"github.com/konorlevich/test_task_s3/internal/rest-service/handler"
)

var port = "8080"
var dbFile = database.DefaultFile
var chunkNum = database.DefaultChunkNum

func init() {
	p := os.Getenv("REST_PORT")
	if p != "" {
		port = p
	}

	d := os.Getenv("DB_FILE")
	if d != "" {
		dbFile = d
	}

	c, err := strconv.Atoi(os.Getenv("CHUNK_NUM"))
	if err != nil && c != 0 {
		chunkNum = c
	}
}

func main() {
	l := log.New().WithFields(log.Fields{
		"rest_port": port,
		"db_file":   dbFile,
		"chunk_num": chunkNum,
	})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer l.Println("got interruption signal")

	db, err := database.NewDb(dbFile)
	if err != nil {
		l.WithError(err).Fatal("failed to open database")
	}
	server := &http.Server{Addr: ":" + port, Handler: handler.NewHandler(database.NewRepository(db), chunkNum, l)}

	go func() {
		l.Printf("listening to port %s\n", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.WithError(err).Fatal("listen and serve returned err")
		}
	}()
	defer func() {
		if err := server.Shutdown(context.Background()); err != nil {
			l.WithError(err).Error("handler shutdown returned an err")
		}
	}()

	<-ctx.Done()
}
