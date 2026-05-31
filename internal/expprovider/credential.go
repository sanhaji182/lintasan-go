package expprovider

// G4 — Credential injection (containment Invariant 3).
//
// INVARIANT 3 (locked): the Experimental adapter NEVER receives the core
// credential store. A provider's token is injected into the child subprocess
// environment at launch, scoped to THAT provider only, and is never persisted
// in the LaunchSpec (which is loggable) nor handed to the adapter as a Go value
// it could exfiltrate. The child sees EXACTLY: the spec's non-secret BaseEnv +
// the one credential variable this Injector adds.
//
// This is deliberately a tiny, auditable surface: one function that takes a
// (spec, secret) and returns the final env slice. The harness (G5) probes that
// no OTHER secret is reachable; this file guarantees only the intended one is
// present.

import (
	"errors"
	"strings"
)

// CredentialSource resolves the secret VALUE for a provider at launch time. It
// is supplied by the host (which owns the secure store); the adapter only ever
// holds a CredentialSource, never the store itself. Returning ("", false) means
// "no credential available" — the caller decides whether that is fatal (it is,
// for api_key/oauth modes).
//
// Implementations MUST scope lookups to the named provider and MUST NOT return
// another provider's secret. A test/double can implement this trivially; the
// real host backs it with the encrypted per-provider store (E2 territory).
type CredentialSource interface {
	Credential(provider string) (secret string, ok bool)
}

// CredentialSourceFunc adapts a function to CredentialSource.
type CredentialSourceFunc func(provider string) (string, bool)

// Credential implements CredentialSource.
func (f CredentialSourceFunc) Credential(provider string) (string, bool) { return f(provider) }

// Injector builds the final child environment for a launch, adding exactly the
// provider's credential (per the spec's AuthMode/AuthEnvVar) on top of the
// non-secret BaseEnv. It holds a CredentialSource, never raw secrets.
type Injector struct {
	src CredentialSource
}

// NewInjector returns an Injector backed by src.
func NewInjector(src CredentialSource) *Injector {
	return &Injector{src: src}
}

// Credential-injection errors.
var (
	ErrNoCredentialSource = errors.New("expprovider: no credential source configured")
	ErrCredentialMissing  = errors.New("expprovider: no credential available for provider")
	ErrBaseEnvHasSecret   = errors.New("expprovider: BaseEnv must not contain the auth env var (secret leak)")
)

// BuildEnv returns the final environment for launching spec's agent. It is the
// ONLY place a secret enters the child env, and it adds at most one variable.
//
// Rules enforced:
//   - AuthNone: returns BaseEnv unchanged (no secret added).
//   - AuthAPIKey/AuthOAuth: the spec's BaseEnv MUST NOT already define
//     AuthEnvVar (guards against a spec accidentally baking a secret); the
//     credential is resolved from the source and appended as AuthEnvVar=secret.
//   - A missing credential for a mode that requires one is a hard error (we do
//     NOT launch an agent that will fail-auth and possibly retry-loop).
//
// The returned slice is a fresh copy; the spec's BaseEnv is never mutated.
func (in *Injector) BuildEnv(spec LaunchSpec) ([]string, error) {
	base := append([]string(nil), spec.BaseEnv...)

	if spec.AuthMode == AuthNone {
		return base, nil
	}

	// api_key / oauth need a source + the BaseEnv must not pre-set the var.
	prefix := spec.AuthEnvVar + "="
	for _, kv := range base {
		if strings.HasPrefix(kv, prefix) {
			return nil, ErrBaseEnvHasSecret
		}
	}
	if in == nil || in.src == nil {
		return nil, ErrNoCredentialSource
	}
	secret, ok := in.src.Credential(spec.Name)
	if !ok || secret == "" {
		return nil, ErrCredentialMissing
	}
	return append(base, spec.AuthEnvVar+"="+secret), nil
}
