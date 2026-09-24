import {useEffect,useState} from 'react'
import {Investigation} from './investigation'

export type EvidenceDetail = {
 workload?:{namespace:string;kind:string;name:string}
 diagnoses?:Array<{code:string;resource:string;evidence:string;nextStep:string}>
 timeline?:Array<{source:string;detail:string;at:string}>
 recovery?:{status:string;detail:string;generation:number;observedGeneration:number}
 warnings?:string[]
 logs?:Array<{pod:string;container:string;previous:boolean;text:string}>
 metrics?:Array<{name:string;value:string;at:string}>
 observedAt:string
}

export function IncidentResponse({initial,load,simulated}:{initial:EvidenceDetail;load:(enrich:boolean)=>Promise<EvidenceDetail>;simulated:boolean}) {
 const [detail,setDetail]=useState(initial),[busy,setBusy]=useState(false),[error,setError]=useState('')
 const hasIncident=!!detail.timeline?.some(event=>event.source==='Simulated rollout')
 const recovered=detail.recovery?.status==='Recovered'
 useEffect(()=>{setDetail(initial)},[initial])
 const refresh=async(enrich:boolean)=>{
  setBusy(true);setError('')
  try {setDetail(await load(enrich))} catch(e){setError(e instanceof Error?e.message:'Observation failed')} finally{setBusy(false)}
 }
 return <section className="incident-response" aria-label="Incident evidence">
  <div className="response-heading"><h3>Diagnosis & recovery</h3><button disabled={busy} onClick={()=>void refresh(false)}>{busy?'Observing…':'Check recovery'}</button></div>
  {simulated&&<p className="response-muted">Simulation: no Kubernetes resources are changed.</p>}
  {simulated&&hasIncident&&<div className="incident-walkthrough" aria-label="Incident walkthrough">
   <ol aria-label="Recovery stages">
    <li data-complete="true"><span>1</span>Find the fault</li>
    <li data-complete={recovered}><span>2</span>Review a fix</li>
    <li data-complete={recovered}><span>3</span>Verify health</li>
   </ol>
   <p>{recovered?'Recovery observed. Close this inspector to see the robot back at work.':'Read the evidence below, then review a rollback under Guarded operations. After running the simulation, use Check recovery to verify the result.'}</p>
   {!recovered&&<small>A restart keeps the same template—it will not fix an invalid probe, image, or resource request.</small>}
  </div>}
  {error&&<p role="alert">{error}</p>}
  {detail.recovery&&<p role="status"><strong>{detail.recovery.status}</strong> — {detail.recovery.detail}</p>}
  {detail.diagnoses?.length?detail.diagnoses.map((finding,index)=><article className="diagnosis" key={index}><h4>{finding.code}</h4><code>{finding.resource}</code><p>{finding.evidence}</p><p><strong>Next step: </strong>{finding.nextStep}</p></article>):<p>No supported failure signals returned. Review readiness and the observation time.</p>}
  <p className="response-muted">Observed {new Date(detail.observedAt).toLocaleTimeString()}. An accepted operation does not establish recovery.</p>
  <details open><summary>Correlated timeline</summary><ol className="response-timeline">{detail.timeline?.map((e,i)=><li key={i}><time>{new Date(e.at).toLocaleTimeString()}</time><div><strong>{e.source}</strong><p>{e.detail}</p></div></li>)}</ol>{!detail.timeline?.length&&<p>No timeline observations available.</p>}</details>
  <button className="response-enrich" disabled={busy} onClick={()=>void refresh(true)}>Load logs & metrics</button>
  <p className="response-muted">Up to three container log excerpts. Metrics require Prometheus and matching application labels.</p>
  {detail.warnings?.map((w,i)=><p className="response-muted" key={i}>{w}</p>)}
  {detail.metrics?.map(m=><p key={m.name}><strong>{m.name}</strong>: {m.value}</p>)}
  {detail.logs?.map((l,i)=><details key={i}><summary>{l.pod}/{l.container}{l.previous?' (previous container)':''}</summary><pre>{l.text}</pre></details>)}
  <Investigation detail={detail} simulated={simulated}/>
 </section>
}
