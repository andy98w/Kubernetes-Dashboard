import {useEffect,useState} from 'react'
import {demoWorkloadDetail,isDemoMode} from './demo-data'
import {Station,type Resident,type StationEvent,type StationWorkload} from './station'
import './habitat.css'

export function Habitat({items,revision,onOpen}:{items:StationWorkload[];revision:string;onOpen:(item:StationWorkload)=>void}) {
  const [residents,setResidents]=useState<Resident[]>([]),[events,setEvents]=useState<StationEvent[]>([])
  const [loading,setLoading]=useState(true),[errors,setErrors]=useState(0)
  const key=JSON.stringify(items.slice(0,24))
  useEffect(()=>{
    const controller=new AbortController();let active=true
    setLoading(true);setErrors(0)
    const selected=JSON.parse(key) as StationWorkload[]
    const collected:Resident[]=[],observations:StationEvent[]=[];let failures=0,cursor=0
    async function worker(){
      while(cursor<selected.length&&active){
        const item=selected[cursor++]
        try{
          let detail
          if(isDemoMode) detail=demoWorkloadDetail(item)
          else {
            const response=await fetch(`/api/v1/workloads/${encodeURIComponent(item.namespace)}/${encodeURIComponent(item.kind)}/${encodeURIComponent(item.name)}`,{signal:AbortSignal.any([controller.signal,AbortSignal.timeout(10000)])})
            if(!response.ok)throw new Error('Unavailable')
            detail=await response.json()
          }
          if(!Array.isArray(detail.pods))throw new Error('Missing pods')
          const services=(detail.services||[]).map((s:{namespace:string;name:string})=>`${s.namespace}/${s.name}`)
          collected.push(...detail.pods.map((pod:Omit<Resident,'workload'|'services'>)=>({...pod,workload:item,services})))
          observations.push(...(detail.events||[]).map((event:Omit<StationEvent,'workload'>)=>({...event,workload:item})))
          if(isDemoMode) observations.push(...(detail.timeline||[]).map((event:{source:string;detail:string;at:string})=>({type:'Normal',reason:event.source,message:event.detail,lastSeen:event.at,workload:item})))
        }catch{failures++}
      }
    }
    void Promise.all(Array.from({length:4},worker)).then(()=>{if(active){setResidents(collected.sort((a,b)=>a.name.localeCompare(b.name)));setEvents(observations);setErrors(failures);setLoading(false)}})
    return()=>{active=false;controller.abort()}
  },[key,revision])
  return <Station residents={residents} events={events} simulated={isDemoMode} loading={loading} errors={errors} truncated={items.length>24} onOpen={onOpen}/>
}
