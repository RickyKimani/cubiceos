# cubiceos

[![CI](https://github.com/RickyKimani/cubiceos/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/RickyKimani/cubiceos/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rickykimani/cubiceos.svg)](https://pkg.go.dev/github.com/rickykimani/cubiceos)
[![Go Report Card](https://goreportcard.com/badge/github.com/rickykimani/cubiceos)](https://goreportcard.com/report/github.com/rickykimani/cubiceos)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Solvers and tools for cubic Equations of State (EOS) in Go.

Supported EOS:
- Van der Waals (vdW)
- Redlich–Kwong (RK)
- Soave–Redlich–Kwong (SRK)
- Peng–Robinson (PR)

This repository contains three parts:
1) Library: `github.com/rickykimani/cubiceos` — reusable Go package for EOS calculations
2) TUI: an interactive terminal app to explore EOS results quickly
3) Web UI: a simple browser app for solving and visualizing results

## Table of Contents

- [Part I — Library](#part-i-library) (`github.com/rickykimani/cubiceos`)
	- [General cubic EOS](#cubic)
	- [Requirements](#requirements)
	- [Install](#install-lib)
	- [Usage](#usage)
	- [API overview](#api-overview)
	- [Interpreting results](#interpreting-results)
	- [Example program](#example-program)
- [Part II — Terminal UI (TUI)](#part-ii-tui)
	- [Install](#install-cli)
	- [Run](#run)
	- [Controls & input](#controls-input)
	- [Results view](#results-view)
    - [Demo](#demo)
- [Part III — Web UI](#part-iii-web)
    - [Features](#web-features)
    - [Run](#web-run)
    - [Screenshots](#web-screens)
- [Project layout](#project-layout)
- [License](#license)

---

<a id="part-i-library"></a>
## Part I — Library (`github.com/rickykimani/cubiceos`)

<a id="cubic"></a>
### What is a cubic EOS?
- Polynomial equations that are cubic in molar volume are the simplest equations capable of 
representing the 𝑃𝑉𝑇 behaviour of both liquids and vapours for a wide range of temperatures, 
pressures, and molar volumes. 

```math
\begin{align*}
P &= \frac{RT}{V-b} - \frac{a(T)}{(V + \varepsilon b)(V + \sigma b)}\\
b &= \Omega\frac{R T_c}{P_c}\\
a(T) &= \Psi\frac{\alpha(T_r, \omega) R^2 {T_c}^2}{P_c}
\end{align*}
```
and the corresponding cubic in V:
```math
\begin{align*}
0 &= V^{3} 
+ \left[\;(\varepsilon + \sigma - 1)b - \frac{RT}{P}\;\right]V^{2} \\[6pt]
&\quad + \left\{\,b\left[(\varepsilon\sigma - \varepsilon - \sigma)b - (\varepsilon + \sigma)\frac{RT}{P}\right] + \frac{a(T)}{P}\,\right\}V \\[6pt]
&\quad -\,\varepsilon\sigma\, b^{2}\left(b + \frac{RT}{P}\right) - \frac{a(T)\,b}{P}.
\end{align*}
```

this describes
![cubic](/resources/sat.png)

<a id="requirements"></a>
### Requirements

- [Go 1.22+](https://go.dev/dl/)

<a id="install-lib"></a>
### Install

```powershell
go get github.com/rickykimani/cubiceos
```

<a id="usage"></a>
### Usage

```go
package main

import (
	"log"

	"github.com/rickykimani/cubiceos"
)

func main() {
    //n-butane example
	eq := cubiceos.NewRKCfg(
		350,    // T (K)
		9.4573, // P (bar)
		425.1,  // Tc (K)
		37.96,  // Pc (bar)
		83.14,  // R (bar•cm^3/(mol•K))
	)
	res, err := cubiceos.CubicEOS(eq)
	if err != nil {
		log.Fatal(err)
	}
	cubiceos.ResultPrinter(res)
}
```

<a id="api-overview"></a>
### API overview

- Builders:
  - `NewvdWCfg(T, P, Tc, Pc, R)`
  - `NewRKCfg(T, P, Tc, Pc, R)`
  - `NewSRKCfg(T, P, Tc, Pc, W, R)`
  - `NewPRCfg(T, P, Tc, Pc, W, R)`
- Call `CubicEOS(cfg)` to solve and return three roots (possibly complex).
- Print `ResultPrinter(roots)`

<a id="interpreting-results"></a>
### Interpreting results

`CubicEOS` returns three roots. Physical molar volumes are the real, positive roots:
- One positive root → single phase.
- Two positive roots (with one unstable in-between) → phase split (liquid/vapor).
- At critical conditions, the three real roots can coalesce.

For examples, see `example/main.go`:

```powershell
go run ./example
```

---

<a id="part-ii-tui"></a>
## Part II — Terminal UI (TUI)

<a id="install-cli"></a>
### Install

```powershell
go install github.com/rickykimani/cubiceos/cmd/eos-cli@latest
```

<a id="run"></a>
### Run

```powershell
eos-cli
```

<a id="controls-input"></a>
### Controls & input

- Navigate: ↑/↓ or `j`/`k`
- Select: `Enter`
- Back: `Esc`
- Quit: `q` or `Ctrl+C`
![Controls](/resources/select.png)
- Inputs: numeric values for `T`, `P`, `Tc`, `Pc`, `R`, and `omega` (for SRK/PR).
![Input](/resources/input.png)

<a id="results-view"></a>
### Results view

- Displays which EOS was used and the computed roots.
![Result](/resources/results.png)

<a id="demo"></a>
### Demo

![Demo](/resources/demo.gif)

---

<a id="part-iii-web"></a>
## Part III — Web UI

<a id="web-features"></a>
### Features

- Per-field unit selectors (T, P, Tc, Pc) with correct SI conversion internally
- Volume display units (e.g., m³/mol, cm³/mol) applied to results
- a(T) and b displayed with unit labels (a(T) in Punit·(Vunit)², b in Vunit)
- Clean results view with phase classification badges and error messages
- Dark mode with a top-right toggle; GitHub link in the header
- Embedded static assets (works when installed via `go install`, no external files required)

<a id="web-run"></a>
### Run

Quick start (opens browser on a local port):

```powershell
eos-cli --http
```


<a id="web-screens"></a>
### Screenshots

![Web Home](/resources/web.png)
![Web Results](/resources/web_result..png)

---

<a id="project-layout"></a>
## Project layout

- `cubiceos.go` — core types and `CubicEOS`
- `solve.go` — general cubic polynomial solver and helpers
- `vdw.go`, `rk.go`, `srk.go`, `pr.go` — EOS implementations and config builders
- `cmd/` — interactive terminal UI and web UI
- `example/` — minimal library usage example

<a id="license"></a>
## License

This project is licensed under the [MIT License](./LICENSE).

