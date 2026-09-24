import {useEffect,useState} from 'react'
type Issue={id:string;service:string;release:string;type:string;location:string;count:number;version:number;status:string;firstSeen:string;lastSeen:string}
export function IssueConsole(){
 const [credential,setCredential]=useState(''),[token,setToken]=useState(''),[items,setItems]=useState<Issue[]>([]),[error,setError]=useState(''),[busy,setBusy]=useState(false)
 async function call(path:string,body?:unknown,key=token){
  const response=await fetch('/api/v1/issues'+path,{method:body?'POST':'GET',headers:{Authorization:'Bearer '+key,...(body?{'Content-Type':'application/json'}:{})},...(body?{body:JSON.stringify(body)}:{}),signal:AbortSignal.timeout(8000)})
  if(!response.ok)throw Error(response.status===409?'New events arrived. Refresh before resolving.':response.status===401?'Credential was rejected.':response.status===503?'Configure the local issue service first.':'Issue request failed ('+response.status+').')
  return response.json()
 }
 async function refresh(key=token){const result=await call('',undefined,key);setItems(result.items)}
 useEffect(()=>{
  if(!token)return
  let active=true
  const capture=(event:ErrorEvent)=>{
   // No exception message, arbitrary stack, user URL, cookies, or request body is sent.
   const type=event.error instanceof TypeError?'TypeError':event.error instanceof RangeError?'RangeError':'Error'
   void call('',{service:'kubevista-web',release:'local',type,location:'browser/runtime'})
    .then(()=>{if(active)return refresh()}).catch(e=>{if(active)setError(String(e.message))})
  }
  window.addEventListener('error',capture)
  return()=>{active=false;window.removeEventListener('error',capture)}
 },[token])
 return <section aria-label="Persisted error service"><h3>Connected error service</h3><p>Opt-in browser error capture. Only error category and fixed application identifiers are sent. No messages, stack traces, or customer data.</p>{!token?<form onSubmit={async e=>{e.preventDefault();setBusy(true);setError('');try{await refresh(credential);setToken(credential);setCredential('')}catch(e){setError((e as Error).message)}finally{setBusy(false)}}}><label>Local service credential<input type="password" autoComplete="off" value={credential} onChange={e=>setCredential(e.target.value)} required minLength={32}/></label><button disabled={busy}>Connect</button></form>:<><button onClick={()=>{setToken('');setItems([])}}>Disconnect</button><button onClick={()=>{setError('');void refresh().catch(e=>setError(e.message))}}>Refresh issues</button><button onClick={()=>setTimeout(()=>{throw new TypeError('KubeVista controlled browser error')},0)}>Trigger browser error</button><p>Connected · persisted by the local API. Disconnecting stops capture.</p>{items.map(item=><article key={item.id}><h3>{item.service} · {item.status}</h3><p>{item.type} · {item.count} occurrences · {item.release}</p><code>{item.location}</code><p>Last received: {new Date(item.lastSeen).toLocaleString()}</p><button disabled={item.status==='Resolved'||busy} onClick={async()=>{setBusy(true);setError('');try{await call('/'+item.id+'/resolve',{version:item.version});await refresh()}catch(e){setError((e as Error).message);void refresh().catch(()=>{})}finally{setBusy(false)}}}>Resolve persisted issue</button></article>)}</>}{error&&<p role="alert">{error}</p>}</section>
}
