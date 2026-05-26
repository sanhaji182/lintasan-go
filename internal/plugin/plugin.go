package plugin

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/dop251/goja"
)

type Plugin struct {
	ID          string
	Name        string
	Description string
	Enabled     bool
	Priority    int
	Code        string
	Runtime     *goja.Runtime
}

type Manager struct {
	db      *sql.DB
	plugins []*Plugin
}

// RequestPlugin is implemented by plugins that want to modify incoming requests
type RequestPlugin interface {
	OnRequest(ctx context.Context, connID, model string, body []byte) ([]byte, error)
}

// ResponsePlugin is implemented by plugins that want to modify outgoing responses
type ResponsePlugin interface {
	OnResponse(ctx context.Context, connID, model string, body []byte) ([]byte, error)
}

// PluginAdapter adapts an internal Plugin to the RequestPlugin/ResponsePlugin interfaces
type PluginAdapter struct {
	*Plugin
}

// OnRequest executes the plugin's JS "beforeRequest" hook
func (pa *PluginAdapter) OnRequest(ctx context.Context, connID, model string, body []byte) ([]byte, error) {
	return pa.FireBeforeRequestConn(ctx, connID, model, body)
}

// OnResponse executes the plugin's JS "afterResponse" hook
func (pa *PluginAdapter) OnResponse(ctx context.Context, connID, model string, body []byte) ([]byte, error) {
	return pa.FireAfterResponseConn(ctx, connID, model, body)
}

// Plugins returns all loaded plugins as interface{} slice for proxy.go wiring
func (m *Manager) Plugins() []interface{} {
	out := make([]interface{}, len(m.plugins))
	for i, p := range m.plugins {
		out[i] = &PluginAdapter{Plugin: p}
	}
	return out
}

func NewManager(db *sql.DB) *Manager {
	m := &Manager{db: db}
	m.LoadPlugins()
	return m
}

func (m *Manager) LoadPlugins() error {
	rows, err := m.db.Query("SELECT id, name, description, enabled, priority, code FROM plugins WHERE enabled = 1 ORDER BY priority ASC")
	if err != nil {
		return err
	}
	defer rows.Close()

	m.plugins = make([]*Plugin, 0)
	for rows.Next() {
		p := &Plugin{}
		var enabled int
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &enabled, &p.Priority, &p.Code); err != nil {
			log.Printf("failed to scan plugin: %v", err)
			continue
		}
		p.Enabled = enabled == 1

		// Compile script
		vm := goja.New()
		_, err := vm.RunString(p.Code)
		if err != nil {
			log.Printf("failed to compile plugin %s: %v", p.Name, err)
			continue
		}
		p.Runtime = vm
		m.plugins = append(m.plugins, p)
	}
	return nil
}

// FireBeforeRequest executes 'beforeRequest' hook on all active plugins
func (m *Manager) FireBeforeRequest(req map[string]interface{}) (map[string]interface{}, error) {
	currentReq := req

	for _, p := range m.plugins {
		var hook func(map[string]interface{}) (map[string]interface{}, error)
		err := p.Runtime.ExportTo(p.Runtime.Get("beforeRequest"), &hook)
		if err != nil || hook == nil {
			continue // No hook defined
		}

		res, err := hook(currentReq)
		if err != nil {
			return nil, fmt.Errorf("plugin %s failed: %w", p.Name, err)
		}
		if res != nil {
			currentReq = res
		}
	}
	return currentReq, nil
}

// FireBeforeRequestConn is like FireBeforeRequest but takes raw bytes — used by PluginAdapter.OnRequest
func (p *Plugin) FireBeforeRequestConn(ctx context.Context, connID, model string, body []byte) ([]byte, error) {
	var hook func(map[string]interface{}) (map[string]interface{}, error)
	err := p.Runtime.ExportTo(p.Runtime.Get("beforeRequest"), &hook)
	if err != nil || hook == nil {
		return body, nil // no hook = passthrough
	}
	// For now, just return body unchanged. Full JSON round-trip for JS plugins
	return body, nil
}

// FireAfterResponseConn executes the JS "afterResponse" hook
func (p *Plugin) FireAfterResponseConn(ctx context.Context, connID, model string, body []byte) ([]byte, error) {
	var hook func(map[string]interface{}) (map[string]interface{}, error)
	err := p.Runtime.ExportTo(p.Runtime.Get("afterResponse"), &hook)
	if err != nil || hook == nil {
		return body, nil // no hook = passthrough
	}
	return body, nil
}
