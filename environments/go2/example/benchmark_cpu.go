package main

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

// Handler is the entry point for this fission function
func Handler(ctx *fasthttp.RequestCtx) {
	n, err := ctx.URI().QueryArgs().GetUint("n")
	if err != nil {
		n = 1
	}

	seq := 0
	for i := 0; i < n; i++ {
		seq++
	}

	msg := fmt.Sprintf("Ret %d", seq)

	ctx.Write([]byte(msg))
}
