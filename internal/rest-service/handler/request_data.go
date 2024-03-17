package handler

import "net/http"

const (
	fieldNameDir      = "dir"
	fieldNameUsername = "username"
	fieldNameFileName = "name"
)

type requestData struct {
	username string
	dir      string
	name     string
}

func newRequestData(r *http.Request) *requestData {
	username, _, _ := r.BasicAuth()

	return &requestData{
		username: username,
		dir:      r.PathValue(fieldNameDir),
		name:     r.PathValue(fieldNameFileName),
	}
}
