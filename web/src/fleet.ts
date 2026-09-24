// Proposed placement only. No production inventory is inferred from this fixture.
export const fleetRequested = new URLSearchParams(window.location.search).get('fleet') === '1'
export const fleetApps = [
  {namespace:'illuma',name:'illuma-web',replicas:2,port:'3000'},
  {namespace:'illuma',name:'illuma-api',replicas:2,port:'8000'},
  {namespace:'ledgly',name:'ledgly-web',replicas:2,port:'3000'},
  {namespace:'ledgly',name:'ledgly-api',replicas:2,port:'3001'},
  {namespace:'club-os',name:'club-web',replicas:1,port:'3000'},
  {namespace:'club-os',name:'club-quant-worker',replicas:1,port:''},
  {namespace:'club-os',name:'club-email-worker',replicas:1,port:''},
  {namespace:'portfolio',name:'portfolio-web',replicas:2,port:'8080'},
]
export const fleetNodes=['demo-worker-1','demo-worker-2','demo-worker-3','demo-worker-4']
export function fleetNode(name:string,index:number){
  if(name.startsWith('club-'))return fleetNodes[3] // Shared SQLite: intentionally co-located, not HA.
  if(name==='otel-agent')return fleetNodes[index%4]
  const app=fleetApps.findIndex(a=>a.name===name)
  const seed=app<0?[...name].reduce((sum,c)=>sum+c.charCodeAt(0),0):app
  return fleetNodes[(seed+index)%3]
}
export const fleetDependencies=[
  {id:'illuma-db',label:'Illuma PostgreSQL',namespace:'illuma',source:'illuma-api',detail:'Proposed managed PostgreSQL outside EKS. Existing provider can remain during migration.'},
  {id:'ledgly-db',label:'Ledgly PostgreSQL',namespace:'ledgly',source:'ledgly-api',detail:'Proposed managed PostgreSQL outside EKS. Backups and restore testing remain required.'},
  {id:'models',label:'Model APIs',namespace:'illuma',source:'illuma-api',detail:'Proposed outbound HTTPS to external model providers. No calls or latency measured.'},
  {id:'gmail',label:'Gmail API',namespace:'ledgly',source:'ledgly-api',detail:'Proposed outbound Gmail integration. Credentials remain server-side.'},
  {id:'illuma-storage',label:'Illuma objects',namespace:'illuma',source:'illuma-api',detail:'Planned object storage for Illuma. Provider, bucket, authentication, and direct-upload versus API-proxy flow are not verified in the local checkout. This outbound API path is a design placeholder, not measured usage; no S3 endpoint is provisioned here.'},
  {id:'email',label:'Email provider',namespace:'club-os',source:'club-email-worker',detail:'Proposed outbound email delivery. Provider and delivery guarantees must be verified before migration.'},
]
