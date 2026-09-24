export type MapService={namespace:string;name:string;type:string;clusterIp:string;ports:string[]}
type ServiceEvidence={services?:MapService[];routes:{namespace:string;service:string}[];endpoints:{namespace:string;service:string}[]}
export const serviceKey=(s:{namespace:string;name:string})=>`${s.namespace}/${s.name}`
export function serviceCatalog(data:ServiceEvidence|null):MapService[]{
 if(!data)return []
 const entries=new Map((data.services||[]).map(s=>[serviceKey(s),s]))
 // Older APIs and partial discovery can still supply route/endpoint evidence.
 for(const e of [...data.routes,...data.endpoints]){
  const key=`${e.namespace}/${e.service}`
  if(!entries.has(key))entries.set(key,{namespace:e.namespace,name:e.service,type:'Unreported',clusterIp:'Unreported',ports:[]})
 }
 return [...entries.values()].sort((a,b)=>serviceKey(a).localeCompare(serviceKey(b)))
}
export function serviceSlot(index:number){
 const u=40+(index%3)*110,v=40+Math.floor(index/3)*100
 return {x:1020+u-v,y:260+(u+v)/2}
}
