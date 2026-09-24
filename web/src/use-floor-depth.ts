import {useEffect,type RefObject} from 'react'
import {floorDepth} from './floor-depth'

export function useFloorDepth(root:RefObject<SVGSVGElement|null>){
 useEffect(()=>{
  // Read all positions before changing painter order. CSS remains responsible
  // for movement, pause, hover and reduced motion; no second animation clock.
  const sort=()=>{
   const plans=Array.from(root.current?.querySelectorAll('.room-actors')??[]).map(layer=>{
    const current=Array.from(layer.children) as SVGElement[]
    const sorted=current.map((element,index)=>{
     const walker=element.querySelector('.station-walker')
     const transform=walker?getComputedStyle(walker).transform:'none'
     const y=transform==='none'?0:new DOMMatrixReadOnly(transform).m42
     return {element,index,depth:floorDepth(Number(element.dataset.floorDepth),y)}
    }).sort((a,b)=>a.depth-b.depth||a.index-b.index)
    return {layer,current,sorted}
   })
   for(const {layer,current,sorted} of plans){
    if(sorted.some((item,index)=>item.element!==current[index])){
     // Moving SVG nodes can restart CSS animations in older browsers.
     // Preserve progress so depth changes never send a robot back to its desk.
     const clocks=sorted.map(({element})=>({element,times:element.getAnimations({subtree:true}).map(a=>a.currentTime)}))
     for(let i=0;i<sorted.length;i++){
      const element=sorted[i].element
      if(layer.children[i]!==element)layer.insertBefore(element,layer.children[i]??null)
     }
     for(const {element,times} of clocks)element.getAnimations({subtree:true}).forEach((animation,i)=>{
      if(times[i]!=null)animation.currentTime=times[i]
     })
    }
   }
  }
  sort()
  const timer=window.setInterval(sort,80)
  return()=>window.clearInterval(timer)
 },[root])
}
