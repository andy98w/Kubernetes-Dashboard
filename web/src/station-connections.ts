type Point={x:number;y:number}
type Tile={u:number;v:number}
const grid=(p:Point)=>({u:(p.y-110)/20+(p.x-720)/40,v:(p.y-110)/20-(p.x-720)/40})
const screen=(p:Tile)=>({x:720+(p.u-p.v)*20,y:110+(p.u+p.v)*10})
const key=(p:Tile)=>`${p.u},${p.v}`
const path=(points:Point[])=>points.map((p,i)=>`${i?'L':'M'}${p.x} ${p.y}`).join(' ')

// Both legs follow the same two axes as the room walls; no Bezier curves.
export function angledConnection(from:Point,to:Point){
 const a=grid(from),b=grid(to)
 return path([from,screen({u:b.u,v:a.v}),to])
}

// Route through the union of room/corridor tiles, so connections use openings.
export function floorConnection(from:Point,to:Point,tiles:Tile[]){
 const a=grid(from),b=grid(to)
 const start={u:Math.floor(a.u),v:Math.floor(a.v)},end={u:Math.floor(b.u),v:Math.floor(b.v)}
 const floor=new Set(tiles.map(key)),queue=[start],previous=new Map<string,Tile|null>([[key(start),null]])
 if(!floor.has(key(start))||!floor.has(key(end)))return ''
 for(let i=0;i<queue.length&&!previous.has(key(end));i++){
  const p=queue[i]
  for(const [du,dv] of [[0,1],[1,0],[0,-1],[-1,0]]){
   const next={u:p.u+du,v:p.v+dv},id=key(next)
   if(floor.has(id)&&!previous.has(id)){previous.set(id,p);queue.push(next)}
  }
 }
 if(!previous.has(key(end)))return ''
 const route:Tile[]=[]
 for(let p:Tile|null=end;p;p=previous.get(key(p))??null)route.push({u:p.u+.5,v:p.v+.5})
 route.reverse()
 const first=route[0],last=route[route.length-1]
 return path([from,screen({u:first.u,v:a.v}),...route.map(screen),screen({u:last.u,v:b.v}),to])
}
