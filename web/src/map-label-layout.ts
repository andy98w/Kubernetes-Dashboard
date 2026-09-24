export type MapLabel={id:string;text:string;x:number;y:number;heading?:boolean;color?:string}
export function layoutMapLabels(labels:MapLabel[]){
 const placed:(MapLabel&{width:number;height:number;labelY:number})[]=[]
 for(const label of labels){
  const width=label.text.length*(label.heading?12:9)+18,height=label.heading?30:26
  let labelY=label.y
  for(let step=0;step<200;step++){
   const offset=step===0?0:Math.ceil(step/2)*30*(step%2?1:-1)
   labelY=label.y+offset
   if(!placed.some(p=>Math.abs(p.x-label.x)<(p.width+width)/2+6&&Math.abs(p.labelY-labelY)<(p.height+height)/2+5))break
  }
  placed.push({...label,width,height,labelY})
 }
 return placed
}
