package web

import "net/http"

// Mount registers the shared static asset route. It never imports any
// module - every module's own web subpackage depends on this package for
// Layout, so this package must not depend back on any of them.
func Mount(mux *http.ServeMux) {
	mux.Handle("GET /static/", http.FileServerFS(staticFS))
}
