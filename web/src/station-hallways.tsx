/** Architectural navigation metaphor, never a measured network connection. */
export function Hallways({rooms}:{rooms:{x:number;y:number;index:number;size:number}[]}){
 // The floor projection is (u-v, (u+v)/2): every run must have
 // slope +1/2 or -1/2, including the turns and room approaches.
 const diagonal=(ax:number,ay:number,bx:number,by:number,side:number)=>{
  const turnX=(ax+bx)/2+side*(by-ay)
  const turnY=ay+side*(turnX-ax)/2
  return `M${ax} ${ay} L${turnX} ${turnY} L${bx} ${by}`
 }
 const paths=[
  'M270 260 L600 425',
  'M930 260 L600 425',
  diagonal(600,305,600,425,1),
  ...rooms.map(r=>diagonal(600,425,
   r.x+(r.index%2?-1:1)*77.5*r.size*1.45,
   r.y+36*r.size*1.45,r.index%2?-1:1)),
 ]
 // Separate side faces keep the passage open at doorways and intersections.
 // Walls rise vertically from the floor edges, not from the centerline.
 const walls=paths.flatMap(d=>{
  const values=d.match(/-?\d+(?:\.\d+)?/g)!.map(Number)
  const faces:{points:string;cap:string;front:boolean;depth:number}[]=[]
  for(let i=0;i<values.length-2;i+=2){
   const ax=values[i],ay=values[i+1],bx=values[i+2],by=values[i+3]
   const length=Math.hypot(bx-ax,by-ay);if(length<48)continue
   const ux=(bx-ax)/length,uy=(by-ay)/length
   for(const side of [-1,1]){
    const nx=-uy*side,ny=ux*side,front=ny>0,h=front?18:38
    const x1=ax+ux*22+nx*19,y1=ay+uy*22+ny*19
    const x2=bx-ux*22+nx*19,y2=by-uy*22+ny*19
    faces.push({points:`${x1},${y1} ${x2},${y2} ${x2},${y2-h} ${x1},${y1-h}`,cap:`M${x1} ${y1-h} L${x2} ${y2-h}`,front,depth:(y1+y2)/2})
   }
  }
  return faces
 }).sort((a,b)=>a.depth-b.depth)
 return <g className="station-hallways" aria-hidden="true" pointerEvents="none" fill="none" strokeLinejoin="round">
  {paths.map((d,i)=><path key={'depth'+i} d={d} transform="translate(0 12)" stroke="#182d40" strokeWidth="42"/>)}
  {paths.map((d,i)=><path key={'edge'+i} d={d} stroke="#9cafb4" strokeWidth="42"/>)}
  {paths.map((d,i)=><path key={'floor'+i} d={d} stroke="#526f80" strokeWidth="36"/>)}
  {paths.map((d,i)=><path key={'tile'+i} d={d} stroke="#718a95" strokeWidth="30" strokeDasharray="1 18"/>)}
  {walls.map((wall,i)=><g key={'wall'+i} className="hallway-wall">
   <polygon points={wall.points} fill={wall.front?'#3e6075':'#628596'} stroke="#8ea7b3" strokeWidth="1"/>
   <path d={wall.cap} stroke="#bccdcb" strokeWidth="3"/>
  </g>)}
 </g>
}
export function Doorway({flip=false}:{flip?:boolean}){
 return <g transform={flip?'scale(-1 1)':undefined} aria-hidden="true" pointerEvents="none">
  <path d="m-15-7 30 15v5l-30-15Z" fill="#c9c7b2"/>
  <path d="M-15-7v-34l30 15V8" fill="none" stroke="#a9c6cc" strokeWidth="3"/>
  <path d="m-11-32 22 11" stroke="#a6d9c1" strokeWidth="2"/>
 </g>
}
