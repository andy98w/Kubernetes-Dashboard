import {test} from 'node:test'
import assert from 'node:assert/strict'
import {breakRoute} from '../src/robot-routine.ts'
test('break waypoints remain within each room for all workstation slots',()=>{
 for(let slot=0;slot<9;slot++)for(let seed=0;seed<30;seed++){
  const u=58+(slot%3)*64,v=58+Math.floor(slot/3)*64
  for(const {x,y} of Object.values(breakRoute(slot,seed))){
   const a=u+y+x/2,b=v+y-x/2
   assert.ok(a>=20&&a<=215&&b>=20&&b<=215,`${slot}: ${a},${b}`)
  }
 }
})
