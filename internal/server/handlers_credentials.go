package server

// handlers_credentials.go — Credential Management V1 API endpoints.
//
// Admin-only endpoints for managing Experimental provider credentials from the
// dashboard. All endpoints are behind authMiddleware (fail-closed). Secrets are
// NEVER returned in full — only masked values.
//
// Endpoints:
//   GET    /api/experimental/credentials              — status of all providers
//   GET    /api/experimental/credentials/{name}       — status of one provider
//   PUT    /api/experimental/credentials/{name}       — set credential
//   DELETE /api/experimental/credentials/{name}       — remove credential

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sanhaji182/lintasan-go/internal/expprovider"
)

// registerCredentialRoutes wires the credential management API endpoints.
func (s *Server) registerCredentialRoutes() {
	s.mux.HandleFunc("GET /api/experimental/credentials", s.handleCredentialList)
	s.mux.HandleFunc("GET /api/experimental/credentials/{name}", s.handleCredentialStatus)
	s.mux.HandleFunc("PUT /api/experimental/credentials/{name}", s.handleCredentialSet)
	s.mux.HandleFunc("DELETE /api/experimental/credentials/{name}", s.handleCredentialDelete)
}

// credStore returns the credential store (lazily uses the master key from DB).
func (s *Server) credStore() *expprovider.DashboardCredentialStore {
	masterKey, _ := s.db.GetSetting("master_key")
	return expprovider.NewDashboardCredentialStore(s.db.Conn(), masterKey)
}

// handleCredentialList returns credential status for all Cohort-A providers.
func (s *Server) handleCredentialList(w http.ResponseWriter, r *http.Request) {
	store := s.credStore()
	ctx := r.Context()
	descriptors := expprovider.CohortADescriptors()

	var statuses []expprovider.CredentialStatus
	for _, d := range descriptors {
		status := store.GetStatus(ctx, d.Name, d.AuthEnvVar)
		statuses = append(statuses, status)
	}

	writeData(w, statuses)
}

// handleCredentialStatus returns credential status for one provider.
func (s *Server) handleCredentialStatus(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
		return
	}

	desc := findDescriptor(name)
	if desc == nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "provider not found in Cohort-A catalog"})
		return
	}

	store := s.credStore()
	status := store.GetStatus(r.Context(), desc.Name, desc.AuthEnvVar)
	writeData(w, status)
}

// handleCredentialSet stores an encrypted credential for a provider.
func (s *Server) handleCredentialSet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
		return
	}

	desc := findDescriptor(name)
	if desc == nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "provider not found in Cohort-A catalog"})
		return
	}

	var body struct {
		Credential string `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
		return
	}
	if strings.TrimSpace(body.Credential) == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "credential cannot be empty"})
		return
	}

	store := s.credStore()
	if err := store.SetCredential(r.Context(), name, body.Credential); err != nil {
		writeJSON(w, map[string]any{"error": "failed to store credential: " + err.Error()})
		return
	}

	// Return updated status
	status := store.GetStatus(r.Context(), desc.Name, desc.AuthEnvVar)
	writeData(w, map[string]any{
		"provider": name,
		"status":   status,
		"message":  "credential stored successfully",
	})
}

// handleCredentialDelete removes a stored credential for a provider.
func (s *Server) handleCredentialDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
		return
	}

	desc := findDescriptor(name)
	if desc == nil {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{"error": "provider not found in Cohort-A catalog"})
		return
	}

	store := s.credStore()
	if err := store.DeleteCredential(r.Context(), name); err != nil {
		writeJSON(w, map[string]any{"error": "failed to delete credential: " + err.Error()})
		return
	}

	// Return updated status (will show env or none)
	status := store.GetStatus(r.Context(), desc.Name, desc.AuthEnvVar)
	writeData(w, map[string]any{
		"provider": name,
		"status":   status,
		"message":  "credential removed",
	})
}
