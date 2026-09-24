import {OfficeProp} from './office-props'
import {namespaceColor} from './station-topology'
import {serviceKey,serviceSlot,type MapService} from './service-layout'
import type {TopologyData} from './topology-data'
export function ServiceDesks({services,topology,selected,onSelect,onInspect,page,pages,onPage}:{services:MapService[];topology:TopologyData|null;selected:string|null;onSelect:(key:string)=>void;onInspect:(title:string,description:string)=>void;page:number;pages:number;onPage:(page:number)=>void}){
 return <g aria-label="Service routing desks">
  {services.map((service,index)=>{
   const key=serviceKey(service),position=serviceSlot(index),color=namespaceColor(service.namespace)
   const endpoints=topology?.endpoints.filter(e=>e.namespace===service.namespace&&e.service===service.name)||[]
   const ready=endpoints.filter(e=>e.ready===true).length
   const inspect=()=>onInspect(key,`${service.type} · ${service.ports.join(', ')||'Ports unreported'} · ${ready}/${endpoints.length} ready EndpointSlice entries. ${topology?.routes.filter(r=>r.namespace===service.namespace&&r.service===service.name).map(r=>r.host+r.path).join(', ')||'No loaded Ingress route'}. Select to trace backend membership; arrows are not measured requests.`)
   return <g key={key} transform={`translate(${position.x} ${position.y})`} data-object="service-desk" className="station-object" role="button" tabIndex={0} aria-label={`Trace Service ${key}`} aria-pressed={selected===key} opacity={selected&&selected!==key?.35:1} onMouseEnter={inspect} onFocus={inspect} onClick={()=>{inspect();onSelect(key)}} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();inspect();onSelect(key)}}}>
    <title>{key}</title><ellipse cy="12" rx="44" ry="22" fill="none" stroke={color} strokeWidth={selected===key?3:1} opacity=".8"/>
    <OfficeProp name="Service"/><circle cx="20" cy="-35" r="5" fill={color}/>
    <text className="map-label-source" y="44" textAnchor="middle" style={{fontSize:10,fill:'#e3e8e9'}}>{service.name.length>18?service.name.slice(0,16)+'…':service.name}</text>
   </g>
  })}
  {!services.length&&<text x="1020" y="355" textAnchor="middle">Service inventory unavailable</text>}
  {pages>1&&<g data-object="service-pages" transform="translate(1040 610)">{[-1,1].map(direction=><g key={direction} role="button" tabIndex={0} className="station-object" aria-label={direction<0?'Previous Service desks':'Next Service desks'} transform={`translate(${direction*65} 0)`} onClick={()=>onPage((page+direction+pages)%pages)} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();onPage((page+direction+pages)%pages)}}}><rect x="-18" y="-15" width="36" height="24" rx="8" fill="#243e53"/><text textAnchor="middle">{direction<0?'‹':'›'}</text></g>)}<text textAnchor="middle">{page+1} / {pages}</text></g>}
 </g>
}
