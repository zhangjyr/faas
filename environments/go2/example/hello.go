package main

import (
	"github.com/valyala/fasthttp"
)

// Handler is the entry point for this fission function
func Handler(ctx *fasthttp.RequestCtx) {
	msg := "Hello, world!\n"
	ctx.Write([]byte(msg))
}
