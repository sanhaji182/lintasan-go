import { useState } from 'react'
import { useApi } from '../hooks/useApi'

export default function Settings() {
  const { data: settings, loading, reload } = useApi('/api/settings')
  const [saving, setSaving] = useState(false)

  const updateSetting = async (key, value) => {
    setSaving(true)
    await fetch('/api/settings', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ [key]: value })
    })
    setSaving(false)
    reload()
  }

  if (loading) return <div className="skeleton" style={{ height: '300px' }} />

  return (
    <div className="fade-in">
      <div style={{ marginBottom: '24px' }}>
        <h2 style={{ fontSize: '20px', fontWeight: 700 }}>Settings</h2>
        <p style={{ fontSize: '13px', color: 'var(--fg-2)' }}>Server configuration</p>
      </div>

      <div className="card">
        {settings && Object.keys(settings).length > 0 ? (
          <div style={{ display: 'grid', gap: '16px' }}>
            {Object.entries(settings).map(([key, value]) => (
              <div key={key} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 0', borderBottom: '1px solid var(--border-light)' }}>
                <div>
                  <div style={{ fontSize: '13px', fontWeight: 600, color: 'var(--fg-0)' }}>{key}</div>
                  <div style={{ fontSize: '12px', color: 'var(--fg-3)', fontFamily: 'var(--mono)' }}>{value}</div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="empty-state">
            <div className="icon">⚙️</div>
            <p>No settings configured yet.</p>
          </div>
        )}
      </div>

      <div className="card" style={{ marginTop: '20px' }}>
        <div className="card-header">
          <div>
            <div className="card-title">Server Info</div>
            <div className="card-subtitle">Runtime information</div>
          </div>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
          <InfoItem label="Version" value="2.0.0 (Go)" />
          <InfoItem label="Runtime" value="Go" />
          <InfoItem label="Database" value="SQLite (WAL)" />
          <InfoItem label="Port" value={window.location.port || '20180'} />
        </div>
      </div>
    </div>
  )
}

function InfoItem({ label, value }) {
  return (
    <div style={{ padding: '12px 16px', background: 'var(--bg-body)', borderRadius: '8px', border: '1px solid var(--border-light)' }}>
      <div style={{ fontSize: '11px', color: 'var(--fg-3)', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px' }}>{label}</div>
      <div style={{ fontSize: '14px', fontWeight: 600, color: 'var(--fg-0)', marginTop: '4px', fontFamily: 'var(--mono)' }}>{value}</div>
    </div>
  )
}
