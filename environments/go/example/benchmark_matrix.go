package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"gonum.org/v1/gonum/mat"
)

// Handler is the entry point for this fission function
func Handler(w http.ResponseWriter, r *http.Request) {
	strN := r.URL.Query().Get("n")
	if strN == "" {
		strN = "1"
	}

	n, parseErr := strconv.Atoi(strN)
	if parseErr != nil {
		n = 1
	}

	// Generate a 6×6 matrix of random values.
	data1 := make([]float64, 100)
	for i := range data1 {
		data1[i] = rand.NormFloat64()
	}
	a := mat.NewDense(10, 10, data1)

	for i := 0; i < n; i++ {
		data2 := make([]float64, 100)
		for i := range data2 {
			data2[i] = rand.NormFloat64()
		}
		b := mat.NewDense(10, 10, data2)

		var c mat.Dense // construct a new zero-sized matrix
		c.Mul(a, b)     // c is automatically adjusted to be 6×6

		a = &c
	}

	msg := fmt.Sprintf("Det %f", mat.Det(a))

	w.Write([]byte(msg))
}
