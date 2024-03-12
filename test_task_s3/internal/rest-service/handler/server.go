package handler

import (
	"log"
	"net/http"
	"sync"
)

func NewHandler(storageServers *sync.Map) *http.ServeMux {

	handler := http.NewServeMux()

	//handler.HandleFunc("GET /object/{dir}/{name}", func(rw http.ResponseWriter, r *http.Request) {})
	//handler.HandleFunc("POST /object/{dir}/{name}", func(rw http.ResponseWriter, r *http.Request) {})

	handler.HandleFunc("POST /storage/register", RegisterStorage(storageServers))
	return handler
}

func RegisterStorage(storageServers *sync.Map) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil || !r.PostForm.Has("hostname") || !r.PostForm.Has("port") {
			rw.WriteHeader(http.StatusNotAcceptable)
			rw.Write([]byte("Can't read request data"))
			return
		}
		hostname := r.PostForm.Get("hostname")
		port := r.PostForm.Get("port")
		storageServers.Store(hostname, port)
		log.Printf("storage service added - %s:%s", hostname, port)
		rw.WriteHeader(200)
		rw.Write([]byte("added " + hostname))
	}
}
