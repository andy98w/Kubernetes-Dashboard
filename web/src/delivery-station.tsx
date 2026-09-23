import {useEffect,useState} from 'react'
import {createPortal} from 'react-dom'
type Rollout={name:string;namespace:string;images:string[];desired:number;updated:number;ready:number;state:string}
type Snapshot={workloads:Rollout[];observedAt:string;buildState:string;buildRevision:string;argoState:string}
const steps=['Commit received','Tests and scans','Signed image stored','GitOps promotion','Rolling deployment','Ready']
export function DeliveryStation({simulated,paused}:{simulated:boolean;paused:boolean}){
 const [open,setOpen]=useState(false),[step,setStep]=useState(-1),[fail,setFail]=useState(false),[data,setData]=useState<Snapshot|null>(null)
 const [panelTarget,setPanelTarget]=useState<Element|null>(null)
 useEffect(()=>{setPanelTarget(document.querySelector('.station-stage'))},[])
 useEffect(()=>{
  if(paused||!simulated||step<0||step>=5||(fail&&step===4))return
  const timer=setTimeout(()=>setStep(s=>s+1),2200);return()=>clearTimeout(timer)
 },[simulated,step,fail,paused])
 useEffect(()=>{
  if(simulated)return
  let active=true;const abort=new AbortController()
  const refresh=async()=>{try{const r=await fetch('/api/v1/delivery',{signal:AbortSignal.any([abort.signal,AbortSignal.timeout(8000)])});if(!r.ok)throw Error();const d=await r.json();if(!Array.isArray(d.workloads)||!Number.isFinite(Date.parse(d.observedAt))||Date.now()-Date.parse(d.observedAt)>45000)throw Error();if(active)setData(d)}catch{if(active)setData(null)}}
  void refresh();const timer=setInterval(refresh,15000);return()=>{active=false;abort.abort();clearInterval(timer)}
 },[simulated])
 const names=['GitHub','CI workshop','ECR'],colors=['#c8b6e7','#9ed5d8','#e8c58a']
 const cratePoints=[[145,33],[260,90],[375,148],[610,338],[280,603],[280,603]]
 return <>
  <g className="delivery-yard" transform="translate(145 100)">
   <path d="M0 0l115 57.5 115 57.5" fill="none" stroke="#b7a2dc" strokeWidth="3" strokeDasharray="5 6"/>
   {names.map((name,i)=><g key={name} transform={`translate(${i*115} ${i*57.5})`} data-object="delivery" role="button" tabIndex={0} className="station-object" aria-label={`Inspect ${name} delivery ${simulated?'demo':'status'}`} onClick={()=>setOpen(!open)} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();setOpen(!open)}}}>
    <title>{name}: {simulated?'simulated release':'build evidence unavailable'}</title>
    <path d="m-54 0 54-27 54 27v10L0 37l-54-27Z" fill="#30485e" stroke="#7895a8"/><path d="m-54 0 54-27 54 27L0 27Z" fill="#536f80"/>
    <path d="m-28 0 0-44 28-14 30 15v44L0 15Z" fill={colors[i]} stroke="#355267"/><path d="M0-29 30-43v44L0 15Z" fill="#526c80"/><path d="m-28-44 28-14 30 15L0-29Z" fill={colors[i]}/>
    {i===0?<path d="M-18-30v24m0-12 12-6v-10" stroke="#394a62" strokeWidth="3" fill="none"/>:i===1?<><path d="m-22-32 16 8v14l-16-8Z" fill="#223b50"/><circle cx="-13" cy="-23" r="3" fill={simulated&&step===1?'#eec884':'#9db3c2'}/></>:<><path d="m6-20 18-9M6-9l18-9M6 2l18-9" stroke="#d8bd8c" strokeWidth="4"/></>}
    <text y="54" textAnchor="middle">{name}</text>
   </g>)}
   <text x="105" y="-75" textAnchor="middle" fontSize="11">{simulated?'Delivery demo':'Delivery · CI unknown'}</text>
  </g>
  {simulated&&step>=0&&<g style={{transform:`translate(${cratePoints[step][0]}px, ${cratePoints[step][1]}px)`}} className="release-crate"><path d="m-9 0 9-5 9 5v10l-9 5-9-5Z" fill={fail&&step===4?'#ed9a8d':'#c4e3c6'} stroke="#466474"/><text x="16" y="5">demo-r2</text><title>{steps[step]} · simulated release crate</title></g>}
  <g transform="translate(610 370)" data-object="delivery" role="button" tabIndex={0} className="station-object" aria-label="Inspect Argo CD dispatch" onClick={()=>setOpen(!open)} onKeyDown={e=>{if(e.key==='Enter'){setOpen(!open)}}}><title>Argo CD dispatch · logical reconciliation desk</title><path d="m-23 0 0-22 30-15 22 11v22l-30 15Z" fill="#aab6cf" stroke="#456175"/><path d="M-10-14v-15l19-9v15Z" fill="#263e54"/><text x="0" y="30" textAnchor="middle">Argo CD</text></g>
  {/* Instructions and image pulls are distinct; neither is measured traffic. */}
  <path d="M260 157.5L680 367.5L720 347.5" fill="none" stroke="#b7a2dc" strokeWidth="2" strokeDasharray="4 7" pointerEvents="none"><title>GitOps instructions toward the control plane; schematic</title></path>
  <path d="M375 215L555 305L195 485" fill="none" stroke="#e8c58a" strokeWidth="2" strokeDasharray="10 6" pointerEvents="none"><title>Worker image pulls from ECR; schematic, not observed downloads</title></path>
  {simulated&&step>=3&&<g transform="translate(680 365)" aria-label="Simulated rollout"><circle r="12" fill={fail&&step===4?'#e99d8f':step===5?'#a6e5c7':'#ebcf89'}/><text x="18" y="4">{fail&&step===4?'Rollout failed':steps[step]}</text></g>}
  {open&&panelTarget&&createPortal(<aside className="topology-panel delivery-inspector" aria-label="Release delivery"><button onClick={()=>setOpen(false)} aria-label="Close delivery inspection">×</button><h3>Release delivery</h3>{simulated?<><p>Simulation only · no builds or deployments are triggered.</p><p role="status">{step<0?'Ready to rehearse':fail&&step===4?'Readiness failed. Promotion stops; revert the release PR to recover.':steps[step]}</p><button onClick={()=>{setFail(false);setStep(0)}}>Play release</button><button onClick={()=>{setFail(true);setStep(0)}}>Try failed rollout</button></>:<><p>Publisher: {data?.buildState||'unknown'} · {data?.buildRevision.slice(0,7)}<br/>Argo CD: {data?.argoState||'unknown'}. ECR contents/signatures are not independently verified here.</p>{data?<><small>Observed {new Date(data.observedAt).toLocaleTimeString()}</small>{data.workloads.map(w=><p key={w.namespace+'/'+w.name}>{w.namespace}/{w.name}: {w.state} · {w.ready}/{w.desired} ready<br/>{w.images.join(', ')}</p>)}{!data.workloads.length&&<p>No matching Deployments returned.</p>}</>:<p>Rollout inventory unavailable.</p>}</>}</aside>,panelTarget)}
 </>
}
