import {test} from 'node:test'
import assert from 'node:assert/strict'
import {recordError,resolveIssue} from '../src/error-triage.ts'

test('groups across releases and reopens a resolved issue',()=>{
 const event={service:'web',release:'r1',type:'TypeError',frame:'app.ts:10',at:'2026-09-23T10:00:00Z',traceId:'trace1'}
 let issues=recordError([],event)
 issues=recordError(issues,{...event,release:'r2'})
 assert.equal(issues.length,1);assert.equal(issues[0].count,2)
 issues=resolveIssue(issues,issues[0].key)
 issues=recordError(issues,{...event,release:'r3'})
 assert.equal(issues[0].status,'Regressed');assert.equal(issues[0].count,3)
 assert.equal(recordError(issues,{...event,service:'api'}).length,2)
})
test('bounds in-memory issue groups',()=>{
 let issues=recordError([],{service:'web',release:'r1',type:'Error',frame:'0',at:'now',traceId:''})
 for(let i=1;i<110;i++)issues=recordError(issues,{service:'web',release:'r1',type:'Error',frame:String(i),at:'now',traceId:''})
 assert.equal(issues.length,100)
})
