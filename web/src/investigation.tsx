import {useEffect,useId,useRef,useState} from 'react'
import type {EvidenceDetail} from './incident-response'
import './investigation.css'

type Packet={mode:string;observedAt:string;evidence:Array<{id:string;source:string;text:string}>;hypotheses:Array<{explanation:string;evidenceIds:string[]}>;warnings:string[]}
export function Investigation({detail,simulated}:{detail:EvidenceDetail;simulated:boolean}){
 const [token,setToken]=useState('')
 const [packet,setPacket]=useState<Packet|null>(null),[busy,setBusy]=useState(false),[error,setError]=useState(''),[logs,setLogs]=useState(false)
 const request=useRef<AbortController|null>(null),prefix=useId()
 const target=detail.workload
 const key=target?`${target.namespace}/${target.kind}/${target.name}`:''
 useEffect(()=>{request.current?.abort();setPacket(null);setError('');setBusy(false);setLogs(false);return()=>request.current?.abort()},[key,detail.observedAt])
 async function investigate(){
  request.current?.abort()
  const controller=new AbortController();request.current=controller
  setBusy(true);setError('');setPacket(null)
  try{
   let result:Packet
   if(simulated){
    const evidence=(detail.diagnoses||[]).map((d,i)=>({id:`E${i+1}`,source:d.resource,text:`${d.code}: ${d.evidence} Next check: ${d.nextStep}`}))
    if(detail.recovery)evidence.push({id:`E${evidence.length+1}`,source:'Recovery observation',text:`${detail.recovery.status}: ${detail.recovery.detail}`})
    result={mode:'demo-evidence',observedAt:detail.observedAt,evidence,hypotheses:[],warnings:['Synthetic browser evidence. No AI model or live cluster was queried.','Logs and Prometheus samples are not collected in this preview.']}
   }else{
    if(!target)throw new Error('Select a workload first.')
    const response=await fetch(`/api/v1/investigate/${[target.namespace,target.kind,target.name].map(encodeURIComponent).join('/')}`,{method:'POST',headers:{Authorization:'Bearer '+token,'X-KubeVista-Investigation':'reviewed',...(logs?{'X-KubeVista-Include-Logs':'true'}:{})},signal:AbortSignal.any([controller.signal,AbortSignal.timeout(15000)])})
    if(!response.ok)throw new Error(response.status===404?'Investigation is disabled or this workload is unavailable.':response.status===401?'Enter a valid local investigation credential.':response.status===403?'This workload is not allowed for investigation.':response.status===429?'Another investigation is running. Try again shortly.':'Investigation unavailable. Please retry.')
    result=await response.json() as Packet
   }
   if(!controller.signal.aborted)setPacket(result)
  }catch(e){if(!controller.signal.aborted)setError(e instanceof Error?e.message:'Investigation failed.')}
  finally{if(!controller.signal.aborted)setBusy(false)}
 }
 return <section className="investigation" aria-label="Read-only investigation">
  <div className="investigation-heading"><div><h3>Investigate</h3><small>Read-only · no operations executed</small></div><button disabled={busy||!target} onClick={()=>void investigate()}>{busy?'Investigating…':packet?'Investigate again':'Investigate workload'}</button></div>
  {!simulated&&<label><input type="checkbox" checked={logs} disabled={busy} onChange={e=>{setLogs(e.target.checked);setPacket(null)}}/>Include bounded logs and metrics (sanitized workloads only)</label>}
  {!simulated&&<label>Local investigation credential <input type="password" autoComplete="off" value={token} onChange={e=>setToken(e.target.value)}/></label>}
  <p className="investigation-note">Evidence stays with the API and its configured local model. Redaction is best-effort; do not include sensitive customer logs.</p>
  {busy&&<p role="status">Collecting bounded evidence… <button onClick={()=>{request.current?.abort();setBusy(false)}}>Cancel</button></p>}
  {error&&<p role="alert">{error}</p>}
  {packet&&<div aria-live="polite">
   <p><strong>{packet.mode==='local-ai'?'AI hypotheses — verify against evidence':packet.mode==='demo-evidence'?'Local evidence · no AI model':'Evidence only · no AI answer'}</strong><br/><small>Observed {new Date(packet.observedAt).toLocaleString()}</small></p>
   {packet.hypotheses.length?packet.hypotheses.map((h,i)=><article key={i}><p>{h.explanation}</p><nav aria-label={`Evidence for hypothesis ${i+1}`}>{h.evidenceIds.map(id=><a key={id} href={`#${prefix}-${id}`} onClick={e=>{e.preventDefault();document.getElementById(`${prefix}-${id}`)?.scrollIntoView({block:'nearest'});document.getElementById(`${prefix}-${id}`)?.focus()}}>{id}</a>)}</nav></article>):<p>No model hypotheses available. Review the observations below; missing evidence does not establish health.</p>}
   <details open><summary>Evidence ({packet.evidence.length})</summary>{packet.evidence.map(e=><article id={`${prefix}-${e.id}`} tabIndex={-1} key={e.id}><strong>{e.id} · {e.source}</strong><pre>{e.text}</pre></article>)}</details>
   {packet.warnings.map((warning,i)=><p className="investigation-note" key={i}>{warning}</p>)}
  </div>}
 </section>
}
