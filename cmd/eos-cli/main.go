package main

import (
	"fmt"
	"log"
	"math"
	"slices"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rickykimani/cubiceos"
)

type eosOption string

const (
	vdW eosOption = "Van der Waals"
	RK  eosOption = "Redlich-Kwong"
	SRK eosOption = "Soave-Redlich-Kwong"
	PR  eosOption = "Peng-Robinson"
	ALL eosOption = "All"
)

type state int

const (
	stateMenu state = iota
	stateForm
	stateResult
)

type validator func(float64) error

func positive(name string) validator {
	return func(v float64) error {
		if v <= 0 {
			return fmt.Errorf("%s must be > 0", name)
		}
		return nil
	}
}

type field struct {
	name     string
	validate validator
}

type model struct {
	state     state
	choice    eosOption
	formIndex int
	inputs    []string
	parsed    map[string]float64
	results   string
	errMsg    string
	list      list.Model
}

var eosChoices = []list.Item{
	item(vdW), item(RK), item(SRK), item(PR), item(ALL),
}

type item string

func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return string(i) }

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	inputStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	boxStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(lipgloss.Color("240"))
	resultStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			Padding(1, 2).
			BorderForeground(lipgloss.Color("35"))
)

func initialModel() model {
	l := list.New(eosChoices, list.NewDefaultDelegate(), 30, 10)
	l.Title = "Select an Equation of State"
	return model{state: stateMenu, list: l, parsed: make(map[string]float64)}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateMenu:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if key, ok := msg.(tea.KeyMsg); ok {
			if key.String() == "q" || key.Type == tea.KeyCtrlC || key.Type == tea.KeyEsc {
				return m, tea.Quit
			}
			if key.Type == tea.KeyEnter {
				if sel := m.list.SelectedItem(); sel != nil {
					if it, ok := sel.(item); ok {
						m.choice = eosOption(it)
					} else {
						m.choice = eosOption(sel.FilterValue())
					}
					m.state = stateForm
					m.formIndex = 0
					m.inputs = []string{}
					m.errMsg = ""
					m.parsed = make(map[string]float64)
					return m, tea.ClearScreen
				}
			}
		}
		return m, cmd

	case stateForm:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.Type {
			case tea.KeyEnter:
				fields := m.requiredFields()
				// Ensure current input slot exists
				for len(m.inputs) <= m.formIndex {
					m.inputs = append(m.inputs, "")
				}
				inp := m.inputs[m.formIndex]

				// Default blank → "1"
				if inp == "" {
					inp = "1"
				}

				// Parse input
				v, err := strconv.ParseFloat(inp, 64)
				if err != nil {
					m.errMsg = fmt.Sprintf("Invalid number for %s: %q", fields[m.formIndex].name, inp)
					return m, nil
				}
				// Validate
				if verr := fields[m.formIndex].validate(v); verr != nil {
					m.errMsg = verr.Error()
					return m, nil
				}

				m.parsed[fields[m.formIndex].name] = v
				m.errMsg = ""

				// Next field or compute
				if m.formIndex >= len(fields)-1 {
					m.results = m.compute()
					m.state = stateResult
				} else {
					m.formIndex++
				}
			case tea.KeyRunes:
				if m.formIndex >= len(m.inputs) {
					m.inputs = append(m.inputs, "")
				}
				m.inputs[m.formIndex] += string(key.Runes)
			case tea.KeyBackspace:
				if m.formIndex < len(m.inputs) && len(m.inputs[m.formIndex]) > 0 {
					s := m.inputs[m.formIndex]
					m.inputs[m.formIndex] = s[:len(s)-1]
				}
			case tea.KeyEsc:
				m.state = stateMenu
			case tea.KeyCtrlC:
				return m, tea.Quit
			}
		}
		return m, nil

	case stateResult:
		if key, ok := msg.(tea.KeyMsg); ok {
			if key.String() == "q" || key.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
			if key.Type == tea.KeyEsc {
				m.state = stateMenu
			}
		}
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	switch m.state {
	case stateMenu:
		return m.list.View()

	case stateForm:
		fields := m.requiredFields()
		if m.formIndex < 0 || m.formIndex >= len(fields) {
			return "Preparing form..."
		}
		curr := fields[m.formIndex].name
		var val string
		if m.formIndex < len(m.inputs) {
			val = m.inputs[m.formIndex]
		}

		view := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(fmt.Sprintf("(%d/%d) Input Required", m.formIndex+1, len(fields))),
			labelStyle.Render("Enter "+curr+":"),
			inputStyle.Render(val),
		)

		if m.errMsg != "" {
			view += "\n" + errorStyle.Render("Error: "+m.errMsg)
		}

		view += "\n\n" + helpStyle.Render("Press Enter to continue. Press Esc to go back.")

		return boxStyle.Render(view)

	case stateResult:
		return resultStyle.Render(m.results +
			"\n\n" + helpStyle.Render("Press Esc to go back or q to quit."))

	default:
		return "Unknown state"
	}
}

func (m model) requiredFields() []field {
	base := []field{
		{"T", positive("T")},
		{"P", positive("P")},
		{"Tc", positive("Tc")},
		{"Pc", positive("Pc")},
		{"R", positive("R")},
	}
	if m.choice == SRK || m.choice == PR || m.choice == ALL {
		// omega can be negative
		base = append(base, field{"omega", func(v float64) error { return nil }})
	}
	return base
}

func calculateb(eq cubiceos.EOSCfg, m model) float64 {
	return eq.Type.Params().Omega * m.parsed["R"] * m.parsed["Tc"] / m.parsed["Pc"]
}

func resultPrinter(eq cubiceos.EOSCfg, m model) string {
	b := calculateb(eq, m)
	const eps = 1e-9

	roots, _ := cubiceos.CubicEOS(eq)

	fs := make([]float64, 0, 3)
	for _, v := range roots {
		if imag(v) < eps {
			fs = append(fs, real(v))
		}
	}
	slices.Sort(fs)

	// --- Lip Gloss styles ---
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	value := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	invalid := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Italic(true)

	out := header.Render(fmt.Sprintf("Results for %s EOS\n", eq.Type.Name()))

	formatRoot := func(name string, v float64) string {
		if v <= 0 {
			return label.Render(name+": ") + invalid.Render(fmt.Sprintf("%.4f (invalid, ≤0)", v))
		}
		if v <= b {
			return label.Render(name+": ") + invalid.Render(fmt.Sprintf("%.4f (invalid, <b=%.4f)", v, b))
		}
		return label.Render(name+": ") + value.Render(fmt.Sprintf("%.4f", v))
	}

	switch len(fs) {
	case 1:
		out += "\n" + formatRoot("Single phase molar volume", fs[0])
	case 2:
		out += "\n" + formatRoot("Root 1", fs[0])
		out += "\n" + formatRoot("Root 2", fs[1])
	case 3:
		if math.Abs(fs[0]-fs[1]) < eps && math.Abs(fs[1]-fs[2]) < eps {
			out += "\n" + formatRoot("Critical volume", fs[0])
		} else {
			out += "\n" + formatRoot("Liquid Vsat", fs[0])
			out += "\n" + formatRoot("Unstable root", fs[1])
			out += "\n" + formatRoot("Vapour Vsat", fs[2])
		}
	default:
		out += "\n" + invalid.Render("No real roots found")
	}

	return out
}

func (m model) compute() string {
	vCfg := cubiceos.NewvdWCfg(m.parsed["T"], m.parsed["P"], m.parsed["Tc"], m.parsed["Pc"], m.parsed["R"])
	rkCfg := cubiceos.NewRKCfg(m.parsed["T"], m.parsed["P"], m.parsed["Tc"], m.parsed["Pc"], m.parsed["R"])
	srkCfg := cubiceos.NewSRKCfg(m.parsed["T"], m.parsed["P"], m.parsed["Tc"], m.parsed["Pc"], m.parsed["omega"], m.parsed["R"])
	prCfg := cubiceos.NewPRCfg(m.parsed["T"], m.parsed["P"], m.parsed["Tc"], m.parsed["Pc"], m.parsed["omega"], m.parsed["R"])

	// If the user selected ALL, render a 2x2 grid
	if m.choice == ALL {
		// Compute each result box
		resVdW := resultPrinter(vCfg, m)
		resRK := resultPrinter(rkCfg, m)
		resSRK := resultPrinter(srkCfg, m)
		resPR := resultPrinter(prCfg, m)

		// Small box style wrapper
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(lipgloss.Color("240"))

		// Render boxes
		boxVdW := boxStyle.Render(resVdW)
		boxRK := boxStyle.Render(resRK)
		boxSRK := boxStyle.Render(resSRK)
		boxPR := boxStyle.Render(resPR)

		// Build two vertical columns (each column contains two boxes stacked)
		leftCol := lipgloss.JoinVertical(lipgloss.Left, boxVdW, boxSRK)
		rightCol := lipgloss.JoinVertical(lipgloss.Left, boxRK, boxPR)

		// Put columns side-by-side with a small gap
		gap := lipgloss.NewStyle().PaddingLeft(2)
		grid := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, gap.Render(rightCol))

		return grid
	}

	// Single EOS mode
	switch m.choice {
	case vdW:
		return resultPrinter(vCfg, m)
	case RK:
		return resultPrinter(rkCfg, m)
	case SRK:
		return resultPrinter(srkCfg, m)
	case PR:
		return resultPrinter(prCfg, m)
	default:
		return "something went wrong"
	}

}

func main() {
	if _, err := tea.NewProgram(initialModel(), tea.WithAltScreen()).Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
