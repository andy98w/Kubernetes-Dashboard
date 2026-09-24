const TILE=20
export const project=(u:number,v:number)=>({x:720+(u-v)*TILE,y:110+(u+v)*TILE/2})
export const nodeOrigin=(index:number)=>{const row=Math.floor(index/4),column=index%4;return {u:-24+column*28+row*28,v:26+row*28}}
export function floorPlan(count:number){
 const tiles=new Map<string,{u:number;v:number;room:boolean}>()
 const rect=(u:number,v:number,w:number,h:number,room=false)=>{for(let a=u;a<u+w;a++)for(let b=v;b<v+h;b++){const key=a+','+b;tiles.set(key,{u:a,v:b,room:room||!!tiles.get(key)?.room})}}
 rect(0,0,9,9,true);rect(3,9,3,9)
 rect(15,0,17,16,true);rect(18,16,3,2)
 // Telemetry houses eight tools: a larger floor, not oversized furniture.
 rect(36,0,13,12,true);rect(39,12,3,6)
 rect(3,18,39,3)
 for(let i=0;i<count;i++){const {u,v}=nodeOrigin(i);rect(u,v,18,18,true);rect(u+3,v-6,3,6)}
 // One straight, three-tile hallway per wing with short doorway spurs.
 for(let row=0;row<Math.ceil(count/4);row++){
  const first=nodeOrigin(row*4)
  const last=nodeOrigin(Math.min(count-1,row*4+3))
  const left=row===0?first.u+3:first.u-7
  const right=Math.max(row===0?42:left,last.u+6)
  rect(left,first.v-8,right-left,3)
  // Join additional wings through a gap between the preceding wing's rooms.
  if(row>0)rect(left,first.v-36,3,31)
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
