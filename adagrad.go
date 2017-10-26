package main

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// adagrad is a structure that holds the memory of the adaptative gradient
type adagrad struct {
	mwhy *mat.Dense //
	mwhh *mat.Dense // memory for the adaptative gradient updagte
	mwxh *mat.Dense //
	mbh  []float64  // This is the biais
	mby  []float64  // This is the biais
}

// Create a new adaptative gradient structure suitable to the rnn shape
func newAdagrad(rnn *rnn) *adagrad {
	a := &adagrad{}
	a.mwxh = mat.NewDense(rnn.config.hiddenNeurons, rnn.config.inputNeurons, nil)
	a.mwhh = mat.NewDense(rnn.config.hiddenNeurons, rnn.config.hiddenNeurons, nil)
	a.mwhy = mat.NewDense(rnn.config.outputNeurons, rnn.config.hiddenNeurons, nil)
	a.mbh = make([]float64, rnn.config.hiddenNeurons)
	a.mby = make([]float64, rnn.config.outputNeurons)
	return a
}

// apply the Adaptative gradient to the rnn
func (a *adagrad) apply(r *rnn, dwxh, dwhh, dwhy *mat.Dense, dbh, dby []float64) {
	memFunc := func(_, _ int, v float64) float64 {
		return math.Sqrt(v + 1e-8)
	}
	learningRateFunc := func(_, _ int, v float64) float64 {
		return -r.config.learningRate * v
	}

	for _, params := range [][3]*mat.Dense{
		[3]*mat.Dense{
			r.wxh, dwxh, a.mwxh,
		},
		[3]*mat.Dense{
			r.whh, dwhh, a.mwhh,
		},
		[3]*mat.Dense{
			r.why, dwhy, a.mwhy,
		},
		[3]*mat.Dense{
			mat.NewDense(len(r.bh), 1, r.bh), mat.NewDense(len(dbh), 1, dbh), mat.NewDense(len(a.mbh), 1, a.mbh),
		},
		[3]*mat.Dense{
			mat.NewDense(len(r.by), 1, r.by), mat.NewDense(len(dby), 1, dby), mat.NewDense(len(a.mby), 1, a.mby),
		},
	} {
		param := params[0]
		dparam := params[1]
		mem := params[2]
		tmp := new(mat.Dense)
		tmp.MulElem(dparam, dparam)
		mem.Add(mem, tmp)
		tmp.Reset()
		tmp.Apply(memFunc, mem)
		tmp2 := new(mat.Dense)
		tmp2.Apply(learningRateFunc, dparam)
		tmp3 := new(mat.Dense)
		tmp3.DivElem(tmp2, tmp)
		param.Add(param, tmp3)
	}
}
