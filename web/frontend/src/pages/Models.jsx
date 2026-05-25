import { useApi } from '../hooks/useApi'

export default function Models() {
  const { data, loading } = useApi('/v1/models')
  const models = data?.data || []

  if (loading) return <div className="skeleton" style={{ height: '400px' }} />

  return (
    <div className="fade-in">
      <div style={{ marginBottom: '24px' }}>
        <h2 style={{ fontSize: '20px', fontWeight: 700 }}>Models</h2>
        <p style={{ fontSize: '13px', color: 'var(--fg-2)' }}>{models.length} models available across all connections</p>
      </div>

      {models.length === 0 ? (
        <div className="card">
          <div className="empty-state">
            <div className="icon">🤖</div>
            <p>No models discovered yet. Add a connection and sync models.</p>
          </div>
        </div>
      ) : (
        <div className="card">
          <table className="table">
            <thead>
              <tr>
                <th>Model ID</th>
                <th>Provider</th>
              </tr>
            </thead>
            <tbody>
              {models.map((m, i) => (
                <tr key={i}>
                  <td><span className="code">{m.id}</span></td>
                  <td style={{ fontSize: '13px', color: 'var(--fg-2)' }}>{m.owned_by}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
