package main

import (
	"fmt"
	"math/rand"

	"gonum.org/v1/gonum/mat"
	"github.com/valyala/fasthttp"
)

// Handler is the entry point for this fission function
func Handler(ctx *fasthttp.RequestCtx) {
	n, err := ctx.URI().QueryArgs().GetUint("n")
	if err != nil {
		n = 1
	}

	// Generate a 6×6 matrix of random values.
	data1 := make([]float64, 100)
	data2 := make([]float64, 100)

	var out *mat.Dense
	for i := 0; i < n; i++ {
		for i := range data1 {
			data1[i] = rand.Float64()
		}
		a := mat.NewDense(10, 10, data1)

		for i := range data2 {
			data2[i] = rand.Float64()
		}
		b := mat.NewDense(10, 10, data2)

		var c mat.Dense // construct a new zero-sized matrix
		c.Mul(a, b)     // c is automatically adjusted to be 6×6

		out = &c
	}

	msg := fmt.Sprintf("Det %f", mat.Det(out))

	ctx.Write([]byte(msg))
}
