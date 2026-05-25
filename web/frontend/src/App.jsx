import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, NavLink, useLocation } from 'react-router-dom'
import Overview from './pages/Overview'
import Connections from './pages/Connections'
import Models from './pages/Models'
import Logs from './pages/Logs'
import Settings from './pages/Settings'

const NAV_ITEMS = [
  { path: '/dashboard', icon: '📊', label: 'Overview' },
  { path: '/dashboard/connections', icon: '🔌', label: 'Connections' },
  { path: '/dashboard/models', icon: '🤖', label: 'Models' },
  { path: '/dashboard/logs', icon: '📋', label: 'Logs' },
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
            <Route path="/dashboard/connections" element={<Connections />} />
            <Route path="/dashboard/models" element={<Models />} />
            <Route path="/dashboard/logs" element={<Logs />} />
            <Route path="/dashboard/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}
