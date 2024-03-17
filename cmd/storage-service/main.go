package main

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"

	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/konorlevich/test_task_s3/internal/storage-service/handler"
	"github.com/konorlevich/test_task_s3/internal/storage-service/register"
	"github.com/konorlevich/test_task_s3/internal/storage-service/storage"
)

var (
	port               = "8080"
	restServiceBaseUrl = ""
	storagePath        = "/var/storage"
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
	s := os.Getenv("STORAGE_PATH")
	if s != "" {
		storagePath = s
	}
}

func main() {
	l := log.New().WithFields(log.Fields{
		"port":                  port,
		"rest_service_base_url": restServiceBaseUrl,
		"storage_path":          storagePath,
	})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer l.Info("got interruption signal")
	s, err := storage.NewStorage(storagePath, l)
	if err != nil {
		l.Fatal(err)
	}
	server := &http.Server{Addr: ":" + port, Handler: handler.NewHandler(s)}

	go func() {
		l.Info("listen and serve")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.WithError(err).Error("listen and serve failed", err)
			stop()
		}
	}()
	defer func() {
		if err := server.Shutdown(context.Background()); err != nil {
			l.WithError(err).Fatal("handler shutdown failed")
		}
	}()

	hostname, err := os.Hostname()
	if err != nil {
		l.WithError(err).Fatal("can't get server name")
	}

	l = l.WithField("server_url", restServiceBaseUrl)
	restServiceUrl, err := url.Parse(restServiceBaseUrl)
	if err != nil {
		l.WithError(err).Fatal("can't parse register server url", err)
	}
	l.Info("registering the service ", restServiceUrl.String())
	if err = register.Register(restServiceUrl, hostname, port); err != nil {
		l.Fatalf("can't register on server %s: %s\n", hostname, err)
	}
	<-ctx.Done()
}
