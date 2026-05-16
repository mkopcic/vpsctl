package web

import (
	"embed"
	"io/fs"
	"net/http"
)

// embed direktiva — kompajler ucita sve fajlove iz static/ u binary
// Nema runtime dependencyja, jedan binary ima sve unutra
//
//go:embed static
var files embed.FS

// FS vraca http.FileSystem koji Gin koristi za StaticFS.
// fs.Sub stripa "static/" prefix pa /ui/index.html mapa na static/index.html
func FS() http.FileSystem {
	sub, _ := fs.Sub(files, "static")
	return http.FS(sub)
}
