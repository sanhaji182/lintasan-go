import { useState } from 'react'
import { useApi } from '../hooks/useApi'
const inputStyle={padding:'10px 14px',background:'var(--bg-body)',border:'1px solid var(--border)',borderRadius:'8px',fontSize:'13px',color:'var(--fg-0)',outline:'none',width:'100%'}
const btn={padding:'9px 14px',background:'var(--primary)',color:'#fff',border:'none',borderRadius:'8px',fontSize:'13px',cursor:'pointer'}
function Header({title,sub}){return <div style={{marginBottom:24}}><h2 style={{fontSize:20,fontWeight:700}}>{title}</h2><p style={{fontSize:13,color:'var(--fg-2)'}}>{sub}</p></div>}
function Card({children}){return <div className='card' style={{marginBottom:20}}>{children}</div>}
function Empty({icon,text}){return <div className='empty-state'><div className='icon'>{icon}</div><p>{text}</p></div>}

export default function Teams(){const {data,reload}=useApi('/api/teams'); const [name,setName]=useState(''); async function create(){await fetch('/api/teams',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'create',name})});setName('');reload()} return <div className='fade-in'><Header title='Teams' sub='Team workspaces and members'/><Card><div style={{display:'flex',gap:10}}><input style={inputStyle} placeholder='Team name' value={name} onChange={e=>setName(e.target.value)}/><button style={btn} onClick={create}>Create</button></div></Card><Card>{data?.length?data.map(t=><div key={t.id} style={{padding:12,borderBottom:'1px solid var(--border)'}}><b>{t.name}</b><p style={{fontSize:12,color:'var(--fg-2)'}}>{(t.members||[]).length} members</p></div>):<Empty icon='👥' text='No teams yet.'/>}</Card></div>}
