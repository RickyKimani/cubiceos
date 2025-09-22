package cubiceos

import (
	"errors"
)

// Params represents the substance agnostic variables in any
// cubic equation of state
type Params struct {
	Sigma   float64
	Epsilon float64
	Omega   float64
	Psi     float64
}

// EOSType defines what makes up an equation of state
type EOSType interface {
	Alpha(tr, w float64) float64
	Params() Params
}

// EOSCfg is a configuration struct for an equation of state
type EOSCfg struct {
	Type EOSType
	T    float64 //Absolute temp
	P    float64 //Pressure
	Tc   float64 //Critical temp (Absolute)
	Pc   float64 //Critical pressure
	W    float64 //Acentric factor
	R    float64 //Universal gas constant
}

// CubicEOS solves the cubic equation and returns the volumes
func CubicEOS(cfg EOSCfg) ([3]complex128, error) {
	if cfg.T <= 0 {
		return [3]complex128{}, errors.New("absolute temp cannot be less than 0")
	}

	if cfg.P <= 0 {
		return [3]complex128{}, errors.New("pressure cannot be less than or equal to 0")
	}

	if cfg.Tc <= 0 {
		return [3]complex128{}, errors.New("critical temp cannot be less than or equal to 0")
	}

	if cfg.Pc <= 0 {
		return [3]complex128{}, errors.New("critical pressure cannot be less than or equal to 0")
	}

	if cfg.R <= 0 {
		return [3]complex128{}, errors.New("universal gas constant cannot be less than or equal to 0")
	}

	//Reduced components
	tr := cfg.T / cfg.Tc
	//pr := cfg.P/cfg.Pc

	//a(T)
	a := cfg.Type.Params().Psi * cfg.Type.Alpha(tr, cfg.W) * cfg.R * cfg.R * cfg.Tc * cfg.Tc / cfg.Pc
	//debug
	// fmt.Printf("alpha: %+f\n", cfg.Type.Alpha(tr, cfg.W))//correct
	// fmt.Printf("Psi: %+f\n", cfg.Type.Params().Psi) //correct

	b := cfg.Type.Params().Omega * cfg.R * cfg.Tc / cfg.Pc
	//fmt.Printf("Omega: %+f\n", cfg.Type.Params().Omega) //correct

	//eV^3 + fV^2 + gV + h = 0

	x := cfg.Type.Params().Epsilon + cfg.Type.Params().Sigma
	y := cfg.Type.Params().Epsilon * cfg.Type.Params().Sigma
	v_ig := cfg.R * cfg.T / cfg.P

	e := 1.0
	f := b*(x-1) - v_ig
	g := b*((y-x)*b-(x*v_ig)) + a/cfg.P
	h := -y*b*b*(b+v_ig) - a*b/cfg.P

	return SolveCubic(e, f, g, h)

}
