package web

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"

	"github.com/rickykimani/cubiceos"
	"github.com/rickykimani/cubiceos/internal/web/pages"
)

func New() *http.ServeMux {
	mux := http.NewServeMux()
	// Serve static assets from internal/web/assets at /assets/
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("internal/web/assets"))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := pages.HomePage().Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/calculate", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		parseFloat := func(name string, required bool) (float64, error) {
			v := r.FormValue(name)
			if v == "" {
				if required {
					return 0, fmt.Errorf("%s required", name)
				}
				return 0, nil
			}
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid %s", name)
			}
			return f, nil
		}

		T, err := parseFloat("T", true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		P, err := parseFloat("P", true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		Tc, err := parseFloat("Tc", true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		Pc, err := parseFloat("Pc", true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		R, err := parseFloat("R", true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		omega, _ := parseFloat("omega", false)
		withAdv := r.FormValue("with_advanced") != ""

		// Build configurations
		vdWCfg := cubiceos.NewvdWCfg(T, P, Tc, Pc, R)
		rkCfg := cubiceos.NewRKCfg(T, P, Tc, Pc, R)
		srkCfg := cubiceos.NewSRKCfg(T, P, Tc, Pc, omega, R)
		prCfg := cubiceos.NewPRCfg(T, P, Tc, Pc, omega, R)

		collect := func(name string, cfg cubiceos.EOSCfg, include bool) *pages.EOSResult {
			if !include {
				return nil
			}
			roots, err := cubiceos.CubicEOS(cfg)
			if err != nil {
				return &pages.EOSResult{Name: name, Classification: err.Error()}
			}
			const eps = 1e-9
			positives := make([]float64, 0, 3)
			for _, rt := range roots {
				if math.Abs(imag(rt)) < eps {
					rv := real(rt)
					if rv > 0 {
						positives = append(positives, rv)
					}
				}
			}
			sort.Float64s(positives)
			res := &pages.EOSResult{Name: name, Roots: positives}
			switch len(positives) {
			case 0:
				res.Classification = "none"
			case 1:
				res.Classification = "single-phase"
				res.Vapor = &positives[0]
			case 2:
				// treat smaller as liquid, larger as vapor (approx)
				res.Classification = "two-phase"
				res.Liquid = &positives[0]
				res.Vapor = &positives[1]
			case 3:
				// Check for critical (all nearly equal)
				if math.Abs(positives[0]-positives[1]) < 1e-6 && math.Abs(positives[1]-positives[2]) < 1e-6 {
					res.Classification = "critical"
					res.Vapor = &positives[0]
				} else {
					res.Classification = "two-phase"
					res.Liquid = &positives[0]
					res.Unstable = &positives[1]
					res.Vapor = &positives[2]
				}
			}
			return res
		}

		results := make([]pages.EOSResult, 0, 4)
		if r := collect("van der Waals", vdWCfg, true); r != nil {
			results = append(results, *r)
		}
		if r := collect("Redlich-Kwong", rkCfg, true); r != nil {
			results = append(results, *r)
		}
		if withAdv {
			if r := collect("Soave-Redlich-Kwong", srkCfg, true); r != nil {
				results = append(results, *r)
			}
			if r := collect("Peng-Robinson", prCfg, true); r != nil {
				results = append(results, *r)
			}
		}

		if err := pages.ResultsPage(results).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	return mux
}

func Run() {
	log.Println("Server running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", New()); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
