import { useState } from 'react'
import { useApi } from '../hooks/useApi'

const inputStyle={padding:'10px 14px',background:'var(--bg-body)',border:'1px solid var(--border)',borderRadius:'8px',fontSize:'13px',color:'var(--fg-0)',outline:'none',width:'100%'}
const btn={padding:'9px 14px',background:'var(--primary)',color:'#fff',border:'none',borderRadius:'8px',fontSize:'13px',cursor:'pointer'}
function Header({title,sub}){return <div style={{marginBottom:24}}><h2 style={{fontSize:20,fontWeight:700}}>{title}</h2><p style={{fontSize:13,color:'var(--fg-2)'}}>{sub}</p></div>}
function Card({children}){return <div className='card' style={{marginBottom:20}}>{children}</div>}
function Empty({icon,text}){return <div className='empty-state'><div className='icon'>{icon}</div><p>{text}</p></div>}

export default function Playground(){const {data:models}=useApi('/v1/models'); const [model,setModel]=useState(''); const [msg,setMsg]=useState('Hello'); const [out,setOut]=useState(''); async function send(){setOut('Loading...'); const r=await fetch('/v1/chat/completions',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({model:model||models?.data?.[0]?.id,messages:[{role:'user',content:msg}],max_tokens:256})}); const j=await r.json(); setOut(j.error?JSON.stringify(j.error,null,2):(j.choices?.[0]?.message?.content||JSON.stringify(j,null,2)))} return <div className='fade-in'><Header title='Playground' sub='Test chat completions directly from dashboard'/><Card><select style={inputStyle} value={model} onChange={e=>setModel(e.target.value)}>{(models?.data||[]).map(m=><option key={m.id}>{m.id}</option>)}</select><textarea style={{...inputStyle,minHeight:120,marginTop:12}} value={msg} onChange={e=>setMsg(e.target.value)}/><button style={{...btn,marginTop:12}} onClick={send}>Send</button></Card><Card><h3 className='card-title'>Response</h3><pre style={{whiteSpace:'pre-wrap'}}>{out||'No response yet.'}</pre></Card></div>}
