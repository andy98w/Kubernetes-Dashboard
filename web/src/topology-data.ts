import {useEffect,useState} from 'react'
import type {Resident} from './station'
import {isFleetDemo} from './demo-data'
import {fleetApps} from './fleet'
export type NodeFact={name:string;zone:string;cpuCapacity:number;memoryCapacity:number;cpuMilli:number|null;memoryBytes:number|null;pressure:string[]}
export type VolumeFact={namespace:string;pod:string;name:string;kind:string;claim:string;phase:string;capacity:string;storageClass:string}
export type PendingFact={namespace:string;pod:string;reason:string;message:string}
export type TopologyData={
 nodes?:NodeFact[]
 volumes?:VolumeFact[]
 pending?:PendingFact[]
 services?:{namespace:string;name:string;type:string;clusterIp:string;ports:string[]}[]
 routes:{namespace:string;ingress:string;host:string;path:string;service:string;port:string}[]
 endpoints:{namespace:string;service:string;pod:string;podUID:string;addresses:string[];ready:boolean|null}[]
 identities:{namespace:string;pod:string;serviceAccount:string;policies:string[];grants:{scope:string;binding:string;rules:unknown[]}[]}[]
 policies:{namespace:string;name:string;spec:unknown}[]
 externalNames:Record<string,string>;warnings:string[];observedAt:string
}
export function useTopology(simulated:boolean,residents:Resident[]):TopologyData|null{
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
  services:[...new Map(residents.flatMap(p=>p.services.map(name=>{const short=name.includes('/')?name.split('/').at(-1)!:name;return [`${p.workload.namespace}/${short}`,{namespace:p.workload.namespace,name:short,type:'ClusterIP',clusterIp:'Not allocated',ports:[(fleetApps.find(a=>a.namespace===p.workload.namespace&&a.name===short)?.port||(short==='kubevista-web'?'80':short==='kubevista-api'?'8080':''))].filter(Boolean).map(port=>port+'/TCP')}] as const}))).values()],
  routes:[{namespace:'kubevista',ingress:'kubevista',host:'kubevista.illuma.me',path:'/api',service:'kubevista-api',port:'8080'},{namespace:'kubevista',ingress:'kubevista',host:'kubevista.illuma.me',path:'/',service:'kubevista-web',port:'80'},...(isFleetDemo?fleetApps.filter(a=>a.port).map(a=>({namespace:a.namespace,ingress:'proposed-'+a.namespace,host:a.namespace+'.example.test',path:a.name.endsWith('-api')?'/api':'/',service:a.name,port:a.port})):[])],
  endpoints:residents.flatMap(p=>p.services.map(name=>({namespace:p.workload.namespace,service:name.includes('/')?name.split('/').at(-1)!:name,pod:p.name,podUID:'',addresses:[],ready:p.ready===p.containers}))),
  identities:residents.map(p=>({namespace:p.workload.namespace,pod:p.name,serviceAccount:p.workload.name,policies:[],grants:[]})),
  policies:[],externalNames:{},warnings:['Traffic and effective permissions are unverified.'],observedAt:new Date().toISOString(),
 } satisfies TopologyData
}
