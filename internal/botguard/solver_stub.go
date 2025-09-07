//go:build !botguard

package botguard

import "context"

// NewGojaSolver is a stub when the 'botguard' build tag is not enabled.
// It returns nil to indicate Botguard is unavailable in this build.
type GojaSolver struct{}

func NewGojaSolver() *GojaSolver { return nil }

// NewGojaSolverWithScript is a stub constructor for non-botguard builds.
func NewGojaSolverWithScript(scriptPath string) *GojaSolver { return nil }

func (s *GojaSolver) Attest(ctx context.Context, input Input) (Output, error) {
	return Output{}, nil
}
