package botguard

import (
	"context"
	"time"
)

// Mode defines how Botguard solving is used.
type Mode int

const (
	// Off disables Botguard usage entirely.
	Off Mode = iota
	// Auto enables Botguard on demand (e.g., after 403/permission errors) or preflight if configured.
	Auto
	// Force always runs Botguard attestation before relevant Innertube calls.
	Force
)

// Input carries the parameters required to perform Botguard attestation.
type Input struct {
	UserAgent        string
	PageURL          string
	ClientName       string
	ClientVersion    string
	VisitorID        string
	AdditionalParams map[string]string
}

// Output contains attestation result to be applied to Innertube requests.
type Output struct {
	Token     string
	ExpiresAt time.Time
	// Optional metadata for diagnostics or advanced integrations
	Metadata map[string]string
}

// Solver is an interface for Botguard attestation providers.
type Solver interface {
	Attest(ctx context.Context, input Input) (Output, error)
}

// Cache is an optional interface for storing Botguard outputs keyed by input characteristics.
type Cache interface {
	Get(key string) (Output, bool)
	Set(key string, value Output)
}

// KeyFromInput derives a cache key from Input fields that influence the attestation result.
func KeyFromInput(in Input) string {
	// Simple concatenation; callers may hash if needed
	return in.UserAgent + "|" + in.ClientName + "|" + in.ClientVersion + "|" + in.VisitorID
}
