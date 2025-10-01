package pages

// EOSResult represents interpreted cubic EOS roots for display.
type EOSResult struct {
	Name           string
	Classification string // single-phase | two-phase | critical | none
	Liquid         *float64
	Unstable       *float64
	Vapor          *float64
	A              float64 // a(T)
	AUnit          string  // units label for a(T), e.g., "Pa·(cm³/mol)²"
	B              float64 // b
	BUnit          string  // units label for b, e.g., "cm³/mol"
	Error          string  // error message from solver (if any)
}
