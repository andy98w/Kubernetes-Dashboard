import {test} from 'node:test'
import assert from 'node:assert/strict'
import {usagePercent} from '../src/node-metrics.ts'
test('unknown capacity or usage never produces a healthy zero gauge',()=>{
 assert.equal(usagePercent(null,2000),null)
 assert.equal(usagePercent(0,undefined),null)
 assert.equal(usagePercent(NaN,2000),null)
 assert.equal(usagePercent(0,2000),0)
 assert.equal(usagePercent(500,2000),25)
 assert.equal(usagePercent(2500,2000),100)
})
