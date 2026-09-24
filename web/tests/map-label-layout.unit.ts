import {test} from 'node:test'
import assert from 'node:assert/strict'
import {layoutMapLabels} from '../src/map-label-layout.ts'
test('crowded labels separate without losing their original anchors',()=>{
 const input=Array.from({length:30},(_,i)=>({id:String(i),text:'Service label '+i,x:(i%3)*80,y:Math.floor(i/3)*35}))
 const result=layoutMapLabels(input)
 assert.equal(result.length,input.length)
 for(let i=0;i<result.length;i++){
  const a=result[i]
  assert.equal(a.y,input[i].y)
  for(const b of result.slice(i+1))assert.ok(Math.abs(a.x-b.x)>=(a.width+b.width)/2+6||Math.abs(a.labelY-b.labelY)>=(a.height+b.height)/2+5)
 }
 assert.deepEqual(layoutMapLabels(input),result)
})
