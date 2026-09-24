import {test} from 'node:test'
import assert from 'node:assert/strict'
import {floorDepth} from '../src/floor-depth.ts'
test('walking changes painter order relative to own and neighboring desks',()=>{
 const desk=floorDepth(70),neighbor=floorDepth(102)
 assert.ok(floorDepth(58,0)<desk)
 assert.ok(floorDepth(58,20)>desk)
 assert.ok(floorDepth(58,60)>neighbor)
 assert.ok(floorDepth(58,-20)<desk)
})
