const TILE=20
export const project=(u:number,v:number)=>({x:720+(u-v)*TILE,y:110+(u+v)*TILE/2})
export const nodeOrigin=(index:number)=>{const row=Math.floor(index/2);return {u:(index%2)*24+row*24,v:22+row*24}}
export function floorPlan(count:number){
 const tiles=new Map<string,{u:number;v:number;room:boolean}>()
 const rect=(u:number,v:number,w:number,h:number,room=false)=>{for(let a=u;a<u+w;a++)for(let b=v;b<v+h;b++){const key=a+','+b;tiles.set(key,{u:a,v:b,room:room||!!tiles.get(key)?.room})}}
 for(const u of [0,15,30]){rect(u,0,9,9,true);rect(u+3,9,3,5)}
 rect(3,14,33,2)
 for(let i=0;i<count;i++){const {u,v}=nodeOrigin(i);rect(u,v,14,14,true);rect(u+3,v-6,3,6)}
 // Subsequent office pairs extend down the same two floor axes.
 for(let row=1;row<Math.ceil(count/2);row++){
  const shift=row*24
  rect(17+(row-1)*24,14+(row-1)*24,3,26)
  rect(17+(row-1)*24,14+shift,37,2)
 }
 const edges:{a:{x:number;y:number};b:{x:number;y:number};back:boolean}[]=[]
 for(const {u,v} of tiles.values()){
  for(const [du,dv,a,b,back] of [
   [-1,0,[u,v],[u,v+1],true],[0,-1,[u,v],[u+1,v],true],
   [1,0,[u+1,v],[u+1,v+1],false],[0,1,[u,v+1],[u+1,v+1],false],
  ] as [number,number,number[],number[],boolean][]){
   if(!tiles.has((u+du)+','+(v+dv)))edges.push({a:project(a[0],a[1]),b:project(b[0],b[1]),back})
  }
 }
 return {tiles:[...tiles.values()],edges}
}
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
