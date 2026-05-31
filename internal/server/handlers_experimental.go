package server

// handlers_experimental.go — P2: Experimental Provider API.
//
// Admin-only endpoints for managing the Experimental provider lifecycle from the
// dashboard. All endpoints are behind the existing authMiddleware (fail-closed:
// JWT/master-key/API-key required). Response shape follows the existing pattern:
// {"data": ...} for success, {"error": "..."} for failure.
//
// Endpoints:
//   GET    /api/experimental/providers          — list all providers + state
//   GET    /api/experimental/providers/{name}   — detail one provider
//   POST   /api/experimental/providers/{name}/admit      — run admission
//   POST   /api/experimental/providers/{name}/activate   — admitted → active
//   POST   /api/experimental/providers/{name}/deactivate — active → deprecated

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/sanhaji182/lintasan-go/internal/expprovider"
)

// registerExperimentalRoutes wires the experimental provider API endpoints.
// Called from server route registration.
func (s *Server) registerExperimentalRoutes() {
	s.mux.HandleFunc("GET /api/experimental/providers", s.handleExpProviderList)
	s.mux.HandleFunc("GET /api/experimental/providers/{name}", s.handleExpProviderDetail)
	s.mux.HandleFunc("POST /api/experimental/providers/{name}/admit", s.handleExpProviderAdmit)
	s.mux.HandleFunc("POST /api/experimental/providers/{name}/activate", s.handleExpProviderActivate)
	s.mux.HandleFunc("POST /api/experimental/providers/{name}/deactivate", s.handleExpProviderDeactivate)
}

// expStore returns the experimental provider store (lazily initialized from the
// application DB). Thread-safe: SQLiteStore is stateless beyond the *sql.DB.
func (s *Server) expStore() *expprovider.SQLiteStore {
	return expprovider.NewSQLiteStore(s.db.Conn())
}

// handleExpProviderList returns all Cohort-A providers with their persisted state
// + live credential status (env var set or not — never the value).
func (s *Server) handleExpProviderList(w http.ResponseWriter, r *http.Request) {
	store := s.expStore()
	ctx := r.Context()

	// Get persisted records
	records, err := store.List(ctx)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}

	// Build response: merge descriptor catalog with persisted state
	descriptors := expprovider.CohortADescriptors()
	type providerView struct {
		Name               string          `json:"name"`
		Track              string          `json:"track"`
		State              string          `json:"state"`
		RiskBadge          string          `json:"risk_badge"`
		AuthEnvVar         string          `json:"auth_env_var"`
		CredentialSet      bool            `json:"credential_set"`
		ValidationEvidence string          `json:"validation_evidence"`
		AdmittedAt         *time.Time      `json:"admitted_at"`
		ActivatedAt        *time.Time      `json:"activated_at"`
		Capabilities       []string        `json:"capabilities"`
		Descriptor         json.RawMessage `json:"descriptor"`
	}

	// Index persisted records by name
	recMap := make(map[string]*expprovider.ProviderRecord, len(records))
	for _, rec := range records {
		recMap[rec.Name] = rec
	}

	var providers []providerView
	for _, d := range descriptors {
		pv := providerView{
			Name:          d.Name,
			Track:         "experimental",
			State:         "proposed",
			RiskBadge:     "experimental",
			AuthEnvVar:    d.AuthEnvVar,
			CredentialSet: os.Getenv(d.AuthEnvVar) != "",
			Capabilities:  capNames(d),
		}
		// Serialize descriptor (without secrets)
		descBytes, _ := json.Marshal(descriptorView(d))
		pv.Descriptor = descBytes

		// Overlay persisted state if exists
		if rec, ok := recMap[d.Name]; ok {
			pv.State = rec.State
			pv.RiskBadge = rec.RiskBadge
			pv.ValidationEvidence = rec.ValidationEvidence
			pv.AdmittedAt = rec.AdmittedAt
			pv.ActivatedAt = rec.ActivatedAt
		}
		providers = append(providers, pv)
	}

	writeData(w, providers)
}

// handleExpProviderDetail returns full detail for one provider including the
// admission report.
func (s *Server) handleExpProviderDetail(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
		return
	}

	store := s.expStore()
	ctx := r.Context()

	// Find descriptor
	desc := findDescriptor(name)
	if desc == nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "provider not found in Cohort-A catalog"})
		return
	}

	// Get persisted record
	rec, err := store.Get(ctx, name)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}

	type detailView struct {
		Name               string          `json:"name"`
		Track              string          `json:"track"`
		State              string          `json:"state"`
		RiskBadge          string          `json:"risk_badge"`
		AuthEnvVar         string          `json:"auth_env_var"`
		AuthMethodID       string          `json:"auth_method_id"`
		CredentialSet      bool            `json:"credential_set"`
		ValidationEvidence string          `json:"validation_evidence"`
		AdmittedAt         *time.Time      `json:"admitted_at"`
		ActivatedAt        *time.Time      `json:"activated_at"`
		DeactivatedAt      *time.Time      `json:"deactivated_at"`
		Capabilities       []string        `json:"capabilities"`
		Descriptor         json.RawMessage `json:"descriptor"`
		AdmissionReport    json.RawMessage `json:"admission_report"`
		DefaultPath        string          `json:"default_path"`
		Args               []string        `json:"args"`
		ForeignAuthVars    []string        `json:"foreign_auth_vars"`
	}

	dv := detailView{
		Name:            desc.Name,
		Track:           "experimental",
		State:           "proposed",
		RiskBadge:       "experimental",
		AuthEnvVar:      desc.AuthEnvVar,
		AuthMethodID:    desc.AuthMethodID,
		CredentialSet:   os.Getenv(desc.AuthEnvVar) != "",
		Capabilities:    capNames(*desc),
		DefaultPath:     desc.DefaultPath,
		Args:            desc.Args,
		ForeignAuthVars: desc.ForeignAuthVars,
	}
	descBytes, _ := json.Marshal(descriptorView(*desc))
	dv.Descriptor = descBytes

	if rec != nil {
		dv.State = rec.State
		dv.RiskBadge = rec.RiskBadge
		dv.ValidationEvidence = rec.ValidationEvidence
		dv.AdmittedAt = rec.AdmittedAt
		dv.ActivatedAt = rec.ActivatedAt
		dv.DeactivatedAt = rec.DeactivatedAt
		dv.AdmissionReport = rec.AdmissionReportJSON
	}

	writeData(w, dv)
}

// handleExpProviderAdmit runs the admission flow (fixture-based, in-process) and
// persists the result. Does NOT trigger live validation (that remains CLI-only).
func (s *Server) handleExpProviderAdmit(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	desc := findDescriptor(name)
	if desc == nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "provider not found in Cohort-A catalog"})
		return
	}

	// Check credential availability
	if os.Getenv(desc.AuthEnvVar) == "" {
		writeJSONStatus(w, http.StatusPreconditionFailed, map[string]any{
			"error":    "credential not available",
			"env_var":  desc.AuthEnvVar,
			"hint":     "set " + desc.AuthEnvVar + " in the server environment before admission",
			"provider": name,
		})
		return
	}

	store := s.expStore()
	ctx := r.Context()

	// Build credential source for this provider
	src := expprovider.CredentialSourceFunc(func(p string) (string, bool) {
		if p == desc.Name {
			v := os.Getenv(desc.AuthEnvVar)
			return v, v != ""
		}
		return "", false
	})

	// Run admission (fixture-based, bounded timeout)
	admitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	spec := desc.LaunchSpec("", nil, nil)
	p, _, rep, err := expprovider.AdmitProvider(admitCtx, nil, spec, desc.Capabilities, src, desc.ForeignAuthVars)
	if p != nil {
		defer p.StopAgent()
	}

	now := time.Now().UTC()
	reportJSON, _ := json.Marshal(rep)
	descJSON, _ := json.Marshal(descriptorView(*desc))

	state := "admitted"
	evidence := "fixture-pass"
	if err != nil {
		state = "proposed"
		evidence = "admission-error: " + err.Error()
	} else if !rep.Go() {
		state = "proposed"
		evidence = "admission-no-go"
	}

	rec := &expprovider.ProviderRecord{
		Name:                name,
		Track:               "experimental",
		State:               state,
		DescriptorJSON:      descJSON,
		AdmissionReportJSON: reportJSON,
		ValidationEvidence:  evidence,
		RiskBadge:           "experimental",
	}
	if state == "admitted" {
		rec.AdmittedAt = &now
	}

	if saveErr := store.Save(ctx, rec); saveErr != nil {
		writeJSON(w, map[string]any{"error": "admission completed but persistence failed: " + saveErr.Error()})
		return
	}

	writeData(w, map[string]any{
		"provider": name,
		"state":    state,
		"go":       err == nil && rep.Go(),
		"report":   rep,
		"evidence": evidence,
	})
}

// handleExpProviderActivate transitions an admitted provider to active.
func (s *Server) handleExpProviderActivate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	store := s.expStore()
	ctx := r.Context()

	rec, err := store.Get(ctx, name)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	if rec == nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "provider not found — run admission first"})
		return
	}
	if rec.State != "admitted" {
		writeJSONStatus(w, http.StatusConflict, map[string]any{
			"error":         "provider must be in 'admitted' state to activate",
			"current_state": rec.State,
		})
		return
	}

	now := time.Now().UTC()
	if err := store.UpdateState(ctx, name, "active", now); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}

	writeData(w, map[string]any{"provider": name, "state": "active", "activated_at": now})
}

// handleExpProviderDeactivate transitions an active provider to deprecated.
func (s *Server) handleExpProviderDeactivate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	store := s.expStore()
	ctx := r.Context()

	rec, err := store.Get(ctx, name)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	if rec == nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "provider not found"})
		return
	}
	if rec.State != "active" && rec.State != "admitted" {
		writeJSONStatus(w, http.StatusConflict, map[string]any{
			"error":         "provider must be 'active' or 'admitted' to deactivate",
			"current_state": rec.State,
		})
		return
	}

	now := time.Now().UTC()
	if err := store.UpdateState(ctx, name, "deprecated", now); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}

	writeData(w, map[string]any{"provider": name, "state": "deprecated", "deactivated_at": now})
}

// --- helpers ---

func findDescriptor(name string) *expprovider.ProviderDescriptor {
	for _, d := range expprovider.CohortADescriptors() {
		if d.Name == name {
			return &d
		}
	}
	return nil
}

func capNames(d expprovider.ProviderDescriptor) []string {
	var names []string
	for _, c := range d.Capabilities.List() {
		names = append(names, string(c))
	}
	return names
}

// descriptorView returns a JSON-safe view of a descriptor (no secrets, no
// internal types). This is what gets persisted and returned via API.
func descriptorView(d expprovider.ProviderDescriptor) map[string]any {
	return map[string]any{
		"name":              d.Name,
		"default_path":      d.DefaultPath,
		"args":              d.Args,
		"auth_mode":         string(d.AuthMode),
		"auth_env_var":      d.AuthEnvVar,
		"auth_method_id":    d.AuthMethodID,
		"foreign_auth_vars": d.ForeignAuthVars,
	}
}
