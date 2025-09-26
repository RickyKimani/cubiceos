package web

import (
	"context"
	"errors"
	"fmt"
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
	"syscall"
	"time"

	"github.com/rickykimani/cubiceos"
	"github.com/rickykimani/cubiceos/internal/web/pages"
)

func newSrvMux() *http.ServeMux {
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

		calcAB := func(cfg cubiceos.EOSCfg) (float64, float64) {
			tr := cfg.T / cfg.Tc
			a := cfg.Type.Params().Psi * cfg.Type.Alpha(tr, cfg.W) * cfg.R * cfg.R * cfg.Tc * cfg.Tc / cfg.Pc
			b := cfg.Type.Params().Omega * cfg.R * cfg.Tc / cfg.Pc
			return a, b
		}

		collect := func(name string, cfg cubiceos.EOSCfg, include bool) *pages.EOSResult {
			if !include {
				return nil
			}
			a, b := calcAB(cfg)
			roots, err := cubiceos.CubicEOS(cfg)
			if err != nil {
				return &pages.EOSResult{Name: name, Classification: "error", Error: err.Error(), A: a, B: b}
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
			res := &pages.EOSResult{Name: name, A: a, B: b}
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

	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Println("server forced to shut down")
	}
	log.Println("server exiting, bye...")

}
