import type {Resident} from './station'
import type {TopologyData} from './topology-data'
import {namespaceColor} from './station-topology'
import {usagePercent} from './node-metrics'
type Room={node:string;x:number;y:number;pods:Resident[]}
export function NodeInsights({rooms,topology,onInspect}:{rooms:Room[];topology:TopologyData|null;onInspect:(title:string,description:string)=>void}){
 return <g>{rooms.map(room=>{
  const fact=topology?.nodes?.find(n=>n.name===room.node)
  const volumes=(topology?.volumes||[]).filter(v=>room.pods.some(p=>p.name===v.pod&&p.workload.namespace===v.namespace))
  const inspect=()=>onInspect(room.node,`CPU: ${fact?.cpuMilli==null?'unavailable':fact.cpuMilli+' millicores'} / ${fact?.cpuCapacity||'unknown'} allocatable. Memory: ${fact?.memoryBytes==null?'unavailable':Math.round(fact.memoryBytes/1048576)+' MiB'} / ${fact?.memoryCapacity?Math.round(fact.memoryCapacity/1048576)+' MiB':'unknown'}. Pressure: ${fact?fact.pressure.join(', ')||'none reported':'unknown'}. AZ: ${fact?.zone||'not reported'}. Usage requires a fresh Metrics Server sample.`)
  return <g key={room.node}>
   {fact?.zone&&<g pointerEvents="none"><path d={`M${room.x} ${room.y+210}l380 190-380 190-380-190Z`} fill="none" stroke={namespaceColor(fact.zone)} strokeWidth="2" strokeDasharray="7 5"/></g>}
   <g transform={`translate(${room.x+260} ${room.y+380})`} data-object="node-gauge" role="button" tabIndex={0} aria-label={`Inspect resources for ${room.node}`} onClick={inspect} onFocus={inspect} onMouseEnter={inspect} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();inspect()}}}>
    <path d="M-36-40 0-58 36-40v58L0 36-36 18Z" fill="#29495d" stroke="#91b6c5"/>
    <path d="M0-22 36-40M0-22v58M-36-40 0-22" fill="none" stroke="#91b6c5"/>
    {[usagePercent(fact?.cpuMilli,fact?.cpuCapacity),usagePercent(fact?.memoryBytes,fact?.memoryCapacity)].map((value,i)=><g key={i} transform={`translate(-28 ${-20+i*20}) skewY(26.565)`}><rect width="22" height="9" rx="2" fill="#102c40"/>{value!=null?<rect width={22*value/100} height="9" rx="2" fill={value>85?'#edb785':'#a6d9c1'}/>:<text x="11" y="8" fontSize="10" textAnchor="middle" fill="#bdccdc">?</text>}</g>)}
    <circle cx="18" cy="-13" r="4" fill={!fact?'#8194a4':fact.pressure.length?'#ec9d82':'#a6d9c1'}/>
    <title>CPU / memory / pressure</title>
   </g>
   {volumes.length>0&&<g transform={`translate(${room.x-235} ${room.y+395})`} data-object="storage" tabIndex={0} role="button" aria-label={`Inspect mounted storage for ${room.node}`} onClick={()=>onInspect('Mounted storage',volumes.map(v=>`${v.namespace}/${v.pod}: ${v.kind==='PVC'?v.claim:v.name} · ${v.kind} · ${v.phase} ${v.capacity} ${v.storageClass}`).join('\n'))} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();e.currentTarget.dispatchEvent(new MouseEvent('click',{bubbles:true}))}}}>
    <path d="M-30-20 0-35 30-20v45L0 40-30 25Z" fill="#718aa0" stroke="#bed0d8"/><path d="M-30-20 0-5 30-20M0-5v45" fill="none" stroke="#bed0d8"/>
    {[0,1,2].map(i=><path key={i} d={`M-24 ${-6+i*12}l17 8`} stroke={volumes.some(v=>v.kind==='PVC')?'#b6abdf':'#d7b88c'} strokeWidth="4"/>)}<title>{volumes.length} mounted PVC / emptyDir volumes; inspect for lifecycle</title>
   </g>}
  </g>
 })}</g>
}

export function PendingBay({pods,topology,x,y,onInspect}:{pods:Resident[];topology:TopologyData|null;x:number;y:number;onInspect:(title:string,description:string)=>void}){
 if(!pods.length)return null
 return <g transform={`translate(${x} ${y})`} aria-label="Unscheduled pod waiting area">
  <path d="M0 0 240 120 0 240-240 120Z" fill="#3d4f65" stroke="#d7b88c" strokeWidth="2" strokeDasharray="7 5"/>
  <text y="5" textAnchor="middle" fill="#e7c58e" fontSize="22">Awaiting placement · {pods.length}</text>
  {pods.slice(0,9).map((p,i)=>{const f=topology?.pending?.find(v=>v.namespace===p.workload.namespace&&v.pod===p.name);const inspect=()=>onInspect(p.name,`${f?.reason||'Not assigned to a node'}. ${f?.message||'Scheduling details unavailable.'}`);return <g key={p.workload.namespace+'/'+p.name} transform={`translate(${(i%3-Math.floor(i/3))*58} ${65+(i%3+Math.floor(i/3))*28})`} data-object="pending" role="button" tabIndex={0} aria-label={`Pending pod ${p.name}`} onClick={inspect} onFocus={inspect} onMouseEnter={inspect} onKeyDown={e=>{if(e.key==='Enter'){inspect()}}}><rect x="-20" y="-20" width="40" height="30" rx="10" fill={namespaceColor(p.workload.namespace)}/><rect x="-15" y="-15" width="30" height="18" rx="7" fill="#153046"/><circle cx="-6" cy="-7" r="2" fill="#c5dfdd"/><circle cx="6" cy="-7" r="2" fill="#c5dfdd"/><path d="M-10 12v7m20-7v7" stroke="#d7b88c" strokeWidth="5"/><title>{f?.reason||'Unscheduled'}</title></g>})}
  {pods.length>9&&<text y="228" textAnchor="middle" fill="#e7c58e">+{pods.length-9} more · use workload inventory</text>}
 </g>
}
