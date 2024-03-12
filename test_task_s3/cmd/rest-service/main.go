package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/konorlevich/test_task_s3/internal/rest-service/handler"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var port = "8080"

func init() {
	p := os.Getenv("REST_PORT")
	if p != "" {
		port = p
	}
}

func main() {
	storageServers := &sync.Map{}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := &http.Server{Addr: ":" + port, Handler: handler.NewHandler(storageServers)}

	go func() {
		fmt.Printf("listening to port %s\n", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen and serve returned err: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("got interruption signal")
	if err := server.Shutdown(context.TODO()); err != nil {
		log.Printf("handler shutdown returned an err: %v\n", err)
	}
}
