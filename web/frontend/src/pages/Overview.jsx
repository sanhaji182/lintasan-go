import { useState, useEffect } from 'react'
import { useApi } from '../hooks/useApi'

export default function Overview() {
  const { data: stats, loading } = useApi('/api/stats')
  const { data: logs } = useApi('/api/logs?limit=6')

  if (loading) return <LoadingSkeleton />

  return (
    <div className="fade-in">
      <BaseUrlCard />

      <div className="metric-grid">
        <MetricCard icon="📡" label="Total Requests" value={stats?.total_requests || 0}
          gradient="linear-gradient(135deg, #3c50e0 0%, #6366f1 100%)" bg="var(--primary-light)" />
        <MetricCard icon="⚡" label="Cache Hit Rate" value={stats?.cache_hit_rate || '0%'}
          gradient="linear-gradient(135deg, #10b981 0%, #34d399 100%)" bg="var(--success-light)" />
        <MetricCard icon="🤖" label="Active Models" value={stats?.active_models || 0}
          gradient="linear-gradient(135deg, #f59e0b 0%, #fbbf24 100%)" bg="var(--warning-light)" />
        <MetricCard icon="⏱️" label="Avg Latency" value={Math.round(stats?.avg_latency_ms || 0) + 'ms'}
          gradient="linear-gradient(135deg, #8b5cf6 0%, #a78bfa 100%)" bg="var(--purple-light)" />
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '28px' }}>
        <div className="card">
          <div className="card-header">
            <div>
              <div className="card-title">Connections</div>
              <div className="card-subtitle">{stats?.active_connections || 0} active providers</div>
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '30px' }}>
            <div style={{ textAlign: 'center' }}>
              <div style={{ fontSize: '48px', fontWeight: 700, fontFamily: 'var(--mono)', color: 'var(--primary)' }}>
                {stats?.active_connections || 0}
              </div>
              <div style={{ fontSize: '13px', color: 'var(--fg-2)', marginTop: '4px' }}>Provider Connections</div>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="card-header">
            <div>
              <div className="card-title">Cache Performance</div>
              <div className="card-subtitle">Embedding cache hit ratio</div>
            </div>
          </div>
          <DonutChart value={parseFloat(stats?.cache_hit_rate) || 0} />
        </div>
      </div>

      <div className="card">
        <div className="card-header">
          <div>
            <div className="card-title">Recent Requests</div>
            <div className="card-subtitle">Latest API calls</div>
          </div>
        </div>
        {(!logs || logs.length === 0) ? (
          <div className="empty-state">
            <div className="icon">📭</div>
            <p>No requests yet. Send a request to /v1/chat/completions</p>
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Provider</th>
                <th>Model</th>
                <th>Status</th>
                <th>Latency</th>
                <th>Cache</th>
                <th>Time</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((l, i) => (
                <tr key={i}>
                  <td style={{ fontWeight: 500 }}>{l.provider || '—'}</td>
                  <td><span className="code">{(l.model || '').split('/').pop()}</span></td>
                  <td><span className={`badge ${l.status < 400 ? 'badge-success' : 'badge-error'}`}>{l.status < 400 ? 'Success' : 'Error'}</span></td>
                  <td><span className="code">{l.latency_ms}ms</span></td>
                  <td>{l.cached ? <span className="badge badge-info">Hit</span> : <span style={{ fontSize: '12px', color: 'var(--fg-3)' }}>Miss</span>}</td>
                  <td style={{ fontSize: '12px', color: 'var(--fg-3)' }}>{timeAgo(l.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

function MetricCard({ icon, label, value, gradient, bg }) {
  return (
    <div className="card metric-card">
      <div className="accent" style={{ background: gradient }} />
      <div className="icon-box" style={{ background: bg }}>{icon}</div>
      <div className="value">{typeof value === 'number' ? fmt(value) : value}</div>
      <div className="label">{label}</div>
    </div>
  )
}

function DonutChart({ value }) {
  const circumference = 2 * Math.PI * 15.9
  const offset = circumference - (value / 100) * circumference
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '20px 0' }}>
      <div style={{ position: 'relative', width: '120px', height: '120px', marginBottom: '16px' }}>
        <svg viewBox="0 0 36 36" style={{ width: '100%', height: '100%', transform: 'rotate(-90deg)' }}>
          <circle cx="18" cy="18" r="15.9" fill="none" stroke="var(--border)" strokeWidth="2.5" />
          <circle cx="18" cy="18" r="15.9" fill="none" stroke="var(--success)" strokeWidth="2.5"
            strokeDasharray={circumference} strokeDashoffset={offset} strokeLinecap="round"
            style={{ transition: 'stroke-dashoffset 1s ease' }} />
        </svg>
        <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', flexDirection: 'column' }}>
          <span style={{ fontSize: '22px', fontWeight: 700, fontFamily: 'var(--mono)' }}>{value}%</span>
          <span style={{ fontSize: '10px', color: 'var(--fg-3)' }}>Hit Rate</span>
        </div>
      </div>
    </div>
  )
}

function BaseUrlCard() {
  const [copied, setCopied] = useState(false)
  const baseUrl = `${window.location.origin}/v1`

  return (
    <div style={{ background: 'linear-gradient(135deg, var(--primary-light) 0%, var(--bg-card) 100%)', borderRadius: 'var(--radius)', padding: '20px 24px', boxShadow: 'var(--shadow)', border: '1px solid var(--primary)', marginBottom: '20px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '16px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
        <div style={{ width: '40px', height: '40px', borderRadius: '10px', background: 'var(--primary)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '18px' }}>🔗</div>
        <div>
          <p style={{ fontSize: '12px', fontWeight: 600, color: 'var(--fg-2)', textTransform: 'uppercase', letterSpacing: '0.5px' }}>Base URL</p>
          <code style={{ fontSize: '14px', fontFamily: 'var(--mono)', color: 'var(--fg-0)', fontWeight: 600 }}>{baseUrl}</code>
          <p style={{ fontSize: '11px', color: 'var(--fg-3)', marginTop: '4px' }}>Use this in your AI tools as the OpenAI-compatible endpoint</p>
        </div>
      </div>
      <button onClick={() => { navigator.clipboard.writeText(baseUrl); setCopied(true); setTimeout(() => setCopied(false), 2000) }}
        style={{ padding: '8px 14px', background: copied ? 'var(--success)' : 'var(--primary)', color: '#fff', border: 'none', borderRadius: 'var(--radius-sm)', fontSize: '12px', fontWeight: 500, cursor: 'pointer', whiteSpace: 'nowrap', transition: 'all 0.2s' }}>
        {copied ? 'Copied ✓' : 'Copy URL'}
      </button>
    </div>
  )
}

function LoadingSkeleton() {
  return (
    <div>
      <div className="metric-grid">
        {[1,2,3,4].map(i => <div key={i} className="skeleton" style={{ height: '140px' }} />)}
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '28px' }}>
        <div className="skeleton" style={{ height: '250px' }} />
        <div className="skeleton" style={{ height: '250px' }} />
      </div>
    </div>
  )
}

function fmt(n) { if (n >= 1000000) return (n/1000000).toFixed(1)+'M'; if (n >= 1000) return Math.round(n/1000)+'K'; return String(n) }
function timeAgo(ts) { if (!ts) return '—'; const d = Date.now() - new Date(ts).getTime(); if (d < 60000) return 'just now'; if (d < 3600000) return Math.floor(d/60000)+'m ago'; if (d < 86400000) return Math.floor(d/3600000)+'h ago'; return Math.floor(d/86400000)+'d ago' }
