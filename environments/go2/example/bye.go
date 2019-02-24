package main

import (
	"github.com/valyala/fasthttp"
)

// Handler is the entry point for this fission function
func Handler(ctx *fasthttp.RequestCtx) {
	msg := "Bye, world!\n"
	ctx.Write([]byte(msg))
}
