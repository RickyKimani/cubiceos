package pages

// EOSResult represents interpreted cubic EOS roots for display.
type EOSResult struct {
	Name           string
	Classification string // single-phase | two-phase | critical | none
	Liquid         *float64
	Unstable       *float64
	Vapor          *float64
	A              float64 // a(T)
	B              float64 // b
	Error          string  // error message from solver (if any)
}
