import {test} from 'node:test'
import assert from 'node:assert/strict'
import {floorPlan,nodeOrigin,project} from '../src/station-geometry.ts'
import {floorConnection} from '../src/station-connections.ts'
test('expanded workers reserve floor space for the equipment perimeter',()=>{
 const {tiles}=floorPlan(4)
 for(let i=0;i<4;i++){
  const o=nodeOrigin(i)
  const room=tiles.filter(t=>t.room&&t.u>=o.u&&t.u<o.u+18&&t.v>=o.v&&t.v<o.v+18)
  assert.equal(room.length,324)
  // Equipment feet in screen coordinates relative to each room's floor origin.
  for(const [x,y] of [[260,186],[-235,205]]){
   const u=(y+x/2)/20,v=(y-x/2)/20
   assert.ok(u>0&&u<18&&v>0&&v<18)
  }
 }
})
test('four worker rooms spread sideways without changing projection or room size',()=>{
 const p=Array.from({length:4},(_,i)=>{const o=nodeOrigin(i);return project(o.u,o.v)})
 assert.ok(p[3].x-p[0].x>=1400)
 assert.equal(p[3].x-p[0].x,2*(p[3].y-p[0].y))
 for(let i=0;i<4;i++)for(let j=i+1;j<4;j++){
  const a=nodeOrigin(i),b=nodeOrigin(j)
  assert.ok(a.u+18<=b.u||b.u+18<=a.u||a.v+18<=b.v||b.v+18<=a.v)
 }
})
test('worker entrances share one straight three-tile hallway',()=>{
 const {tiles}=floorPlan(4),all=new Set(tiles.map(t=>`${t.u},${t.v}`))
 for(let i=0;i<4;i++)assert.equal(nodeOrigin(i).v,26)
 for(let u=-21;u<54;u++)for(let v=18;v<21;v++)assert.ok(all.has(`${u},${v}`))
})
test('corridors connect every room for full and incomplete worker wings',()=>{
 for(const count of [1,2,3,4,5,8]){
  const {tiles,edges}=floorPlan(count),all=new Set(tiles.map(t=>`${t.u},${t.v}`)),seen=new Set(['0,0']),queue=[[0,0]]
  for(let n=0;n<queue.length;n++){
   const [u,v]=queue[n]
   for(const [du,dv] of [[1,0],[-1,0],[0,1],[0,-1]]){
    const key=`${u+du},${v+dv}`
    if(all.has(key)&&!seen.has(key)){seen.add(key);queue.push([u+du,v+dv])}
   }
  }
  assert.equal(seen.size,all.size,`disconnected floor at ${count} nodes`)
  for(let i=0;i<count;i++){
   const origin=nodeOrigin(i),target=project(origin.u+4.5,origin.v+2.5)
   for(const source of [project(4.5,7.5),project(19.5,7.5),project(40.5,7.5)]){
    assert.ok(floorConnection(source,target,tiles),`missing facility route to node ${i}`)
   }
  }
  for(const {a,b} of edges)assert.equal(Math.abs(b.y-a.y)*2,Math.abs(b.x-a.x))
 }
})
