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
 const surfaces=tiles.map(({u,v,room})=>({depth:project(u+.5,v+.5).y,order:0,element:<polygon key={'tile'+u+','+v} points={[[u,v],[u+1,v],[u+1,v+1],[u,v+1]].map(([a,b])=>{const p=project(a,b);return p.x+','+p.y}).join(' ')} fill={room?'#48657a':'#617e8d'} stroke="#78909c" strokeWidth=".5"/>}))
 for(const [i,{a,b,back}] of edges.entries()){
   const h=back?60:10
   surfaces.push({depth:(a.y+b.y)/2,order:1,element:<g key={'edge'+i}><polygon points={`${a.x},${a.y} ${b.x},${b.y} ${b.x},${b.y-h} ${a.x},${a.y-h}`} fill={back?'#52758b':'#304c61'} stroke="#708e9f" strokeWidth=".6"/><path d={`M${a.x} ${a.y-h}L${b.x} ${b.y-h}`} stroke="#b2c5ca" strokeWidth="2"/></g>})
 }
 return <g className="shared-floor" aria-hidden="true" pointerEvents="none">{surfaces.sort((a,b)=>a.depth-b.depth||a.order-b.order).map(s=>s.element)}</g>
}
