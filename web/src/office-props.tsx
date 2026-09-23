/** Symbolic office furniture, not physical infrastructure or replica counts.
 * Every vertex shares the room's ground projection; z=0 is the floor. */
const point=(u:number,v:number,z=0)=>`${u-v},${(u+v)/2-z}`
function Box({u=0,v=0,w=32,d=20,z=0,h=32,color='#91adb7'}:{u?:number;v?:number;w?:number;d?:number;z?:number;h?:number;color?:string}){
 return <g stroke="#23394d" strokeWidth=".65" strokeLinejoin="round">
  <polygon points={[point(u,v+d,z),point(u+w,v+d,z),point(u+w,v+d,z+h),point(u,v+d,z+h)].join(' ')} fill={color}/>
  <polygon points={[point(u+w,v,z),point(u+w,v+d,z),point(u+w,v+d,z+h),point(u+w,v,z+h)].join(' ')} fill={color} style={{filter:'brightness(.72)'}}/>
  <polygon points={[point(u,v,z+h),point(u+w,v,z+h),point(u+w,v+d,z+h),point(u,v+d,z+h)].join(' ')} fill={color} style={{filter:'brightness(1.16)'}}/>
 </g>
}
export const officeLayout:Record<string,{u:number;v:number}>={
 'API server':{u:40,v:30},Scheduler:{u:120,v:24},Controllers:{u:35,v:112},etcd:{u:145,v:98},
 Service:{u:45,v:32},EndpointSlice:{u:130,v:28},'Pod backend':{u:130,v:120},
 Prometheus:{u:28,v:24},Grafana:{u:87,v:24},Alertmanager:{u:146,v:24},
 Loki:{u:25,v:84},Tempo:{u:148,v:84},'OTel gateway':{u:25,v:140},'OTel agent':{u:148,v:140},
}
export function OfficeProp({name}:{name:string}){
 const desk=['API server','Grafana','Controllers','Service'].includes(name)
 const color=name==='etcd'?'#b5abc8':name==='Loki'?'#95b5b5':name==='Alertmanager'?'#d6a271':'#91adb7'
 return <g strokeLinejoin="round">
  <title>{name}</title>
  <polygon points={desk?'-26,14 16,-7 51,12 9,33':'-21,11 11,-5 38,9 6,25'} fill="#071725" opacity=".26"/>
  {desk?<g>
   {[[-19,0],[19,0],[-19,21],[19,21]].map(([u,v])=><Box key={u+','+v} u={u} v={v} w={3} d={3} h={27} color="#607887"/>)}
   <Box u={-22} w={47} d={27} z={27} h={3} color="#d7b88c"/>
   <Box u={-3} v={5} w={5} d={4} z={30} h={7} color="#526a77"/>
   <Box u={-15} v={3} w={30} d={3} z={36} h={18} color="#23394d"/>
   <path d="m-16-32 6-3 5 6 6-2 6 0" fill="none" stroke="#a6d9c1" strokeWidth="1.6" className="office-indicator"/>
   <Box u={-10} v={15} w={21} d={7} z={30} h={1} color="#708792"/>
   <g transform="translate(-35 19)">
    <path d="M0 0v-13M-9-4 9 4M-9 4 9-4" stroke="#263c4b" strokeWidth="2"/>
    <Box u={-8} v={-8} w={16} d={16} z={12} h={4} color="#839c92"/>
    <Box u={-8} v={6} w={16} d={3} z={16} h={16} color="#839c92"/>
   </g>
  </g>:<g>
   <Box u={-15} w={30} d={21} h={3} color="#253b49"/>
   <Box u={-15} w={30} d={21} z={3} h={name==='Scheduler'?48:36} color={color}/>
   {[10,21,32].map(z=><g key={z}>
    <path d={`M${point(-12,21.5,z)} L${point(12,21.5,z)}`} stroke="#3c5363" strokeWidth="1.5"/>
    <path d={`M${point(-3,22,z+4)} L${point(3,22,z+4)}`} stroke="#e9ddc3" strokeWidth="2"/>
   </g>)}
   {name==='Alertmanager'&&<g><Box u={-4} v={5} w={8} d={8} z={39} h={7} color="#edb785"/><circle cx="-4" cy="-39" r="2" fill="#ffe3a3" className="office-indicator"/></g>}
   {name==='Prometheus'&&<g transform="matrix(1 .5 0 1 -22 -23)"><circle r="7" fill="#ede2cb"/><path d="M0 0 3-4" stroke="#bb7558" strokeWidth="2"/></g>}
  </g>}
 </g>
}
