import type {NodeFact} from './topology-data'
import {equipment} from './station-topology'
import {officeLayout} from './office-props'
import {serviceSlot,serviceKey,type MapService} from './service-layout'
import {cloudAnchors} from './cloud-station'
import {fleetDependencies} from './fleet'
import {layoutMapLabels,type MapLabel} from './map-label-layout'
export function MapLabels({rooms,desks,fleet,nodes}:{nodes?:NodeFact[];rooms:{x:number;y:number;index:number;node:string;pods:unknown[]}[];desks:MapService[];fleet:boolean}){
 const labels:MapLabel[]=[]
 const add=(id:string,text:string,x:number,y:number,heading=false,color?:string)=>labels.push({id,text,x,y,heading,color})
 for(const room of rooms){add('node'+room.index,room.node?`Node ${room.index+1}`:'Waiting bay',room.x,room.y+145,true);add('count'+room.index,`${room.pods.length} pods`,room.x,room.y+670)}
 for(const room of rooms){const zone=nodes?.find(n=>n.name===room.node)?.zone;if(zone)add('zone'+room.index,zone,room.x,room.y+635)}
 for(const [key,x,y,title] of [['control',720,110,'Control plane'],['services',1020,260,'Service routing'],['telemetry',1440,470,'Telemetry']] as const){
  add(key,title,x,y+135,true)
  if(key!=='services')for(const name of equipment[key]){const pos=officeLayout[name];add(key+name,name,x+pos.u-pos.v,y+230+(pos.u+pos.v)/2+30)}
 }
 desks.forEach((s,i)=>{const p=serviceSlot(i);add(serviceKey(s),s.name,p.x,p.y+274)})
 for(const [i,name] of ['GitHub','CI workshop','ECR'].entries())add(name,name,145+i*115,157+i*57.5)
 add('argo','Argo CD',490,330)
 add('internet','Internet',375,5);add('policy','Policy',800,205)
 if(!fleet)add('egress','Egress',1200,205)
 if(fleet){
  for(const kind of ['igw','alb','nat'] as const){const p=cloudAnchors[kind];add(kind,{igw:'Internet gateway',alb:'Shared ALB',nat:'NAT gateway'}[kind],p.x,p.y+65)}
  fleetDependencies.forEach((d,i)=>add(d.id,d.label,125+i*245,-62))
  const p=officeLayout['Error triage'];add('triage','Error triage',1440+p.u-p.v,700+(p.u+p.v)/2+39)
  add('aws','AWS · us-west-2',470,410,false,'#e7c58e')
  add('public','Public subnets',1310,462,false,'#e7c58e')
  if(rooms.length)add('private','Private worker subnets',rooms[0].x,rooms[0].y+620,false,'#a2c5d1')
 }
 return <g className="map-label-overlay" pointerEvents="none" aria-hidden="true">
  {layoutMapLabels(labels).map(label=><g key={label.id}>
   {label.labelY!==label.y&&<path d={`M${label.x} ${label.y-8}V${label.labelY}`} stroke="#a8bdcc" strokeWidth="1" opacity=".6"/>}
   <rect x={label.x-label.width/2} y={label.labelY-label.height/2} width={label.width} height={label.height} rx="5" fill="#14273b" fillOpacity=".94"/>
   <text x={label.x} y={label.labelY} textAnchor="middle" dominantBaseline="central" style={{fontSize:label.heading?22:16,fontWeight:label.heading?650:500,fill:label.color||'#e4edf2'}}>{label.text}</text>
  </g>)}
 </g>
}
