import { useState } from 'react'
import { useApi } from '../hooks/useApi'

export default function Connections() {
  const { data: connections, reload } = useApi('/api/connections')
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({ name: '', base_url: '', api_key: '', format: 'openai' })

  const addConnection = async () => {
    await fetch('/api/connections', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form)
    })
    setForm({ name: '', base_url: '', api_key: '', format: 'openai' })
    setShowAdd(false)
    reload()
  }

  const deleteConnection = async (id) => {
    if (!confirm('Delete this connection?')) return
    await fetch(`/api/connections/${id}`, { method: 'DELETE' })
    reload()
  }

  return (
    <div className="fade-in">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: 700 }}>Connections</h2>
          <p style={{ fontSize: '13px', color: 'var(--fg-2)' }}>Manage your LLM provider connections</p>
        </div>
        <button onClick={() => setShowAdd(!showAdd)} style={{ padding: '10px 18px', background: 'var(--primary)', color: '#fff', border: 'none', borderRadius: 'var(--radius-sm)', fontSize: '13px', fontWeight: 500, cursor: 'pointer' }}>
          + Add Connection
        </button>
      </div>

      {showAdd && (
        <div className="card" style={{ marginBottom: '20px' }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
            <input placeholder="Name (e.g. Sumopod)" value={form.name} onChange={e => setForm({...form, name: e.target.value})}
              style={inputStyle} />
            <input placeholder="Base URL (e.g. https://api.openai.com)" value={form.base_url} onChange={e => setForm({...form, base_url: e.target.value})}
              style={inputStyle} />
            <input placeholder="API Key" type="password" value={form.api_key} onChange={e => setForm({...form, api_key: e.target.value})}
              style={inputStyle} />
            <select value={form.format} onChange={e => setForm({...form, format: e.target.value})} style={inputStyle}>
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
              <option value="custom">Custom</option>
            </select>
          </div>
          <div style={{ marginTop: '12px', display: 'flex', gap: '8px' }}>
            <button onClick={addConnection} style={{ padding: '8px 16px', background: 'var(--success)', color: '#fff', border: 'none', borderRadius: '6px', fontSize: '13px', cursor: 'pointer' }}>Save</button>
            <button onClick={() => setShowAdd(false)} style={{ padding: '8px 16px', background: 'var(--bg-body)', color: 'var(--fg-1)', border: '1px solid var(--border)', borderRadius: '6px', fontSize: '13px', cursor: 'pointer' }}>Cancel</button>
          </div>
        </div>
      )}

      {(!connections || connections.length === 0) ? (
        <div className="card">
          <div className="empty-state">
            <div className="icon">🔌</div>
            <p>No connections yet. Add your first LLM provider.</p>
          </div>
        </div>
      ) : (
        <div style={{ display: 'grid', gap: '12px' }}>
          {connections.map(c => (
            <div key={c.id} className="card" style={{ padding: '18px 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
                <div style={{ width: '40px', height: '40px', borderRadius: '10px', background: c.is_active ? 'var(--success-light)' : 'var(--bg-body)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '18px' }}>
                  {c.is_active ? '🟢' : '🔴'}
                </div>
                <div>
                  <div style={{ fontWeight: 600, fontSize: '14px' }}>{c.name}</div>
                  <div style={{ fontSize: '12px', color: 'var(--fg-3)', fontFamily: 'var(--mono)' }}>{c.base_url}</div>
                </div>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <span className="badge badge-info">{c.format || 'openai'}</span>
                <span style={{ fontSize: '12px', color: 'var(--fg-3)' }}>{c.models_count || 0} models</span>
                <button onClick={() => deleteConnection(c.id)} style={{ padding: '6px 10px', background: 'var(--error-light)', color: 'var(--error)', border: 'none', borderRadius: '6px', fontSize: '11px', cursor: 'pointer' }}>Delete</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

const inputStyle = { padding: '10px 14px', background: 'var(--bg-body)', border: '1px solid var(--border)', borderRadius: '8px', fontSize: '13px', color: 'var(--fg-0)', outline: 'none', width: '100%' }
