import { useApi } from '../hooks/useApi'

export default function Logs() {
  const { data: logs, loading, reload } = useApi('/api/logs?limit=50')

  if (loading) return <div className="skeleton" style={{ height: '400px' }} />

  return (
    <div className="fade-in">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: 700 }}>Request Logs</h2>
          <p style={{ fontSize: '13px', color: 'var(--fg-2)' }}>All proxied API requests</p>
        </div>
        <button onClick={reload} style={{ padding: '8px 14px', background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: '8px', fontSize: '12px', cursor: 'pointer', color: 'var(--fg-1)' }}>
          🔄 Refresh
        </button>
      </div>

      <div className="card">
        {(!logs || logs.length === 0) ? (
          <div className="empty-state">
            <div className="icon">📋</div>
            <p>No request logs yet.</p>
          </div>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table className="table">
              <thead>
                <tr>
                  <th>Provider</th>
                  <th>Model</th>
                  <th>Status</th>
                  <th>Latency</th>
                  <th>Tokens In</th>
                  <th>Tokens Out</th>
                  <th>Cache</th>
                  <th>Time</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((l, i) => (
                  <tr key={i}>
                    <td style={{ fontWeight: 500 }}>{l.provider || '—'}</td>
                    <td><span className="code">{(l.model || '').split('/').pop()}</span></td>
                    <td><span className={`badge ${l.status < 400 ? 'badge-success' : 'badge-error'}`}>{l.status}</span></td>
                    <td><span className="code">{l.latency_ms}ms</span></td>
                    <td style={{ fontSize: '12px', color: 'var(--fg-2)' }}>{l.input_tokens || 0}</td>
                    <td style={{ fontSize: '12px', color: 'var(--fg-2)' }}>{l.output_tokens || 0}</td>
                    <td>{l.cached ? <span className="badge badge-info">Hit</span> : <span style={{ fontSize: '12px', color: 'var(--fg-3)' }}>Miss</span>}</td>
                    <td style={{ fontSize: '12px', color: 'var(--fg-3)' }}>{timeAgo(l.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

function timeAgo(ts) { if (!ts) return '—'; const d = Date.now() - new Date(ts).getTime(); if (d < 60000) return 'just now'; if (d < 3600000) return Math.floor(d/60000)+'m ago'; if (d < 86400000) return Math.floor(d/3600000)+'h ago'; return Math.floor(d/86400000)+'d ago' }
