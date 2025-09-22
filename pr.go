package cubiceos

import "math"

type PR struct{}

func (PR) Alpha(tr, w float64) float64 {
	a := 0.37464 + 1.54226*w - 0.26992*w*w
	b := 1 - math.Sqrt(tr)
	c := 1 + a*b
	return c * c
}

func (PR) Params() Params {
	return Params{
		Sigma:   1 + math.Sqrt2,
		Epsilon: 1 - math.Sqrt2,
		Omega:   0.07780,
		Psi:     0.45724,
	}
}

// NewPRCfg creates a configuration for the Peng/Robinson cubic equation of state
func NewPRCfg(T, P, Tc, Pc, W, R float64) EOSCfg {
	return EOSCfg{
		Type: PR{},
		T:    T,
		P:    P,
		Tc:   Tc,
		Pc:   Pc,
		W:    W,
		R:    R,
	}
}
