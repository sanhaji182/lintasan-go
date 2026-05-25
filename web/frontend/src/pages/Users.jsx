import { useState } from 'react'
import { useApi } from '../hooks/useApi'
const inputStyle={padding:'10px 14px',background:'var(--bg-body)',border:'1px solid var(--border)',borderRadius:'8px',fontSize:'13px',color:'var(--fg-0)',outline:'none',width:'100%'}
const btn={padding:'9px 14px',background:'var(--primary)',color:'#fff',border:'none',borderRadius:'8px',fontSize:'13px',cursor:'pointer'}
function Header({title,sub}){return <div style={{marginBottom:24}}><h2 style={{fontSize:20,fontWeight:700}}>{title}</h2><p style={{fontSize:13,color:'var(--fg-2)'}}>{sub}</p></div>}
function Card({children}){return <div className='card' style={{marginBottom:20}}>{children}</div>}
function Empty({icon,text}){return <div className='empty-state'><div className='icon'>{icon}</div><p>{text}</p></div>}

export default function Users(){const {data,reload}=useApi('/api/users'); const [username,setUsername]=useState(''); async function create(){await fetch('/api/users',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'create',username,role:'viewer'})});setUsername('');reload()} return <div className='fade-in'><Header title='Users' sub='Local users and roles'/><Card><div style={{display:'flex',gap:10}}><input style={inputStyle} placeholder='Username' value={username} onChange={e=>setUsername(e.target.value)}/><button style={btn} onClick={create}>Create</button></div></Card><Card><table className='table'><thead><tr><th>User</th><th>Role</th><th>Status</th></tr></thead><tbody>{(data||[]).map(u=><tr><td>{u.username}</td><td><span className='badge badge-info'>{u.role}</span></td><td>{u.active?'Active':'Inactive'}</td></tr>)}</tbody></table></Card></div>}
