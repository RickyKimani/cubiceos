package cubiceos

import "math"

type SRK struct{}

func (SRK) Alpha(tr, w float64) float64 {
	a := 0.480 + 1.574*w - 0.716*w*w
	b := 1 - math.Sqrt(tr)
	c := 1 + a*b
	return c * c
}

func (SRK) Params() Params {
	return Params{
		Sigma:   1,
		Epsilon: 0,
		Omega:   0.08664,
		Psi:     0.42728,
	}
}

// NewSRKCfg creates a configuration for the Soave/Redlich/Kwong cubic equation of state
func NewSRKCfg(T, P, Tc, Pc, W, R float64) EOSCfg {
	return EOSCfg{
		Type: SRK{},
		T:    T,
		P:    P,
		Tc:   Tc,
		Pc:   Pc,
		W:    W,
		R:    R,
	}
}
