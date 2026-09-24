import {test} from 'node:test'
import assert from 'node:assert/strict'
import {serviceCatalog,serviceKey,serviceSlot} from '../src/service-layout.ts'
test('services without backends remain visible and names stay namespace scoped',()=>{
 const services=serviceCatalog({services:[{namespace:'b',name:'api',type:'ClusterIP',clusterIp:'None',ports:['80/TCP']}],routes:[{namespace:'a',service:'api'}],endpoints:[{namespace:'a',service:'api'}]})
 assert.deepEqual(services.map(serviceKey),['a/api','b/api'])
 assert.equal(services[1].clusterIp,'None')
 assert.equal(services[0].type,'Unreported')
 assert.deepEqual(serviceCatalog(null),[])
})
test('all nine desk anchors fit the expanded service floor with furniture clearance',()=>{
 for(let i=0;i<9;i++){
  const p=serviceSlot(i),dx=p.x-1020,dy=p.y-260
  const u=dy+dx/2,v=dy-dx/2
  assert.ok(u>=30&&u<=300&&v>=25&&v<=280)
 }
})
