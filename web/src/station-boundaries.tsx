import {useEffect,useState} from 'react'
import {demoData} from './demo-data'
import type {TopologyData} from './topology-data'
type Network={ingresses:{namespace:string;name:string;class:string;hosts:string[];address:string}[];policies:{namespace:string;name:string;ingressRules:number;egressRules:number}[];observedAt:string}
export type Layer='entry'|'egress'|'policy'|'identity'
export function BoundaryObjects({onInspect}:{onInspect:(layer:Layer)=>void}){
 return <g className="boundary-objects">
  {([{kind:'entry',x:400,label:'Internet / ingress'},{kind:'policy',x:800,label:'Policy gate · unknown'},{kind:'egress',x:1200,label:'Egress · unknown'}] as const).map(o=><g key={o.kind} data-object="boundary" className="station-object" role="button" tabIndex={0} aria-label={o.label} transform={`translate(${o.x} 110)`} onClick={()=>onInspect(o.kind)} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();onInspect(o.kind)}}}>
   <title>{o.label} · conceptual object, inspect configuration</title>
   <path d="m-95 10 95-45 95 45v12L0 67l-95-45Z" fill="#293f54" stroke="#8da6b4"/>
   <path d="m-95 10 95-45 95 45L0 55Z" fill="#536f80"/>
   {o.kind==='entry'?<g>
    <path d="M-59-30c-24 0-24-31-4-35 0-28 38-33 47-13 25-9 42 10 32 29 18 18 4 32-15 26Z" fill="#b9d9e2" stroke="#6b9bab" strokeWidth="2"/>
    <path d="m20 7 0-49 31-15 27 14v49L48 22Z" fill="#638b9a" stroke="#b1ccd1"/>
    <path d="m28-35 18 8v24l-18-8Z" fill="#203d52"/><path d="m51-26 18-9v25l-18 9Z" fill="#9cc8d5"/>
   </g>:o.kind==='policy'?<g>
    <path d="M-48 18v-55l20-10v55ZM35 18v-55l20-10v55Z" fill="#6a8795" stroke="#b4c5c5"/>
    <path d="M-35-24 44 10" stroke="#d8be88" strokeWidth="9" strokeDasharray="10 5"/>
    <circle cy="-51" r="15" fill="#243b50" stroke="#afc2cd"/><text y="-46" textAnchor="middle">?</text>
   </g>:<g>
    <path d="m-51 5 42-21 54 27-42 21Z" fill="#98b0b7"/>
    <path d="M-43 0v-36l21-11v36Zm71 18v-36l21-11V7Z" fill="#648595" stroke="#b0c4cd"/>
    <path d="m-19-33 44 22m-12-19 12 19-22 1" fill="none" stroke="#d6c18f" strokeWidth="6"/>
    <path d="m-35 17 27 14m-13-21 27 14" stroke="#8299a6" strokeWidth="3"/>
   </g>}
   <text y="86" textAnchor="middle">{o.kind==='entry'?'Internet':o.kind==='policy'?'Policy':'Egress'}</text>
  </g>)}
 </g>
}
export function BoundaryControls({simulated,layer,onClose,topology,identityPod,onRoute}:{simulated:boolean;layer:Layer|null;onClose:()=>void;topology:TopologyData|null;identityPod:string|null;onRoute:(service:string)=>void}){
 const [data,setData]=useState<Network|null>(null),[loading,setLoading]=useState(false)
 useEffect(()=>{
  if(!layer)return
  let active=true;const controller=new AbortController()
  setData(null);setLoading(true)
  void (async()=>{try{
   const value=simulated?demoData('network'):await fetch('/api/v1/network',{signal:AbortSignal.any([controller.signal,AbortSignal.timeout(8000)])}).then(r=>{if(!r.ok)throw Error();return r.json()})
   if(active)setData(value as Network)
  }catch{if(active)setData(null)}finally{if(active)setLoading(false)}})()
  return()=>{active=false;controller.abort()}
 },[layer,simulated])
 useEffect(()=>{if(!layer)return;const close=(e:KeyboardEvent)=>{if(e.key==='Escape')onClose()};window.addEventListener('keydown',close);return()=>window.removeEventListener('keydown',close)},[layer,onClose])
 return <>
  {layer&&<aside className="topology-panel boundary-panel" aria-label="Boundary inspection"><header><h2>{{entry:'Internet → cluster',egress:'Outbound access',policy:'NetworkPolicy gates',identity:'Workload identity'}[layer]}</h2><button onClick={onClose} aria-label="Close boundary inspection">×</button></header>
   <p className="topology-source">{simulated?'Demo configuration':'API inventory'} · not a traffic or authorization test</p>
   {topology?.warnings.map(w=><p key={w}>{w}</p>)}
   {layer==='entry'&&topology?.routes.map((r,i)=><button key={i} className="topology-service-path" onClick={()=>onRoute(r.namespace+'/'+r.service)}>{r.host||'Default host'}{r.path} → {r.namespace}/{r.service}:{r.port}<small>{topology.endpoints.filter(e=>e.namespace===r.namespace&&e.service===r.service).length} discovered endpoints · select to trace</small></button>)}
   {layer==='identity'&&topology?.identities.filter(p=>p.namespace+'/'+p.pod===identityPod).map(p=><section key={p.pod}><h3>{p.namespace}/{p.serviceAccount||'(not reported)'}</h3><p>{p.pod}</p>{p.grants.map((g,i)=><details key={i}><summary>{g.binding} · scope {g.scope}</summary><pre>{JSON.stringify(g.rules,null,2)}</pre></details>)}<p>{p.grants.length?'Declared RBAC grants, not an effective authorization test.':'No grants returned; check discovery warnings. This does not prove denial.'}</p><p>Selected policies: {p.policies.join(', ')||'None returned'}</p></section>)}
   {layer==='policy'&&topology?.policies.map(p=><details key={p.namespace+'/'+p.name}><summary>{p.namespace}/{p.name}</summary><pre>{JSON.stringify(p.spec,null,2)}</pre></details>)}
   {layer==='egress'&&Object.entries(topology?.externalNames||{}).map(([service,host])=><p key={service}>{service} → {host} (ExternalName configuration, not observed traffic)</p>)}
   {layer==='egress'&&topology&&!Object.keys(topology.externalNames).length&&<p>No ExternalName destinations returned. No Egress link is drawn; this does not mean outbound access is blocked.</p>}
   {layer==='policy'&&topology&&!topology.identities.some(p=>p.policies.length)&&<p>No pod-policy associations returned. No Policy links are drawn; access remains unknown.</p>}
   {loading?<p>Loading inventory…</p>:!data?<p>Inventory unavailable. Access state is unknown.</p>:<>
    <small>Observed {new Date(data.observedAt).toLocaleTimeString()}</small>
    {layer==='entry'&&<><p>The Internet is outside the cluster. These are declared Ingress resources and their reported addresses—not verified reachability.</p>{data.ingresses.map(i=><section key={i.namespace+'/'+i.name}><h3>{i.namespace}/{i.name}</h3><p>{i.class||'Class unknown'} · {i.hosts.join(', ')||'No host reported'}</p><p>{i.address||'Load balancer address not reported'}</p></section>)}{!data.ingresses.length&&<p>No Ingress resources returned. Other entry mechanisms may exist.</p>}<p>Gateway API routes are not loaded. Selected Ingress routes use EndpointSlice target references; an endpoint need not be ready.</p><a href="#/network">Inspect network inventory</a></>}
    {layer==='egress'&&<><p>ExternalName destinations appear above when discovered. NAT paths and runtime outbound calls are not loaded.</p><p>Egress rule counts appear under Policy gates; counts alone cannot determine which connections are allowed.</p></>}
    {layer==='policy'&&<><p>Gate state: unknown. Declared selectors, peers, and ports are expandable above. End-to-end policy decisions and CNI enforcement are not evaluated. A namespace color is not a security boundary.</p>{data.policies.map(p=><section key={p.namespace+'/'+p.name}><h3>◇ {p.namespace}/{p.name}</h3><p>{p.ingressRules} ingress / {p.egressRules} egress rules · enforcement unknown</p></section>)}</>}
    {layer==='identity'&&<><p>Service accounts and declared grants appear above when discovery succeeds. Effective permissions are not evaluated.</p><p>RBAC controls Kubernetes API operations. NetworkPolicy governs supported pod traffic. Neither should be inferred from the other.</p><p>Robot colors identify namespaces; wrench badges identify DaemonSets, not access privileges.</p><a href="#/security">Inspect available security findings</a></>}
   </>}
  </aside>}
 </>
}
