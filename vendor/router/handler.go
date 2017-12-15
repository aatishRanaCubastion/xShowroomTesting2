package router

import (
	"net/http"
	"github.com/julienschmidt/httprouter"
	"github.com/gorilla/context"
)

func HandlerFunc(h http.HandlerFunc) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		context.Set(r, params, p)
		h.ServeHTTP(w, r)
	}
}

func Handler(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		context.Set(r, params, p)
		h.ServeHTTP(w, r)
	}
}
