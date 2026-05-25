import { useState } from 'react'
import { useApi } from '../hooks/useApi'

const inputStyle={padding:'10px 14px',background:'var(--bg-body)',border:'1px solid var(--border)',borderRadius:'8px',fontSize:'13px',color:'var(--fg-0)',outline:'none',width:'100%'}
const btn={padding:'9px 14px',background:'var(--primary)',color:'#fff',border:'none',borderRadius:'8px',fontSize:'13px',cursor:'pointer'}
function Header({title,sub}){return <div style={{marginBottom:24}}><h2 style={{fontSize:20,fontWeight:700}}>{title}</h2><p style={{fontSize:13,color:'var(--fg-2)'}}>{sub}</p></div>}
function Card({children}){return <div className='card' style={{marginBottom:20}}>{children}</div>}
function Empty({icon,text}){return <div className='empty-state'><div className='icon'>{icon}</div><p>{text}</p></div>}

export default function Keys(){const {data,reload}=useApi('/api/keys'); const [name,setName]=useState(''); async function create(){await fetch('/api/keys',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'create',name})});setName('');reload()} return <div className='fade-in'><Header title='API Keys' sub='Create and manage API keys with limits'/><Card><div style={{display:'flex',gap:10}}><input style={inputStyle} placeholder='Key name' value={name} onChange={e=>setName(e.target.value)}/><button style={btn} onClick={create}>Create</button></div></Card><Card><table className='table'><thead><tr><th>Name</th><th>Key</th><th>Created</th></tr></thead><tbody>{(data||[]).map(k=><tr key={k.id}><td>{k.name}</td><td><code className='code'>{k.key}</code></td><td>{k.created_at}</td></tr>)}</tbody></table></Card></div>}
