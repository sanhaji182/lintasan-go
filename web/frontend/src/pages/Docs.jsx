import { useState } from 'react'
import { useApi } from '../hooks/useApi'
const inputStyle={padding:'10px 14px',background:'var(--bg-body)',border:'1px solid var(--border)',borderRadius:'8px',fontSize:'13px',color:'var(--fg-0)',outline:'none',width:'100%'}
const btn={padding:'9px 14px',background:'var(--primary)',color:'#fff',border:'none',borderRadius:'8px',fontSize:'13px',cursor:'pointer'}
function Header({title,sub}){return <div style={{marginBottom:24}}><h2 style={{fontSize:20,fontWeight:700}}>{title}</h2><p style={{fontSize:13,color:'var(--fg-2)'}}>{sub}</p></div>}
function Card({children}){return <div className='card' style={{marginBottom:20}}>{children}</div>}
function Empty({icon,text}){return <div className='empty-state'><div className='icon'>{icon}</div><p>{text}</p></div>}

export default function Docs(){return <div className='fade-in'><Header title='Docs' sub='Quick start and endpoint reference'/><Card><h3>OpenAI-compatible endpoint</h3><pre className='code' style={{display:'block',whiteSpace:'pre-wrap',marginTop:12}}>Base URL: {window.location.origin}/v1\nPOST /v1/chat/completions\nGET /v1/models\nPOST /v1/embeddings</pre></Card><Card><h3>CLI</h3><pre className='code' style={{display:'block',whiteSpace:'pre-wrap',marginTop:12}}>lintasan start\nlintasan setup\nlintasan mitm start</pre></Card></div>}
