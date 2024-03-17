package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/konorlevich/test_task_s3/internal/rest-service/database"
	"github.com/konorlevich/test_task_s3/internal/rest-service/handler"
)

var port = "8080"
var dbFile = database.DefaultFile

func init() {
	p := os.Getenv("REST_PORT")
	if p != "" {
		port = p
	}

	d := os.Getenv("DB_FILE")
	if d != "" {
		dbFile = d
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer log.Println("got interruption signal")

	db, err := database.NewDb(dbFile)
	if err != nil {
		log.Fatal("failed to open database")
	}
	server := &http.Server{Addr: ":" + port, Handler: handler.NewHandler(database.NewRepository(db))}

	go func() {
		log.Printf("listening to port %s\n", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen and serve returned err: %v", err)
		}
	}()
	defer func() {
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("handler shutdown returned an err: %v\n", err)
		}
	}()

	<-ctx.Done()
}
