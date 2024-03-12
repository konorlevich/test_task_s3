package main

import (
	"context"
	"errors"
	"github.com/konorlevich/test_task_s3/internal/storage-service/handler"
	"github.com/konorlevich/test_task_s3/internal/storage-service/register"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
)

var (
	port               = "8080"
	restServiceBaseUrl = ""
)

func init() {
	p := os.Getenv("STORAGE_PORT")
	if p != "" {
		port = p
	}
	r := os.Getenv("REST_SERVICE_URL")
	if r != "" {
		restServiceBaseUrl = r
	}
}
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := &http.Server{Addr: ":" + port, Handler: handler.NewHandler()}

	go func() {
		log.Printf("listening to port %s\n", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen and serve returned err: %v", err)
		}
	}()
	defer func() {
		if err := server.Shutdown(context.TODO()); err != nil {
			log.Printf("handler shutdown returned an err: %v\n", err)
		}
	}()

	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("can't get server name: %v\n", err)
		os.Exit(3)
	}

	restServiceUrl, err := url.Parse(restServiceBaseUrl)
	if err != nil {
		log.Printf("can't parse register server url %s: %s\n", restServiceBaseUrl, err)
		os.Exit(3)
	}
	log.Println("registering the service to ", restServiceUrl.String())
	if err = register.Register(restServiceUrl, hostname, port); err != nil {
		log.Printf("can't register on server %s: %s\n", hostname, err)
		os.Exit(3)
	}
	<-ctx.Done()
	log.Println("got interruption signal")
}
