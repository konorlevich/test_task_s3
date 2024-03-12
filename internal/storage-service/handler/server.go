package handler

import "net/http"

func NewHandler() *http.ServeMux {
	handler := http.NewServeMux()

	//handler.HandleFunc("GET /object/{dir}/{name}", func(rw http.ResponseWriter, r *http.Request) {})
	//handler.HandleFunc("POST /object/{dir}/{name}", func(rw http.ResponseWriter, r *http.Request) {})

	return handler
}
