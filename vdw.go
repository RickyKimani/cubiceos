package cubiceos

type VdW struct{}

func (VdW) Alpha(tr, w float64) float64 {
	return 1.0
}

func (VdW) Params() Params {
	return Params{
		Sigma:   0,
		Epsilon: 0,
		Omega:   1.0 / 8.0,
		Psi:     27.0 / 64.0,
	}
}

// NewvdW creates a configuration for the van der Waals cubic equation of state
func NewvdWCfg(T, P, Tc, Pc, R float64) EOSCfg {
	return EOSCfg{
		Type: VdW{},
		T:    T,
		P:    P,
		Tc:   Tc,
		Pc:   Pc,
		W:    0,
		R:    R,
	}
}
