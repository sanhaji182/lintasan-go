import { useState } from 'react'
import { useApi } from '../hooks/useApi'

const inputStyle={padding:'10px 14px',background:'var(--bg-body)',border:'1px solid var(--border)',borderRadius:'8px',fontSize:'13px',color:'var(--fg-0)',outline:'none',width:'100%'}
const btn={padding:'9px 14px',background:'var(--primary)',color:'#fff',border:'none',borderRadius:'8px',fontSize:'13px',cursor:'pointer'}
function Header({title,sub}){return <div style={{marginBottom:24}}><h2 style={{fontSize:20,fontWeight:700}}>{title}</h2><p style={{fontSize:13,color:'var(--fg-2)'}}>{sub}</p></div>}
function Card({children}){return <div className='card' style={{marginBottom:20}}>{children}</div>}
function Empty({icon,text}){return <div className='empty-state'><div className='icon'>{icon}</div><p>{text}</p></div>}

export default function Usage(){const {data}=useApi('/api/usage'); return <div className='fade-in'><Header title='Usage' sub='Provider, model, and daily token usage'/><div style={{display:'grid',gridTemplateColumns:'1fr 1fr',gap:20}}><BarCard title='By Provider' items={data?.providers||[]} label='provider'/><BarCard title='By Model' items={data?.models||[]} label='model'/></div><Card><h3 className='card-title'>Daily Usage</h3><table className='table'><thead><tr><th>Date</th><th>Requests</th><th>Tokens</th></tr></thead><tbody>{(data?.daily||[]).map((d,i)=><tr key={i}><td>{d.date}</td><td>{d.requests}</td><td>{d.tokens}</td></tr>)}</tbody></table></Card></div>}
function BarCard({title,items,label}){let max=Math.max(1,...items.map(x=>x.tokens||0));return <Card><h3 className='card-title'>{title}</h3>{items.length?items.map((x,i)=><div key={i} style={{margin:'14px 0'}}><div style={{display:'flex',justifyContent:'space-between',fontSize:12}}><span>{x[label]||'—'}</span><b>{x.tokens}</b></div><div style={{height:8,background:'var(--bg-body)',borderRadius:999,overflow:'hidden'}}><div style={{width:((x.tokens||0)/max*100)+'%',height:'100%',background:'var(--primary)'}}/></div></div>):<Empty icon='🪙' text='No usage yet.'/>}</Card>}
