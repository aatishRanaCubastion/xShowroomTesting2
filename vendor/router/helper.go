package router

import (
	"net/http"
)

func Delete(path string, fn http.HandlerFunc) {
	r.Router.DELETE(path, HandlerFunc(fn))
}

func Get(path string, fn http.HandlerFunc) {
	r.Router.GET(path, HandlerFunc(fn))
}

func Head(path string, fn http.HandlerFunc) {
	r.Router.HEAD(path, HandlerFunc(fn))
}

func Options(path string, fn http.HandlerFunc) {
	r.Router.OPTIONS(path, HandlerFunc(fn))
}

func Patch(path string, fn http.HandlerFunc) {
	r.Router.PATCH(path, HandlerFunc(fn))
}

func Post(path string, fn http.HandlerFunc) {
	r.Router.POST(path, HandlerFunc(fn))
}

func Put(path string, fn http.HandlerFunc) {
	r.Router.PUT(path, HandlerFunc(fn))
}

func PostHandler(path string, handler http.Handler) {
	r.Router.Handler("POST", path, handler)
}
