package plugin

import (
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
