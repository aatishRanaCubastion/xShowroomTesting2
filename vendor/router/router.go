package router

import (
	"net/http"

	"github.com/gorilla/context"
	"github.com/justinas/alice"
	"github.com/julienschmidt/httprouter"
)

var (
	r RouterInfo
)

const (
	params = "params"
)

type RouterInfo struct {
	Router *httprouter.Router
}

func init() {
	r.Router = httprouter.New()
}

func ReadConfig() RouterInfo {
	return r
}

func Instance() *httprouter.Router {
	return r.Router
}

func Params(r *http.Request) httprouter.Params {
	return context.Get(r, params).(httprouter.Params)
}

func Chain(fn http.HandlerFunc, c ...alice.Constructor) httprouter.Handle {
	return Handler(alice.New(c...).ThenFunc(fn))
}
