import {useEffect,useState} from 'react'
type Signal={state:'ready'|'warning'|'unknown';detail:string}
export type FacilityHealth={control:Signal;services:Signal;telemetry:Signal;observedAt:string}
const unknown:Signal={state:'unknown',detail:'No fresh health observation available.'}
const empty:FacilityHealth={control:unknown,services:unknown,telemetry:unknown,observedAt:''}
export function useFacilityHealth(simulated:boolean){
 const [health,setHealth]=useState(empty)
 useEffect(()=>{
   let active=true
   const controller=new AbortController()
   async function refresh(){
     if(simulated){const demo:Signal={state:'ready',detail:'Simulated readiness. No live cluster is being probed.'};setHealth({control:demo,services:demo,telemetry:demo,observedAt:new Date().toISOString()});return}
     try{const response=await fetch('/api/v1/station-health',{signal:AbortSignal.any([controller.signal,AbortSignal.timeout(8000)])});if(!response.ok)throw Error();const data=await response.json();if(!['control','services','telemetry'].every(k=>['ready','warning','unknown'].includes(data[k]?.state))||!Number.isFinite(Date.parse(data.observedAt))||Date.now()-Date.parse(data.observedAt)>45000)throw Error();if(active)setHealth(data)}catch{if(active)setHealth(empty)}
   }
   void refresh();const timer=setInterval(()=>void refresh(),15000)
   const expiry=setInterval(()=>setHealth(h=>h.observedAt&&Date.now()-Date.parse(h.observedAt)>45000?empty:h),5000)
   return()=>{active=false;controller.abort();clearInterval(timer);clearInterval(expiry)}
 },[simulated])
 return health
}
