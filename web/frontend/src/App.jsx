import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, NavLink, useLocation } from 'react-router-dom'
import Overview from './pages/Overview'
import Connections from './pages/Connections'
import Models from './pages/Models'
import Logs from './pages/Logs'
import Settings from './pages/Settings'
import Analytics from './pages/Analytics'
import Usage from './pages/Usage'
import Routing from './pages/Routing'
import Playground from './pages/Playground'
import Keys from './pages/Keys'
import Plugins from './pages/Plugins'
import Teams from './pages/Teams'
import Users from './pages/Users'
import Webhooks from './pages/Webhooks'
import Backup from './pages/Backup'
import Docs from './pages/Docs'

const NAV_ITEMS = [
  { path: '/dashboard', icon: '📊', label: 'Overview' },
  { path: '/dashboard/analytics', icon: '📈', label: 'Analytics' },
  { path: '/dashboard/usage', icon: '🪙', label: 'Usage' },
  { path: '/dashboard/routing', icon: '🧭', label: 'Routing' },
  { path: '/dashboard/playground', icon: '💬', label: 'Playground' },
  { path: '/dashboard/connections', icon: '🔌', label: 'Connections' },
  { path: '/dashboard/models', icon: '🤖', label: 'Models' },
  { path: '/dashboard/keys', icon: '🔑', label: 'API Keys' },
  { path: '/dashboard/plugins', icon: '🧩', label: 'Plugins' },
  { path: '/dashboard/teams', icon: '👥', label: 'Teams' },
  { path: '/dashboard/users', icon: '👤', label: 'Users' },
  { path: '/dashboard/webhooks', icon: '🪝', label: 'Webhooks' },
  { path: '/dashboard/backup', icon: '💾', label: 'Backup' },
  { path: '/dashboard/logs', icon: '📋', label: 'Logs' },
  { path: '/dashboard/docs', icon: '📚', label: 'Docs' },
  { path: '/dashboard/settings', icon: '⚙️', label: 'Settings' },
]

function Sidebar() {
  const [theme, setTheme] = useState(() => localStorage.getItem('theme') || 'light')

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('theme', theme)
  }, [theme])

  return (
    <aside className="sidebar">
      <div className="sidebar-logo">
        <span style={{ fontSize: '24px' }}>🚀</span>
        <h1>Lintasan</h1>
        <span>v2</span>
      </div>
      <nav className="sidebar-nav">
        {NAV_ITEMS.map(item => (
          <NavLink
            key={item.path}
            to={item.path}
            end={item.path === '/dashboard'}
            className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
          >
            <span className="icon">{item.icon}</span>
            {item.label}
          </NavLink>
        ))}
      </nav>
      <div className="theme-toggle">
        <button onClick={() => setTheme(t => t === 'dark' ? 'light' : 'dark')}>
          {theme === 'dark' ? '☀️ Light Mode' : '🌙 Dark Mode'}
        </button>
      </div>
    </aside>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <div className="app-layout">
        <Sidebar />
        <main className="main-content">
          <Routes>
            <Route path="/dashboard" element={<Overview />} />
            <Route path="/dashboard/analytics" element={<Analytics />} />
            <Route path="/dashboard/usage" element={<Usage />} />
            <Route path="/dashboard/routing" element={<Routing />} />
            <Route path="/dashboard/playground" element={<Playground />} />
            <Route path="/dashboard/connections" element={<Connections />} />
            <Route path="/dashboard/models" element={<Models />} />
            <Route path="/dashboard/keys" element={<Keys />} />
            <Route path="/dashboard/plugins" element={<Plugins />} />
            <Route path="/dashboard/teams" element={<Teams />} />
            <Route path="/dashboard/users" element={<Users />} />
            <Route path="/dashboard/webhooks" element={<Webhooks />} />
            <Route path="/dashboard/backup" element={<Backup />} />
            <Route path="/dashboard/logs" element={<Logs />} />
            <Route path="/dashboard/docs" element={<Docs />} />
            <Route path="/dashboard/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}
