import {useMemo} from 'react'
import {angledConnection} from './station-connections'
import {floorPlan,nodeOrigin,project} from './station-floor'
export const cloudAnchors={igw:{x:960,y:255},alb:{x:1130,y:340},nat:{x:1370,y:340}}
type Inspect=(title:string,description:string)=>void
/** Infrastructure plan, only mounted for the planned fleet. No AWS discovery. */
export function CloudBoundaries({count,onInspect}:{count:number;onInspect:Inspect}){
 const perimeter=useMemo(()=>{
  const occupied=new Set<string>()
  for(const {u,v} of floorPlan(count).tiles)for(let du=-3;du<=3;du++)for(let dv=-3;dv<=3;dv++)occupied.add(`${u+du},${v+dv}`)
  for(let u=-4;u<24;u++)for(let v=-24;v<1;v++)occupied.add(`${u},${v}`)
  const edges:string[]=[]
  for(const key of occupied){
   const [u,v]=key.split(',').map(Number)
   for(const [du,dv,a,b] of [[-1,0,[u,v],[u,v+1]],[0,-1,[u,v],[u+1,v]],[1,0,[u+1,v],[u+1,v+1]],[0,1,[u,v+1],[u+1,v+1]]] as [number,number,number[],number[]][]){
    if(occupied.has(`${u+du},${v+dv}`))continue
    const start=project(a[0],a[1]),end=project(b[0],b[1])
    edges.push(`M${start.x} ${start.y+230}L${end.x} ${end.y+230}`)
   }
  }
  return edges.join(' ')
 },[count])
 const label=project(-4,6)
 return <g className="cloud-boundaries">
  <path d={perimeter} fill="none" stroke="#d9ae74" strokeWidth="1" strokeLinejoin="miter" strokeDasharray="8 8" opacity=".5" pointerEvents="none"/>
  <g data-object="cloud-boundary" role="button" tabIndex={0} className="station-object" onClick={()=>onInspect('AWS · us-west-2','Infrastructure plan based on Terraform defaults. No live AWS resource discovery or account ID is loaded. Managed EKS control-plane hosts belong to AWS, outside the customer VPC.')} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();onInspect('AWS · us-west-2','Terraform default region; no live AWS resource discovery or account ID is loaded.')}}} aria-label="Inspect AWS boundary"><text className="map-label-source" x={label.x-12} y={label.y+215} textAnchor="end" fill="#d9ae74">AWS · us-west-2</text></g>
  {Array.from({length:count},(_,index)=>{
   const first=nodeOrigin(index),u=first.u-1,v=first.v-1,w=20,h=20
   const points=[[u,v],[u+w,v],[u+w,v+h],[u,v+h]].map(([a,b])=>{const p=project(a,b);return `${p.x},${p.y+230}`}).join(' ')
   const p=project(u,v+h)
   return <g key={index} pointerEvents="none"><polygon points={points} fill="none" stroke="#89b6c4" strokeWidth="1.2" strokeDasharray="5 7" opacity=".65"/><text className="map-label-source" x={p.x+10} y={p.y+260} style={{fontSize:13,fill:'#89b6c4'}}>VPC · private worker subnets</text></g>
  })}
 </g>
}
export function CloudObjects({connections,onInspect}:{connections:boolean;onInspect:Inspect}){
 const descriptions={
  igw:'Internet gateway attached to the VPC. Public subnet routes provide Internet access for the internet-facing ALB and public NAT gateway. This is the Terraform-based plan, not observed packet traffic.',
  alb:'Shared internet-facing Application Load Balancer. Host and path rules select each app’s Ingress backend; EndpointSlices identify its pod targets. This is an ALB, not an AWS API Gateway. Separate ALBs can be used if isolation requires them.',
  nat:'Single public NAT gateway in the cost-focused Terraform configuration. Private workloads initiate outbound connections through NAT and the Internet gateway; NAT does not accept unsolicited inbound app requests. One NAT is a cost/availability tradeoff; per-AZ NAT is configurable.',
 }
 return <g className="cloud-infrastructure">
  <path d="M1030 290 1260 175 1540 315 1310 430Z" fill="#294252" fillOpacity=".35" stroke="#d9ae74" strokeDasharray="5 7" opacity=".7" pointerEvents="none"/>
  <text className="map-label-source" x="1310" y="440" textAnchor="middle" style={{fontSize:13,fill:'#d9ae74'}}>VPC · public subnets</text>
  {connections&&<g fill="none" strokeWidth="1.5" pointerEvents="none">
   <path d={angledConnection({x:448,y:132},cloudAnchors.igw)+angledConnection(cloudAnchors.igw,cloudAnchors.alb)} stroke="#8fd2dd" className="connection-flow" vectorEffect="non-scaling-stroke"/>
   <path d={angledConnection(cloudAnchors.nat,cloudAnchors.igw)} stroke="#a5cfbb" className="connection-flow" vectorEffect="non-scaling-stroke"/>
  </g>}
  {(['igw','alb','nat'] as const).map(kind=>{
   const {x,y}=cloudAnchors[kind],name={igw:'Internet gateway',alb:'Shared ALB',nat:'NAT gateway'}[kind]
   const inspect=()=>onInspect(name,descriptions[kind])
   return <g key={kind} data-object="cloud-equipment" role="button" tabIndex={0} className="station-object" aria-label={`Inspect ${name}`} transform={`translate(${x} ${y})`} onClick={inspect} onMouseEnter={inspect} onFocus={inspect} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();inspect()}}}>
    <title>{name}</title><path d="m-58 5 58-29 58 29v8L0 42l-58-29Z" fill="#3c5c6d" stroke="#8ca5af"/>
    {kind==='igw'?<g fill="#91b9be" stroke="#324f64"><path d="M-32 8v-60l12-6 12 6V8ZM20 8v-60l12-6 12 6V8Z"/><path d="m-32-52 64-32 12 6-64 32Z"/><path d="M-14-18 20-1m-7-13 7 13-16-1" fill="none" stroke="#edcc8c" strokeWidth="4"/></g>:<g>
     <path d="m-30 5 0-48 30-15 35 18V8L0 25Z" fill="#c7ab7e" stroke="#243e53"/><path d="M0-28 35-40V8L0 25Z" fill="#8c795f"/><path d="m-22-32 15 7v19l-15-7Z" fill="#233c50"/>
     {kind==='alb'?<path d="m-18-23 7 4m-4-2v8m-4-5 8 4" stroke="#9fe2d8" strokeWidth="2"/>:<path d="m-20-19 12 6m-5-7 5 7-7 1" fill="none" stroke="#9fe2d8" strokeWidth="2"/>}
     {[0,10,20].map(n=><path key={n} d={`m7 ${-18+n} 20-9`} stroke="#dccba7" strokeWidth="2"/>)}
    </g>}
    <text className="map-label-source" y="61" textAnchor="middle" style={{fontSize:13}}>{name}</text>
   </g>
  })}
 </g>
}
