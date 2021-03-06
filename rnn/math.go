package rnn

import (
	"math"

	"github.com/gonum/matrix/mat64"
)

func tanh(a []float64) []float64 {
	ret := make([]float64, len(a))
	for i, v := range a {
		ret[i] = math.Tanh(v)
	}
	return ret
}

func dotVec(a, b []float64) *mat64.Dense {
	va := mat64.NewDense(len(a), 1, a)
	vb := mat64.NewDense(1, len(b), b)
	ret := new(mat64.Dense)
	ret.Mul(va, vb)
	return ret
}
func dot(a mat64.Matrix, b []float64) []float64 {
	t := mat64.NewDense(len(b), 1, b)
	row, _ := a.Dims()
	backend := make([]float64, row)
	r := mat64.NewDense(row, 1, backend)
	r.Mul(a, t)
	return backend
}

func exp(a []float64) []float64 {
	ret := make([]float64, len(a))
	for i, v := range a {
		ret[i] = math.Exp(v)
	}
	return ret
}

func sum(a []float64) float64 {
	var res float64
	for _, v := range a {
		res += v
	}
	return res
}
func div(a []float64, val float64) []float64 {
	ret := make([]float64, len(a))
	for i, v := range a {
		ret[i] = v / val
	}
	return ret

}
func add(a ...[]float64) []float64 {
	ret := make([]float64, len(a[0]))
	for _, element := range a {
		for i, v := range element {
			ret[i] += v
		}
	}
	return ret
}

// Calculate the normalized probability of the second dimension
// of the array
func normalizeByRow(ys [][]float64) (ps [][]float64) {
	inputSize := len(ys)
	// probabilities for next chars
	ps = make([][]float64, inputSize)
	for t := 0; t < inputSize; t++ {
		ps[t] = make([]float64, len(ys[t]))
		expYS := exp(ys[t])
		ps[t] = div(expYS, sum(expYS))
	}
	return
}
