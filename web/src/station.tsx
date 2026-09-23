import {useEffect,useRef,useState,type CSSProperties,type KeyboardEvent} from 'react'
import './station.css'
import {useTopology} from './topology-data'
import {BoundaryControls,BoundaryObjects,type Layer} from './station-boundaries'
import {StationFloor,nodeOrigin,project,floorPlan} from './station-floor'
import {angledConnection,floorConnection} from './station-connections'
import {DeliveryStation} from './delivery-station'
import {RobotWork,workRole,workDescription,type WorkRole} from './robot-work'
import {OfficeProp,officeLayout} from './office-props'
import {TopologyPanel,equipment,namespaceColor,type Facility} from './station-topology'
import {useFacilityHealth} from './facility-health'
export type StationWorkload={kind:string;namespace:string;name:string;ready:number;desired:number;status:string;createdAt:string}
export type Resident={name:string;node:string;phase:string;ready:number;containers:number;restarts:number;workload:StationWorkload;services:string[]}
export type StationEvent={type:string;reason:string;message:string;lastSeen:string;workload:StationWorkload}
type Info={title:string;description:string;workload?:StationWorkload}
const labels={ready:'Ready',repair:'Needs attention',waiting:'Awaiting placement',complete:'Completed'}
function mood(p:Resident):keyof typeof labels{return p.phase==='Succeeded'?'complete':p.phase==='Failed'?'repair':!p.node?'waiting':p.phase==='Running'&&p.containers>0&&p.ready===p.containers?'ready':'repair'}
function Robot({state,accent,daemon,role,simulated,identity}:{state:keyof typeof labels;accent:string;daemon:boolean;role:WorkRole;simulated:boolean;identity:string}){
  let seed=0;for(const character of identity)seed=(seed*31+character.charCodeAt(0))>>>0
  const phrases=['Tea break?','Stretch time.','Nice office.','Beep, hello!','Need a snack.','Back in a bit.']
  return <g className={`station-robot ${state} ${simulated&&state==='ready'?'robot-personality':''}`} style={{'--routine-duration':`${38+seed%19}s`,'--routine-delay':`-${seed%53}s`,'--blink-duration':`${4.3+(seed%37)/10}s`,'--blink-delay':`-${seed%71/10}s`} as CSSProperties}>
  <g className="station-walker"><ellipse cy="3" rx="19" ry="7" fill="#0b182a" opacity=".4"/>
  <path d="M-8-10v10M8-10v10" stroke="#7e9bae" strokeWidth="7" strokeLinecap="round"/><path d="m-15-24-5 10m35-10 5 10" stroke="#eadcba" strokeWidth="6" strokeLinecap="round"/>
  <rect x="-13" y="-29" width="26" height="22" rx="7" fill="#d6c7a6"/><rect x="-6" y="-24" width="12" height="10" rx="3" fill={accent}/><circle cy="-19" r="2.5" className="station-light"/>
  <path d="M0-46v-7" stroke="#afc4d2" strokeWidth="3"/><circle cy="-55" r="3" className="station-light"/><rect x="-20" y="-47" width="40" height="23" rx="9" fill={accent}/><rect x="-15" y="-43" width="30" height="14" rx="6" fill="#1d334b"/>
  {daemon&&<g className="station-daemon-badge" aria-hidden="true"><circle cx="17" cy="-20" r="8" fill="#243c51" stroke={accent} strokeWidth="1.5"/><path d="m14-17 5-6m-2-1 3 1-1 3" fill="none" stroke="#f0e2c4" strokeWidth="2" strokeLinecap="round"/></g>}
  <g className="station-eyes" stroke="#b9f1e1" strokeWidth="2.5" strokeLinecap="round" fill="none">{state==='repair'?<path d="m-9-39 4 5m0-5-4 5m14-5 4 5m0-5-4 5"/>:<path d="M-7-39v3m14-3v3m-9 3q2 2 4 0"/>}</g>
  {simulated&&state==='ready'&&<g className="robot-chatter" aria-hidden="true" pointerEvents="none"><path d="M-38-84h76v19H5l-5 6-3-6h-35Z" fill="#f0e2c4" stroke="#49677b"/><text y="-71" textAnchor="middle">{phrases[seed%phrases.length]}</text></g>}
  </g>{state!=='waiting'&&<RobotWork role={role} active={simulated&&state==='ready'}/>}</g>}

export function Station({residents,events,simulated,loading,errors,truncated,onOpen}:{residents:Resident[];events:StationEvent[];simulated:boolean;loading:boolean;errors:number;truncated:boolean;onOpen:(w:StationWorkload)=>void}){
  const [facility,setFacility]=useState<Facility|null>(null),[namespace,setNamespace]=useState<string|null>(null),[highlight,setHighlight]=useState<string|null>(null)
  const [previewNamespace,setPreviewNamespace]=useState<string|null>(null)
  const [daemonFocus,setDaemonFocus]=useState<string|null>(null)
  const workloadKey=(w:StationWorkload)=>`${w.namespace}/${w.kind}/${w.name}`
  const activeNamespace=previewNamespace??namespace
  const [boundary,setBoundary]=useState<Layer|null>(null)
  const topology=useTopology(simulated,residents)
  const [routeService,setRouteService]=useState<string|null>(null)
  const [identityPod,setIdentityPod]=useState<string|null>(null)
  const facilityHealth=useFacilityHealth(simulated)
  const [paused,setPaused]=useState(false),[network,setNetwork]=useState(true),[showEvents,setShowEvents]=useState(false)
  const [info,setInfo]=useState<Info|null>(null),[zoom,setZoom]=useState(1),[pan,setPan]=useState({x:0,y:0}),[pages,setPages]=useState<Record<string,number>>({})
  const drag=useRef<{x:number;y:number;px:number;py:number}|null>(null)
  useEffect(()=>{setInfo(null);setDaemonFocus(null)},[residents])
  const groups=new Map<string,Resident[]>(),seen=new Set<string>()
  for(const p of residents){const id=`${p.workload.namespace}/${p.name}`;if(seen.has(id))continue;seen.add(id);groups.set(p.node||'',[...(groups.get(p.node||'')||[]),p])}
  const rooms=[...groups].sort(([a],[b])=>a.localeCompare(b)).map(([node,pods],index)=>({node,pods,...project(nodeOrigin(index).u,nodeOrigin(index).v),index,size:1.2}))
  const height=Math.max(1200,Math.ceil(rooms.length/2)*480+720)
  const connectionTiles=floorPlan(rooms.length).tiles
  const visibleResidents=rooms.flatMap(room=>room.pods.slice((pages[room.node]||0)*9,(pages[room.node]||0)*9+9).map((pod,i)=>({pod,x:room.x+((i%3-Math.floor(i/3))*64)*room.size,y:room.y+(58+(i%3+Math.floor(i/3))*32)*room.size})))
  const visibleRoutes=new Set((topology?.routes||[]).filter(r=>!routeService||`${r.namespace}/${r.service}`===routeService).map(r=>`${r.namespace}/${r.service}`))
  const world=useRef<SVGSVGElement>(null)
  const camera=useRef({zoom,pan});camera.current={zoom,pan}
  useEffect(()=>{
    const svg=world.current;if(!svg)return
    const wheel=(event:WheelEvent)=>{
      event.preventDefault()
      const matrix=svg.getScreenCTM();if(!matrix)return
      const point=new DOMPoint(event.clientX,event.clientY).matrixTransform(matrix.inverse())
      const current=camera.current
      const delta=event.deltaY*(event.deltaMode===1?16:event.deltaMode===2?svg.clientHeight:1)
      const next=Math.max(.45,Math.min(3,current.zoom*Math.exp(-Math.max(-100,Math.min(100,delta))*.0025)))
      const ratio=next/current.zoom
      const offset={x:point.x-800-(point.x-800-current.pan.x)*ratio,y:point.y-height/2-(point.y-height/2-current.pan.y)*ratio}
      camera.current={zoom:next,pan:offset};setZoom(next);setPan(offset)
    }
    svg.addEventListener('wheel',wheel,{passive:false})
    return()=>svg.removeEventListener('wheel',wheel)
  },[height])
  const services=[...new Set(residents.flatMap(p=>p.services))].sort()
  const eventKeys=new Set<string>();const recent=[...events].sort((a,b)=>Date.parse(b.lastSeen)-Date.parse(a.lastSeen)).filter(e=>{const id=`${e.workload.namespace}/${e.workload.name}/${e.reason}/${e.message}/${e.lastSeen}`;if(eventKeys.has(id))return false;eventKeys.add(id);return true}).slice(0,8)
  const activate=(e:KeyboardEvent,action:()=>void)=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();action()}}
  const describe=(p:Resident)=>{
    const daemon=p.workload.kind==='DaemonSet'
    setDaemonFocus(daemon?workloadKey(p.workload):null)
    const nodes=new Set(residents.filter(r=>workloadKey(r.workload)===workloadKey(p.workload)&&r.node).map(r=>r.node)).size
    setInfo({title:p.name,description:`${p.workload.namespace} · ${labels[mood(p)]} · ${p.ready}/${p.containers} containers ready · ${p.restarts} restarts · ${simulated?'Illustrated demo activity':'Role illustration only; live activity is not measured'}: ${workDescription[workRole(p.workload.name)]}${daemon?` · DaemonSet: ${p.workload.ready}/${p.workload.desired} ready (reported); pods loaded on ${nodes} nodes. Per-node eligibility is not loaded.`:''}`,workload:p.workload})
  }
  return <section className={`station ${paused?'station-paused':''}`} aria-label="Cluster habitat">
    <details className="map-options"><summary>Map options</summary><header className="station-toolbar"><div><h1>Cluster station</h1></div><div className="station-tools"><button title="Optional relationships; hallways are a navigation metaphor" aria-label="Show Service connections" aria-pressed={network} onClick={()=>setNetwork(!network)}>⌁ <span>Connections</span></button><button title="Recent events" aria-label="Recent events" aria-pressed={showEvents} onClick={()=>setShowEvents(!showEvents)}>◷ <b>{recent.length}</b></button><button title={paused?'Resume motion':'Pause motion'} aria-label={paused?'Resume motion':'Pause motion'} aria-pressed={paused} onClick={()=>setPaused(!paused)}>{paused?'▶':'Ⅱ'}</button></div></header>
    </details><div className="namespace-map-key station-namespace-legend" aria-label="Namespace highlighting"><button aria-pressed={!namespace&&!highlight} onClick={()=>{setNamespace(null);setHighlight(null)}}>All</button>{[...new Set(residents.map(p=>p.workload.namespace))].sort().map(ns=><button key={ns} aria-pressed={namespace===ns} onMouseEnter={()=>setPreviewNamespace(ns)} onMouseLeave={()=>setPreviewNamespace(null)} onFocus={()=>setPreviewNamespace(ns)} onBlur={()=>setPreviewNamespace(null)} onClick={()=>{setNamespace(namespace===ns?null:ns);setHighlight(null)}}><i style={{background:namespaceColor(ns)}}/>{ns}</button>)}{highlight&&<button onClick={()=>setHighlight(null)}>Clear {highlight} ×</button>}</div>
    {(loading||errors>0||truncated)&&<div className="station-notice" role="status">{loading?'Refreshing; previous observation may be shown. ':''}{errors>0?`${errors} workloads unavailable; partial scene. `:''}{truncated?'First 24 workloads shown; filter to narrow.':''}</div>}
    <div className="station-stage"><svg ref={world} tabIndex={0} onKeyDown={e=>{if(e.target!==e.currentTarget)return;if(e.key==='0'){setZoom(1);setPan({x:0,y:0})}else if(e.key==='+'||e.key==='='){e.preventDefault();setZoom(z=>Math.min(3,z*1.15))}else if(e.key==='-'){e.preventDefault();setZoom(z=>Math.max(.45,z/1.15))}}} className="station-world" viewBox={`0 0 1600 ${height}`} aria-label="Interactive node rooms; scroll to zoom, drag to pan. Keyboard: plus, minus, zero to reset" onPointerDown={e=>{if((e.target as Element).closest('[data-object],button,a,input,select,textarea,summary,.delivery-inspector'))return;drag.current={x:e.clientX,y:e.clientY,px:pan.x,py:pan.y};e.currentTarget.setPointerCapture(e.pointerId)}} onPointerMove={e=>{if(!drag.current)return;const scale=Math.max(1600/e.currentTarget.clientWidth,height/e.currentTarget.clientHeight);setPan({x:drag.current.px+(e.clientX-drag.current.x)*scale,y:drag.current.py+(e.clientY-drag.current.y)*scale})}} onPointerUp={()=>{drag.current=null}} onPointerCancel={()=>{drag.current=null}}>
      <defs><pattern id="station-stars" width="110" height="95" patternUnits="userSpaceOnUse"><circle cx="18" cy="24" r="1" fill="#617b96"/><circle cx="83" cy="70" r=".7" fill="#617b96"/></pattern><pattern id="station-floor" width="32" height="32" patternUnits="userSpaceOnUse" patternTransform="matrix(1 .5 -1 .5 0 0)"><rect width="32" height="32" fill="#48657a"/><path d="M32 0H0V32" fill="none" stroke="#617d8d"/></pattern></defs>
      <rect width="1600" height={height} fill="url(#station-stars)"/><g transform={`translate(${pan.x} ${pan.y}) translate(800 ${height/2}) scale(${zoom}) translate(-800 ${-height/2})`}>
      <g transform="translate(0 230)">
      <StationFloor count={rooms.length}/>
      {network&&topology&&visibleRoutes.size>0&&<g className="discovered-route" pointerEvents="none">
        <path d={angledConnection({x:448,y:-98},project(19.5,0))+floorConnection(project(19.5,0),{x:1033,y:299},connectionTiles)} fill="none" stroke="#8fd2dd" strokeWidth="3" strokeLinejoin="miter"/>
        {rooms.flatMap(room=>room.pods.slice((pages[room.node]||0)*9,(pages[room.node]||0)*9+9).map((pod,i)=>({pod,i,room}))).map(({pod,i,room})=>{
          const endpoints=topology.endpoints.filter(e=>visibleRoutes.has(`${e.namespace}/${e.service}`)&&e.namespace===pod.workload.namespace&&e.pod===pod.name)
          if(!endpoints.length)return null
          const ready=endpoints.every(e=>e.ready===true)?true:endpoints.every(e=>e.ready===false)?false:null
          const color=ready===true?'#a6e5c7':ready===false?'#edb785':'#93a4b7'
          const x=room.x+((i%3-Math.floor(i/3))*64)*room.size,y=room.y+(58+(i%3+Math.floor(i/3))*32)*room.size
          return <g key={pod.workload.namespace+'/'+pod.name} data-endpoint-readiness={ready===null?'unknown':String(ready)}><title>{pod.name}: endpoint readiness {ready===null?'unknown':String(ready)}; membership, not measured traffic</title><path d={floorConnection({x:1033,y:299},{x,y},connectionTiles)} fill="none" stroke={color} strokeWidth="3" strokeLinejoin="miter" strokeDasharray={ready===true?undefined:'5 7'}/><ellipse cx={x} cy={y} rx="31" ry="15" fill="none" stroke={color} strokeWidth="3"/></g>
        })}
      </g>}
      {network&&topology&&<g className="boundary-relationships">
        {visibleResidents.map(({pod,x,y})=>{
          const identity=topology.identities.find(p=>p.namespace===pod.workload.namespace&&p.pod===pod.name)
          if(!identity?.policies.length)return null
          const label=`Policies selecting ${pod.name}: ${identity.policies.join(', ')}. Selection only; enforcement unknown.`
          const d=angledConnection({x:800,y:-105},project(19.5,0))+floorConnection(project(19.5,0),{x,y},connectionTiles)
          return <g key={pod.workload.namespace+'/'+pod.name} data-object="policy-relationship" role="button" tabIndex={0} aria-label={label} className="station-object" onClick={()=>setBoundary('policy')} onKeyDown={e=>activate(e,()=>setBoundary('policy'))} onMouseEnter={()=>setInfo({title:'Policy selection',description:label})} onFocus={()=>setInfo({title:'Policy selection',description:label})}><title>{label}</title><path d={d} fill="none" stroke="transparent" strokeWidth="16"/><path d={d} fill="none" stroke="#efb49d" strokeWidth="3" strokeDasharray="3 6" strokeLinejoin="miter"/></g>
        })}
        {Object.keys(topology.externalNames).length>0&&<g data-object="egress-relationship" role="button" tabIndex={0} className="station-object" aria-label="ExternalName Service destinations; callers and outbound traffic unknown" onClick={()=>setBoundary('egress')} onKeyDown={e=>activate(e,()=>setBoundary('egress'))} onMouseEnter={()=>setInfo({title:'ExternalName destinations',description:Object.entries(topology.externalNames).map(([s,h])=>`${s} → ${h}`).join('; ')+'. DNS configuration only; callers and traffic unknown.'})}>
          <title>Service desk → Egress: ExternalName DNS configuration, not observed outbound calls</title>
          <path d={floorConnection({x:1033,y:299},project(22.5,0),connectionTiles)+angledConnection(project(22.5,0),{x:1200,y:-105})} fill="none" stroke="transparent" strokeWidth="16"/>
          <path d={floorConnection({x:1033,y:299},project(22.5,0),connectionTiles)+angledConnection(project(22.5,0),{x:1200,y:-105})} fill="none" stroke="#9bd4b1" strokeWidth="3" strokeDasharray="10 5 2 5" strokeLinejoin="miter"/>
        </g>}
      </g>}
      {network&&<g>{rooms.flatMap(room=>[
        {kind:'control',x:720,y:290,color:'#d0a6ff',dash:'4 9',show:!!room.node,description:'Purple dotted · Conceptual API / node management relationship; not an observed connection.'},
        {kind:'services',x:1020,y:440,color:'#6de1f2',dash:'12 7',show:room.pods.some(p=>p.services.length),description:'Cyan dashed · Loaded Service selector matches include pods on this node. Not verified endpoint membership or traffic.'},
        {kind:'telemetry',x:1320,y:590,color:'#f1ce7b',dash:undefined,show:room.pods.some(p=>p.workload.namespace==='observability'),description:'Gold solid · Observed placement: this node hosts loaded observability pods. This is not a telemetry flow measurement.'}
      ].filter(link=>link.show).map(link=><g key={room.node+link.kind} data-object="relationship" role="button" tabIndex={0} aria-label={link.kind+' relationship to '+room.node} className="station-object" onMouseEnter={()=>setInfo({title:link.kind+' → '+room.node,description:link.description})} onFocus={()=>setInfo({title:link.kind+' → '+room.node,description:link.description})} onClick={()=>setInfo({title:link.kind+' → '+room.node,description:link.description})} onKeyDown={e=>activate(e,()=>setInfo({title:link.kind+' → '+room.node,description:link.description}))}>
        <path d={floorConnection(project(({control:0,services:15,telemetry:30}[link.kind as Facility])+4.5,7.5),project(nodeOrigin(room.index).u+4.5,nodeOrigin(room.index).v+2.5),connectionTiles)} fill="none" stroke="transparent" strokeWidth="16"/>
        <path d={floorConnection(project(({control:0,services:15,telemetry:30}[link.kind as Facility])+4.5,7.5),project(nodeOrigin(room.index).u+4.5,nodeOrigin(room.index).v+2.5),connectionTiles)} fill="none" stroke={link.color} strokeWidth="3" strokeLinejoin="miter" strokeDasharray={link.dash} opacity=".9"/>
      </g>))}</g>}
      {([{key:'control',name:'Control plane',x:720,y:110,target:'/summary'},{key:'services',name:'Service routing',x:1020,y:260,target:'/network'},{key:'telemetry',name:'Telemetry',x:1320,y:410,target:'/observability'}] as const).map(facility=>{
        const signal=facilityHealth[facility.key],color=signal.state==='ready'?'#a6e5c7':signal.state==='warning'?'#edb785':'#93a4b7'
        const details={title:facility.name,description:`${simulated?'Demo · ':''}${signal.state}. ${signal.detail}${facilityHealth.observedAt?' Observed '+new Date(facilityHealth.observedAt).toLocaleTimeString():''}`}
        return <g key={facility.key} data-object={facility.key} className="station-object" role="link" tabIndex={0} aria-label={`${facility.name}: ${signal.state}. Open inspection`} transform={`translate(${facility.x} ${facility.y})`} onMouseEnter={()=>setInfo(details)} onFocus={()=>setInfo(details)} onClick={()=>setFacility(facility.key)} onKeyDown={e=>activate(e,()=>setFacility(facility.key))}>
          {(facility.key==='services'?['Service']:equipment[facility.key]).slice().sort((a,b)=>(officeLayout[a].u+officeLayout[a].v)-(officeLayout[b].u+officeLayout[b].v)).map(name=><g key={name} transform={`translate(${officeLayout[name].u-officeLayout[name].v} ${(officeLayout[name].u+officeLayout[name].v)/2})`}>
            <OfficeProp name={name}/>
            <text y="25" textAnchor="middle" style={{fontSize:8,fill:'#e3e8e9'}}>{name}</text>
          </g>)}
          {facility.key==='control'&&<text y="-65" textAnchor="middle">AWS-managed · conceptual suite</text>}
          {facility.key==='services'&&<text y="105" textAnchor="middle">Service → EndpointSlice → pod</text>}
          <text y="-88" textAnchor="middle" className="station-room-label">{facility.name}</text>
          <text y="218" textAnchor="middle" style={{fill:color}}>{simulated?'Demo · ':''}{signal.state==='ready'?'Ready':signal.state==='warning'?'Attention':'Unknown'}</text>
        </g>
      })}
      {rooms.map(room=>{const attention=room.pods.filter(p=>mood(p)==='repair').length;const page=Math.min(pages[room.node]||0,Math.max(0,Math.ceil(room.pods.length/9)-1));const start=page*9;const visible=room.pods.slice(start,start+9);const roomInfo={title:room.node||'Waiting bay',description:`${room.pods.length} observed pods · ${attention} need attention. Equal-size node rooms: allocatable capacity is not loaded. Light summarizes pods, not node health.`};return <g key={room.node} transform={`translate(${room.x} ${room.y}) scale(${room.size})`} className={`station-room ${attention?'station-alert':''}`}>
        <g transform="scale(1.45)" data-object="room" className="station-object" role="button" tabIndex={0} aria-label={`${roomInfo.title}: ${roomInfo.description}`} onMouseEnter={()=>setInfo(roomInfo)} onFocus={()=>setInfo(roomInfo)} onClick={()=>setInfo(roomInfo)} onKeyDown={e=>activate(e,()=>setInfo(roomInfo))}>
          <path d="M-155 75 0-3l155 78L0 153Z" fill="transparent"/>
          <text y="-93" textAnchor="middle" className="station-room-label">{room.node?`Node ${room.index+1}`:'Waiting bay'}{attention?'  ⚠':''}</text><text y="200" textAnchor="middle" className="station-count">{room.pods.length} pods{attention?` / ${attention} need attention`:''}</text>
          <path d="m-119 48 23-12 23 12v29l-23 12-23-12Z" fill="#253e54" stroke="#9cb2bc"/><path d="m-110 54 12 6m-12 2 12 6m-12 2 12 6" stroke="#b3dcd6" strokeWidth="2"/>
        </g>
        <g aria-hidden="true" pointerEvents="none">
          <path d="m-20-15-84 42v30l84-42Z" fill="#19374d" stroke="#89a8b6" strokeWidth="2"/>
          <path d="m-27-5-68 34m34-17v18" fill="none" stroke="#8ebbc9" strokeWidth="2"/>
          <path d="m132 31 67 34v25l-67-34Z" fill="#20384d" stroke="#789eaf"/>
          <path d="m140 43 47 24m-47-14 29 15" stroke="#9cc9bb" strokeWidth="2"/>
          <path d="m-16-33-168 84m191-91 53 27m38 19 115 57" className="station-room-light" fill="none" strokeWidth="3"/>
          <path d="m-181 105 20-10 22 11v35l-20 10-22-11Z" fill="#28475b" stroke="#90a8ae"/>
          <path d="m-175 114 11 6m-11 3 11 6m-11 3 11 6" stroke="#a4c4c2" strokeWidth="2"/>
        </g>
        {visible.map((pod,i)=><g key={`${pod.workload.namespace}/${pod.name}`} data-object="robot" role="button" tabIndex={0} className="station-bot-target" opacity={daemonFocus?workloadKey(pod.workload)===daemonFocus?1:.18:activeNamespace&&pod.workload.namespace!==activeNamespace||highlight&&!pod.workload.name.toLowerCase().includes(highlight.toLowerCase().replace('otel ','otel-'))?.18:1} aria-label={`${pod.name}, namespace ${pod.workload.namespace}, ${pod.workload.kind}: ${labels[mood(pod)]}. Open ${pod.workload.name}`} transform={`translate(${(i%3-Math.floor(i/3))*64} ${58+(i%3+Math.floor(i/3))*32})`} onMouseEnter={()=>describe(pod)} onMouseLeave={()=>setDaemonFocus(null)} onFocus={()=>describe(pod)} onBlur={()=>setDaemonFocus(null)} onClick={()=>onOpen(pod.workload)} onKeyDown={e=>activate(e,()=>onOpen(pod.workload))} style={{'--walk-delay':`${-i*1.7}s`} as CSSProperties}><title>{pod.name} — {labels[mood(pod)]}</title><circle cy="-20" r="29" className="station-hit"/><ellipse cy="5" rx="22" ry="9" fill="none" stroke={namespaceColor(pod.workload.namespace)} strokeWidth="3"/><g className="robot-nametag" aria-hidden="true"><rect x="-66" y="-83" width="132" height="19" rx="5" fill="#152a40" stroke={namespaceColor(pod.workload.namespace)}/><text y="-70" textAnchor="middle">{pod.workload.name.length>21?pod.workload.name.slice(0,19)+'…':pod.workload.name}</text></g><Robot state={mood(pod)} accent={namespaceColor(pod.workload.namespace)} daemon={pod.workload.kind==='DaemonSet'} role={workRole(pod.workload.name)} simulated={simulated} identity={`${pod.workload.namespace}/${pod.name}`}/></g>)}
        {visible.map((pod,i)=><g key={pod.name+'-identity'} data-object="identity" role="button" tabIndex={0} className="station-object" aria-label={`Identity for ${pod.name}: permissions unknown`} transform={`translate(${(i%3-Math.floor(i/3))*64+24} ${58+(i%3+Math.floor(i/3))*32-28})`} onClick={()=>{setIdentityPod(`${pod.workload.namespace}/${pod.name}`);setBoundary('identity')}} onKeyDown={e=>activate(e,()=>{setIdentityPod(`${pod.workload.namespace}/${pod.name}`);setBoundary('identity')})}><title>Identity · permissions unknown</title><rect x="-5" y="-7" width="10" height="14" rx="2" fill="#d8ddcf" stroke="#38546a"/><circle cy="-2" r="2" fill="#607b8b"/><path d="M-2 3h4" stroke="#607b8b"/></g>)}
        {room.pods.length>9&&<g data-object="page" role="button" tabIndex={0} className="station-object" transform="translate(0 315)" aria-label={`Next pods in ${room.node||'waiting bay'}`} onClick={()=>setPages({...pages,[room.node]:start+9>=room.pods.length?0:page+1})} onKeyDown={e=>activate(e,()=>setPages({...pages,[room.node]:start+9>=room.pods.length?0:page+1}))}><rect x="-53" y="-16" width="106" height="26" rx="12" fill="#31475e"/><text textAnchor="middle">{start+1}–{Math.min(start+9,room.pods.length)} / {room.pods.length} ›</text></g>}
      </g>})}
      {!rooms.length&&!loading&&<text x="600" y="330" textAnchor="middle">No observed pods. Try another filter.</text>}
      </g>
      <BoundaryObjects onInspect={setBoundary}/>
      <DeliveryStation simulated={simulated} paused={paused}/>
      </g></svg>
      <BoundaryControls simulated={simulated} layer={boundary} onClose={()=>setBoundary(null)} topology={topology} identityPod={identityPod} onRoute={service=>{setRouteService(service);setNetwork(true);setBoundary(null)}}/>
      {routeService&&<button className="route-clear" onClick={()=>setRouteService(null)}>All routes ×</button>}
      {info&&<div className="station-inspect"><button className="station-inspect-close" aria-label="Dismiss object details" onClick={()=>setInfo(null)}>×</button><strong>{info.title}</strong><p>{info.description}</p>{info.workload&&<button onClick={()=>onOpen(info.workload!)}>Inspect workload ↗</button>}</div>}
      {facility&&<TopologyPanel facility={facility} residents={residents} simulated={simulated} onClose={()=>setFacility(null)} onHighlight={name=>{setHighlight(name);setNamespace(null)}}/>}
      {showEvents&&<aside className="station-events" aria-label="Recent observed events"><header><strong>Event log</strong><button aria-label="Close events" onClick={()=>setShowEvents(false)}>×</button></header>{recent.length?recent.map((e,i)=><button key={i} className={e.type==='Warning'?'event-warning':''} onMouseEnter={()=>setInfo({title:e.reason,description:e.message,workload:e.workload})} onFocus={()=>setInfo({title:e.reason,description:e.message,workload:e.workload})} onClick={()=>onOpen(e.workload)}><span>{e.type==='Warning'?'⚠':'●'}</span><div><strong>{e.reason}</strong><small>{e.workload.name}</small><time>{new Date(e.lastSeen).toLocaleTimeString()}</time></div></button>):<p>No events reported for this selection.</p>}<a href="#/events">All cluster events ↗</a></aside>}
    </div>
  </section>
}
