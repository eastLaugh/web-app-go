package server

import (
	"io/fs"
	"net/http"
)

func Serve(fsys fs.FS) {
	handler := http.FileServer(http.FS(fsys))
	http.Handle("/app/", http.StripPrefix("/app/", handler))
	http.Handle("/", http.RedirectHandler("/app/", http.StatusTemporaryRedirect))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
