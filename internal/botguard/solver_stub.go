//go:build !botguard

package botguard

import "context"

// GojaSolver is a stub when the 'botguard' build tag is not enabled.
// It returns nil to indicate Botguard is unavailable in this build.
type GojaSolver struct{}

// NewGojaSolver creates a new GojaSolver stub
func NewGojaSolver() *GojaSolver { return nil }

// NewGojaSolverWithScript is a stub constructor for non-botguard builds.
func NewGojaSolverWithScript(scriptPath string) *GojaSolver { return nil }

// Attest performs Botguard attestation (stub implementation)
func (s *GojaSolver) Attest(ctx context.Context, input Input) (Output, error) {
	return Output{}, nil
}
