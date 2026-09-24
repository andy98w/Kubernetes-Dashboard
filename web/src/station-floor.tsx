import {project,floorPlan} from './station-geometry'
export {project,nodeOrigin,floorPlan} from './station-geometry'
export function StationFloor({count}:{count:number}){
 const {tiles,edges}=floorPlan(count)
 const occupied=new Set(tiles.map(t=>`${t.u},${t.v}`))
 const corners=(u:number,v:number,inset=0)=>[[u+inset,v+inset],[u+1-inset,v+inset],[u+1-inset,v+1-inset],[u+inset,v+1-inset]].map(([a,b])=>{const p=project(a,b);return `${p.x},${p.y}`}).join(' ')
 const surfaces=tiles.map(({u,v,room})=>{
  const perimeter=[[-1,0],[1,0],[0,-1],[0,1]].some(([a,b])=>!occupied.has(`${u+a},${v+b}`))
  const shade=(u*7+v*13)%11
  const center=project(u+.5,v+.5)
  return {depth:center.y,order:0,element:<g key={'tile'+u+','+v}>
   <polygon points={corners(u,v)} fill={perimeter?'#536e7d':room?(shade===0?'#4c697b':'#435f72'):'#607d87'} stroke="#78909c" strokeWidth=".45"/>
   {!room&&!perimeter&&<polygon points={corners(u,v,.12)} fill="#3a566a" opacity=".55"/>}
   {room&&shade===0&&!perimeter&&<path d={`M${center.x-5} ${center.y-1}l4 2m2-3 4 2`} stroke="#8ea3ad" strokeWidth=".6" opacity=".5"/>}
  </g>}
 })
 for(const [i,{a,b,back}] of edges.entries()){
   const h=back?60:10
   const dx=b.x-a.x,dy=b.y-a.y
   surfaces.push({depth:(a.y+b.y)/2,order:1,element:<g key={'edge'+i}>
    <polygon points={`${a.x},${a.y} ${b.x},${b.y} ${b.x},${b.y-h} ${a.x},${a.y-h}`} fill={back?(dy>0?'#48687e':'#52758b'):'#304c61'} stroke="#708e9f" strokeWidth=".6"/>
    <path d={`M${a.x} ${a.y-h}L${b.x} ${b.y-h}`} stroke="#b2c5ca" strokeWidth="2"/>
    {back&&<g>
     <path d={`M${a.x} ${a.y-5}l${dx} ${dy}m${-dx} ${-dy-9}l${dx} ${dy}`} stroke="#2d495c" strokeWidth="2"/>
     <path d={`M${a.x} ${a.y-47}l${dx} ${dy}`} stroke="#8299a3" strokeWidth=".8" opacity=".6"/>
     {i%9===0&&<g transform={`matrix(${dx/20} ${dy/20} 0 1 ${a.x} ${a.y})`}>
      <rect x="3" y="-40" width="14" height="15" rx="1" fill="#314b60" stroke="#90a5ad" strokeWidth=".7"/>
      {[-36,-33,-30].map(y=><path key={y} d={`M5 ${y}h10`} stroke="#76919e" strokeWidth="1"/>)}
     </g>}
     {i%17===5&&<g transform={`matrix(${dx/20} ${dy/20} 0 1 ${a.x} ${a.y})`}>
      <rect x="3" y="-43" width="14" height="19" fill="#d5c7a5"/>
      <rect x="5" y="-41" width="10" height="15" fill="#304e66"/>
      <circle cx="10" cy="-36" r="2" fill="#ebbb7e"/>
      <path d="m5-27 4-6 3 3 3-2v5Z" fill="#91bbad"/>
     </g>}
    </g>}
   </g>})
 }
 return <g className="shared-floor" aria-hidden="true" pointerEvents="none">{surfaces.sort((a,b)=>a.depth-b.depth||a.order-b.order).map(s=>s.element)}</g>
}
