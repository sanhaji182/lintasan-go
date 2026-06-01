package server

// experimental_bootstrap.go — R1: Bootstrap Hydration for Experimental Providers.
//
// On server startup, this reads the experimental_providers table and registers
// ACPProvider instances for every provider in "active" (or "admitted") state into
// the ProxyHandler's provider registry. This closes the gap where a provider
// activated via the dashboard would lose its runtime presence after a restart.
//
// DESIGN:
//   - Reads persisted records from SQLite via the existing SQLiteStore.
//   - For each "active" provider, looks up the matching CohortA descriptor,
//     builds a LaunchSpec + credential source, and constructs an ACPProvider.
//   - Registers the ACPProvider into providerReg (Track()==Experimental, so
//     membrane invariant is preserved — ResolveRoutable can never return it).
//   - Errors are logged but non-fatal: a missing credential at boot means the
//     provider is registered without being launchable (Run will fail with
//     ErrCredentialMissing, which is the correct runtime behavior).
//
// INVARIANT PRESERVATION:
//   - Only Experimental-track providers are registered here.
//   - ResolveRoutable filters TrackOfficial only — unchanged.
//   - Official routing pool is never touched.

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sanhaji182/lintasan-go/internal/expprovider"
	"github.com/sanhaji182/lintasan-go/internal/provider"
)

// hydrateExperimentalProviders loads persisted experimental provider state from
// the database and registers active providers into the runtime registry. Called
// once during NewProxyHandler initialization, after initProviderSDK.
func (p *ProxyHandler) hydrateExperimentalProviders() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store := expprovider.NewSQLiteStore(p.db.Conn())
	records, err := store.List(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[experimental] bootstrap: failed to load providers from DB: %v\n", err)
		return
	}

	// Build the credential source with priority: dashboard > env.
	masterKey, _ := p.db.GetSetting("master_key")
	credStore := expprovider.NewDashboardCredentialStore(p.db.Conn(), masterKey)

	// Build env var mapping from descriptors.
	descriptors := expprovider.CohortADescriptors()
	descMap := make(map[string]expprovider.ProviderDescriptor, len(descriptors))
	envMap := make(map[string]string, len(descriptors))
	for _, d := range descriptors {
		descMap[d.Name] = d
		if d.AuthEnvVar != "" {
			envMap[d.Name] = d.AuthEnvVar
		}
	}

	// DashboardCredentialSource: dashboard credential > env var > missing.
	credSrc := expprovider.NewDashboardCredentialSource(credStore, envMap)

	registered := 0
	for _, rec := range records {
		// Only hydrate providers that are active or admitted (admitted providers
		// are registered but dormant — they can be activated without restart).
		if rec.State != "active" && rec.State != "admitted" {
			continue
		}

		desc, ok := descMap[rec.Name]
		if !ok {
			fmt.Fprintf(os.Stderr, "[experimental] bootstrap: provider %q in DB but not in Cohort-A catalog — skipping\n", rec.Name)
			continue
		}

		// Build the ACPProvider from the descriptor.
		spec := desc.LaunchSpec("", nil, nil)
		caps := desc.Capabilities
		acpProvider := expprovider.NewACPProvider(spec, caps, expprovider.NewInjector(credSrc))

		// Register into the provider registry. Track()==Experimental ensures
		// membrane invariant: ResolveRoutable will never return this provider.
		if err := p.providerReg.Register(acpProvider); err != nil {
			fmt.Fprintf(os.Stderr, "[experimental] bootstrap: failed to register %q: %v\n", rec.Name, err)
			continue
		}

		registered++
		fmt.Fprintf(os.Stderr, "[experimental] bootstrap: registered %q (state=%s, track=%s)\n",
			rec.Name, rec.State, provider.TrackExperimental)
	}

	if registered > 0 {
		fmt.Fprintf(os.Stderr, "[experimental] bootstrap: %d provider(s) hydrated into registry\n", registered)
	}
}
