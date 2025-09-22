package cubiceos

import "math"

type RK struct{}

func (RK) Alpha(tr, w float64) float64 {
	return 1 / math.Sqrt(tr)
}

func (RK) Params() Params {
	return Params{
		Sigma:   1,
		Epsilon: 0,
		Omega:   0.08664,
		Psi:     0.42728,
	}
}

// NewRKCfg creates a configuration for the Redlich/Kwong cubic equation of state
func NewRKCfg(T, P, Tc, Pc, R float64) EOSCfg {
	return EOSCfg{
		Type: RK{},
		T:    T,
		P:    P,
		Tc:   Tc,
		Pc:   Pc,
		W:    0,
		R:    R,
	}
}
