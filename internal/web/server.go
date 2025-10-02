package web

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rickykimani/cubiceos"
	"github.com/rickykimani/cubiceos/internal/web/pages"
)

var lastActivity int64

func newSrvMux() *http.ServeMux {
	atomic.StoreInt64(&lastActivity, time.Now().Unix())

	mux := http.NewServeMux()
	// Serve static assets from the embedded filesystem.
	var f fs.FS = embeddedAssets
	if sub, err := fs.Sub(embeddedAssets, "assets"); err == nil {
		f = sub
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(f))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.StoreInt64(&lastActivity, time.Now().Unix())
		if err := pages.HomePage().Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/calculate", func(w http.ResponseWriter, r *http.Request) {
		atomic.StoreInt64(&lastActivity, time.Now().Unix())
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		// Unit helpers
		toKelvin := func(val float64, unit string) (float64, error) {
			switch strings.ToUpper(unit) {
			case "K", "":
				if val <= 0 {
					return 0, fmt.Errorf("temperature in K must be > 0")
				}
				return val, nil
			case "C":
				if val < -273.15 {
					return 0, fmt.Errorf("temperature in °C must be ≥ -273.15")
				}
				return val + 273.15, nil
			default:
				return 0, fmt.Errorf("unsupported temperature unit")
			}
		}
		toPa := func(val float64, unit string) (float64, error) {
			switch strings.ToLower(unit) {
			case "pa", "":
				return val, nil
			case "kpa":
				return val * 1e3, nil
			case "bar":
				return val * 1e5, nil
			case "atm":
				return val * 101325.0, nil
			default:
				return 0, fmt.Errorf("unsupported pressure unit")
			}
		}

		vFactor := func(unit string) (float64, string) {
			switch unit {
			case "m3_per_mol":
				return 1.0, "m³/mol"
			case "m3_per_kmol":
				return 1e3, "m³/kmol"
			case "cm3_per_mol":
				return 1e-6, "cm³/mol"
			case "cm3_per_kmol":
				return 1e-9, "cm³/kmol"
			default:
				return 1.0, "m³/mol"
			}
		}
		pFactor := func(unit string) (float64, string) {
			switch strings.ToLower(unit) {
			case "pa", "":
				return 1.0, "Pa"
			case "kpa":
				return 1e3, "kPa"
			case "bar":
				return 1e5, "bar"
			case "atm":
				return 101325.0, "atm"
			default:
				return 1.0, "Pa"
			}
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
		const internalR = 8.314 // Pa·m^3/(mol·K)
		_, _ = parseFloat("R", false)
		omega, _ := parseFloat("omega", false)
		withAdv := r.FormValue("with_advanced") != ""

		// Units from form
		tUnit := r.FormValue("T_unit")
		pUnit := r.FormValue("P_unit")
		tcUnit := r.FormValue("Tc_unit")
		pcUnit := r.FormValue("Pc_unit")
		vUnit := r.FormValue("v_unit")
		if vUnit == "" {
			vUnit = "cm3_per_mol"
		}

		// Convert to SI
		Tsi, err := toKelvin(T, tUnit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		Tcsi, err := toKelvin(Tc, tcUnit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		Psi, err := toPa(P, pUnit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		Pcsi, err := toPa(Pc, pcUnit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Display scaling factors and labels
		vFac, vLabel := vFactor(vUnit)
		pFac, pLabel := pFactor(pUnit)

		// Build configurations
		vdWCfg := cubiceos.NewvdWCfg(Tsi, Psi, Tcsi, Pcsi, internalR)
		rkCfg := cubiceos.NewRKCfg(Tsi, Psi, Tcsi, Pcsi, internalR)
		srkCfg := cubiceos.NewSRKCfg(Tsi, Psi, Tcsi, Pcsi, omega, internalR)
		prCfg := cubiceos.NewPRCfg(Tsi, Psi, Tcsi, Pcsi, omega, internalR)

		calcAB := func(cfg cubiceos.EOSCfg) (float64, float64) {
			tr := cfg.T / cfg.Tc
			a := cfg.Type.Params().Psi * cfg.Type.Alpha(tr, cfg.W) * cfg.R * cfg.R * cfg.Tc * cfg.Tc / cfg.Pc
			b := cfg.Type.Params().Omega * cfg.R * cfg.Tc / cfg.Pc
			// Scale for display: a in (P_unit · V_unit^2), b in V_unit
			aDisp := a / (pFac * vFac * vFac)
			bDisp := b / vFac
			return aDisp, bDisp
		}

		collect := func(name string, cfg cubiceos.EOSCfg, include bool) *pages.EOSResult {
			if !include {
				return nil
			}
			a, b := calcAB(cfg)
			roots, err := cubiceos.CubicEOS(cfg)
			if err != nil {
				return &pages.EOSResult{Name: name, Classification: "error", Error: err.Error(), A: a, B: b, AUnit: fmt.Sprintf("%s·(%s)²", pLabel, vLabel), BUnit: vLabel}
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
			res := &pages.EOSResult{Name: name, A: a, B: b, AUnit: fmt.Sprintf("%s·(%s)²", pLabel, vLabel), BUnit: vLabel}
			switch len(positives) {
			case 0:
				res.Classification = "none"
			case 1:
				res.Classification = "single-phase"
				v := positives[0] / vFac
				res.Vapor = &v
			case 2:
				// treat smaller as liquid, larger as vapor
				res.Classification = "two-phase"
				liq := positives[0] / vFac
				vap := positives[1] / vFac
				res.Liquid = &liq
				res.Vapor = &vap
			case 3:
				// Check for critical (all nearly equal)
				if math.Abs(positives[0]-positives[1]) < 1e-6 && math.Abs(positives[1]-positives[2]) < 1e-6 {
					res.Classification = "critical"
					v := positives[0] / vFac
					res.Vapor = &v
				} else {
					res.Classification = "two-phase"
					liq := positives[0] / vFac
					unst := positives[1] / vFac
					vap := positives[2] / vFac
					res.Liquid = &liq
					res.Unstable = &unst
					res.Vapor = &vap
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

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return errors.New("unsupported platform")
	}

}
func Run() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to find an open port: %v", err)
	}
	serverUrl, err := url.JoinPath("http://", ln.Addr().String())
	if err != nil {
		log.Fatalf("failed to join server url: %v", err)
	}

	mux := newSrvMux()
	var chainedHandler http.Handler = mux
	chainedHandler = loggingMiddleware(chainedHandler)
	chainedHandler = panicRecoveryMiddleware(chainedHandler)
	srv := &http.Server{
		Handler: chainedHandler,
	}

	go func() {
		log.Printf("Server running at %v", serverUrl)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	go func() {
		time.Sleep(200 * time.Millisecond)
		if err := openBrowser(serverUrl); err != nil {
			log.Printf("error opening browser: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	inactive := make(chan struct{})

	go func() {
		timeout := 3 * time.Minute
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			last := atomic.LoadInt64(&lastActivity)
			if time.Since(time.Unix(last, 0)) > timeout {
				log.Println("Re-run server with 'eos-cli --http'")
				close(inactive)
				return
			}
		}
	}()

	select {
	case <-quit:
		log.Println("Recieved interupt signal")
	case <-inactive:
		log.Println("Inactivity timeout reached")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Println("server forced to shut down")
	}
	log.Println("server exiting, bye...")

}
