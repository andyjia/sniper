package server

import (
	"net/http"

	"sniper/cmd/server/hook"

	"github.com/bilibili/twirp"

	foo_v1 "sniper/rpc/foo/v1"
	"sniper/server/fooserver1"
)

var hooks = twirp.ChainHooks(
	hook.NewRequestID(),
	hook.NewLog(),
)

var loginHooks = twirp.ChainHooks(
	hook.NewRequestID(),
	hook.NewCheckLogin(),
	hook.NewLog(),
)

func initMux(mux *http.ServeMux, isInternal bool) {
	{
		server := &fooserver1.Server{}
		// handler := foo_v1.NewFooServer(server, hooks)
		handler := foo_v1.NewFooServer(server, loginHooks)
		mux.Handle(foo_v1.FooPathPrefix, handler)
	}
}

func initInternalMux(mux *http.ServeMux) {
}
