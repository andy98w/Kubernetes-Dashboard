import {useEffect,useState} from 'react'
import type {Resident} from './station'
export type TopologyData={
 routes:{namespace:string;ingress:string;host:string;path:string;service:string;port:string}[]
 endpoints:{namespace:string;service:string;pod:string;podUID:string;addresses:string[];ready:boolean|null}[]
 identities:{namespace:string;pod:string;serviceAccount:string;policies:string[];grants:{scope:string;binding:string;rules:unknown[]}[]}[]
 policies:{namespace:string;name:string;spec:unknown}[]
 externalNames:Record<string,string>;warnings:string[];observedAt:string
}
export function useTopology(simulated:boolean,residents:Resident[]){
 const [data,setData]=useState<TopologyData|null>(null)
 useEffect(()=>{
  if(simulated)return
  let active=true;const controller=new AbortController()
  const refresh=async()=>{try{
   const r=await fetch('/api/v1/topology',{signal:AbortSignal.any([controller.signal,AbortSignal.timeout(10000)])});if(!r.ok)throw Error()
   const value=await r.json();if(!Array.isArray(value.routes)||!Array.isArray(value.endpoints)||!Array.isArray(value.identities))throw Error()
   if(active)setData(value)
  }catch{if(active)setData(null)}}
  void refresh();const timer=setInterval(()=>void refresh(),30000)
  return()=>{active=false;controller.abort();clearInterval(timer)}
 },[simulated])
 if(!simulated)return data
 // Explicit demo fixtures; never used as a live discovery fallback.
 return {
  routes:[{namespace:'kubevista',ingress:'kubevista',host:'kubevista.illuma.me',path:'/api',service:'kubevista-api',port:'8080'},{namespace:'kubevista',ingress:'kubevista',host:'kubevista.illuma.me',path:'/',service:'kubevista-web',port:'80'}],
  endpoints:residents.filter(p=>p.workload.namespace==='kubevista').map(p=>({namespace:p.workload.namespace,service:p.workload.name,pod:p.name,podUID:'',addresses:[],ready:p.ready===p.containers})),
  identities:residents.map(p=>({namespace:p.workload.namespace,pod:p.name,serviceAccount:p.workload.name,policies:[],grants:[]})),
  policies:[],externalNames:{},warnings:['Demo route and identity fixtures. No runtime traffic or effective permissions measured.'],observedAt:new Date().toISOString(),
 } satisfies TopologyData
}
