import {cloudAnchors} from './cloud-station'
import {useEffect,useState} from 'react'
import {createPortal} from 'react-dom'
import {fleetDependencies} from './fleet'
import {recordError,resolveIssue,type Issue} from './error-triage'
import type {Resident} from './station'
import {angledConnection} from './station-connections'
import './fleet-station.css'
import {IssueConsole} from './issue-console'
import {OfficeProp,officeLayout} from './office-props'
import {floorConnection} from './station-connections'
import {floorPlan,project} from './station-floor'

export function FleetStation({residents,connections,onInspect,selectedWorkload}:{selectedWorkload?:string|null;residents:{pod:Resident;x:number;y:number}[];connections:boolean;onInspect:(title:string,description:string)=>void}){
 const [open,setOpen]=useState(false),[issues,setIssues]=useState<Issue[]>([])
 const [dependency,setDependency]=useState<string|null>(null)
 useEffect(()=>{if(!open)return;const close=(e:KeyboardEvent)=>{if(e.key==='Escape')setOpen(false)};window.addEventListener('keydown',close);return()=>window.removeEventListener('keydown',close)},[open])
 const triage=officeLayout['Error triage']
 const triageFloor=project(36+triage.u/20,triage.v/20)
 const target=document.querySelector('.station-stage')
 const inject=(service:string)=>setIssues(current=>recordError(current,{service,release:current.some(i=>i.service===service&&i.status==='Resolved')?'r2':'r1',type:'TypeError',frame:'checkout.ts:42',at:new Date().toISOString(),traceId:'trace-'+(current.reduce((n,i)=>n+i.count,0)+1)}))
 return <>
  <g className="fleet-annex" aria-label="External dependencies">
   {connections&&<path className="connection-flow connection-outbound" d={angledConnection(cloudAnchors.nat,{x:project(22.5,0).x,y:project(22.5,0).y+230})} stroke="#a5cfbb" strokeWidth="1.5" vectorEffect="non-scaling-stroke" strokeDasharray="6 5" fill="none" pointerEvents="none"/>}
   {fleetDependencies.map((d,i)=>{
    const x=125+i*245,y=-135
    return <g key={d.id} opacity={selectedWorkload&&!residents.some(r=>`${r.pod.workload.namespace}/${r.pod.workload.kind}/${r.pod.workload.name}`===selectedWorkload&&r.pod.workload.name===d.source)?.15:1}>
     {connections&&<path className="connection-flow connection-outbound" d={angledConnection({x,y:y+30},cloudAnchors.igw)} stroke="#a5cfbb" strokeWidth={dependency===d.id?2:1.3} vectorEffect="non-scaling-stroke" strokeDasharray="6 5" fill="none" opacity={dependency&&dependency!==d.id?.45:.85} pointerEvents="none"><title>{d.label} via egress; select to inspect workload route</title></path>}
     {connections&&<g transform="translate(0 230)" pointerEvents="none" opacity={dependency&&dependency!==d.id?.25:.8}>{residents.filter(r=>r.pod.workload.name===d.source).map(r=><path className="connection-flow connection-outbound" key={r.pod.name} d={floorConnection(project(22.5,0),{x:r.x,y:r.y},floorPlan(new Set(residents.map(p=>p.pod.node)).size).tiles)} stroke="#a5cfbb" strokeWidth={dependency===d.id?2:1.2} vectorEffect="non-scaling-stroke" strokeDasharray="6 5" fill="none"/>)}</g>}
     <g transform={`translate(${x} ${y})`} data-object="dependency" className="station-object" tabIndex={0} role="button" aria-label={`Inspect ${d.label}`} onClick={()=>{setDependency(dependency===d.id?null:d.id);onInspect(d.label,d.detail)}} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();setDependency(dependency===d.id?null:d.id);onInspect(d.label,d.detail)}}}>
      <path d="m-60 10 60-30 60 30v10L0 50l-60-30Z" fill="#39596b" stroke="#9cb4bb"/>
      {d.id==='illuma-storage'?<g>
       <path d="M-29-42v46c0 17 58 17 58 0v-46" fill="#829dbb" stroke="#345166"/>
       <ellipse cy="-42" rx="29" ry="12" fill="#b9cddd" stroke="#345166"/>
       <ellipse cy="4" rx="29" ry="12" fill="none" stroke="#526f8a"/>
       <path d="m-13-23 19 3v21l-19-3Z" fill="#e4d7b5"/><path d="m-9-17 11 2m-11 4 11 2" stroke="#8297a3"/>
      </g>:<g><path d="m-30 0 0-55 30-15 32 16v55L0 17Z" fill="#a6c7bd" stroke="#345166"/><path d="M0-40 32-54v55L0 17Z" fill="#5e8991"/>
      <text x="-16" y="-16" style={{fill:"#20394e",fontSize:18}}>{d.id.endsWith("-db")?"DB":d.id==="models"?"AI":d.id==='illuma-storage'?'▱':'@'}</text><circle cx="17" cy="-20" r="4" fill="#e8cb92"/></g>}
      <text className="map-label-source" y="70" textAnchor="middle">{d.label}</text>
     </g>
    </g>
   })}
   <g transform={`translate(${triageFloor.x} ${triageFloor.y+230})`} data-object="triage" role="button" tabIndex={0} className="station-object" aria-label="Open error triage" onClick={()=>setOpen(true)} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();setOpen(true)}}}>
    <OfficeProp name="Error triage"/>
    <text className="map-label-source" y="39" textAnchor="middle" style={{fontSize:12,fill:'#e3e8e9'}}>Error triage</text>
   </g>
  </g>
  {open&&target&&createPortal(<aside className="topology-panel fleet-triage" aria-label="Error triage"><header><h2>Error triage</h2><button aria-label="Close error triage" onClick={()=>setOpen(false)}>×</button></header><p>The example groups below reset on reload. Connected-service issues are stored separately.</p><div className="triage-inject">{['illuma-web','ledgly-api','club-web'].map(service=><button key={service} onClick={()=>inject(service)}>Test {service}</button>)}</div><p role="status">{issues.length} groups · {issues.reduce((n,i)=>n+i.count,0)} events</p>{!issues.length&&<p>Inject an error, repeat it to group occurrences, then resolve and inject again to see a regression.</p>}{issues.map(issue=><article key={issue.key}><h3>{issue.service} <small>{issue.status}</small></h3><p>{issue.type} · {issue.count} occurrences</p><code>{issue.frame}</code><p>{issue.release} · {issue.traceId}</p><small>First: {new Date(issue.firstSeen).toLocaleTimeString()} · Latest: {new Date(issue.lastSeen).toLocaleTimeString()}</small><button disabled={issue.status==='Resolved'} onClick={()=>setIssues(current=>resolveIssue(current,issue.key))}>Resolve issue</button></article>)}<IssueConsole/><details><summary>Still needed for production</summary><p>Per-user operator authorization, a shared production database, reviewed privacy controls, source-map symbolication, real trace links, ownership, notifications, retention, and end-to-end tests alongside Sentry. The connected service is loopback-only, single-process, and disabled in production.</p></details></aside>,target)}
 </>
}
