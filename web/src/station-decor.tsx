import {nodeOrigin,project} from './station-floor'

// Decorative furniture uses the same floor coordinates as the building.
// These props are not resources, status lights, or interactive controls.
const p=(u:number,v:number,z=0)=>`${(u-v)*20},${(u+v)*10-z}`
function Storage(){
 return <g stroke="#2d4557" strokeWidth=".8" strokeLinejoin="round">
  <polygon points={[p(0,0),p(1,0),p(1,2),p(0,2)].join(' ')} fill="#132c3c" opacity=".35" transform="translate(3 3)"/>
  <polygon points={[p(0,0),p(0,2),p(0,2,34),p(0,0,34)].join(' ')} fill="#769393"/>
  <polygon points={[p(0,2),p(1,2),p(1,2,34),p(0,2,34)].join(' ')} fill="#8aa8a3"/>
  <polygon points={[p(1,0),p(1,2),p(1,2,34),p(1,0,34)].join(' ')} fill="#587582"/>
  <polygon points={[p(0,0,34),p(1,0,34),p(1,2,34),p(0,2,34)].join(' ')} fill="#b5c3b7"/>
  {[9,20,31].map(z=><g key={z}><path d={`M${p(1,0.12,z)}L${p(1,1.88,z)}`} stroke="#344f62"/><path d={`M${p(1,.7,z-4)}L${p(1,1.2,z-4)}`} stroke="#e0d8b9" strokeWidth="2"/></g>)}
  <path d={`M${p(.15,.2,35)}L${p(.85,.2,35)}L${p(.85,.8,35)}L${p(.15,.8,35)}Z`} fill="#d9c39c"/>
  <path d={`M${p(.2,.22,38)}L${p(.8,.22,38)}L${p(.8,.75,38)}L${p(.2,.75,38)}Z`} fill="#9fb8c3"/>
 </g>
}
function Plant(){
 return <g>
  <ellipse cy="2" rx="12" ry="5" fill="#152c3b" opacity=".3"/>
  <path d="M-8-13-6 0Q0 5 6 0l2-13Z" fill="#ba9378" stroke="#3d5361"/>
  <ellipse cy="-13" rx="8" ry="4" fill="#d4b391"/><ellipse cy="-13" rx="6" ry="2.5" fill="#4a514b"/>
  <path d="M0-12v-23m0 15-8-9m8 3 8-9" stroke="#7caa92" strokeWidth="2"/>
  <path d="M0-27Q-12-27-10-37Q0-37 0-27M1-23Q3-35 12-32Q14-23 1-23M0-32Q-5-45 3-45Q10-37 0-32" fill="#8cb79d" stroke="#466f65" strokeWidth=".7"/>
 </g>
}
export function NodeDecor({index}:{index:number}){
 const origin=nodeOrigin(index),base=project(origin.u,origin.v)
 const at=(u:number,v:number)=>{const point=project(origin.u+u,origin.v+v);return `translate(${point.x-base.x} ${point.y-base.y})`}
 return <g aria-hidden="true" pointerEvents="none" className="station-decor">
  <g transform={at(.65,10.3)}><Storage/></g>
  <g transform={at(12.6,1.3)}><Plant/></g>
  <g transform={at(12.4,12.4)}><Plant/></g>
 </g>
}
