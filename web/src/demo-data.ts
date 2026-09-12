const ago = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString()

const workloads = [
  {kind:'Deployment',namespace:'kubevista-lab',name:'probe-failure',ready:1,desired:1,status:'Healthy',createdAt:ago(10)},
  {kind:'Deployment',namespace:'kubevista',name:'kubevista-api',ready:2,desired:2,status:'Healthy',createdAt:ago(240)},
  {kind:'Deployment',namespace:'kubevista',name:'kubevista-web',ready:2,desired:2,status:'Healthy',createdAt:ago(240)},
  {kind:'StatefulSet',namespace:'observability',name:'prometheus',ready:1,desired:1,status:'Healthy',createdAt:ago(310)},
  {kind:'StatefulSet',namespace:'observability',name:'loki',ready:1,desired:1,status:'Healthy',createdAt:ago(310)},
  {kind:'StatefulSet',namespace:'observability',name:'tempo',ready:1,desired:1,status:'Healthy',createdAt:ago(310)},
  {kind:'DaemonSet',namespace:'observability',name:'otel-agent',ready:2,desired:2,status:'Healthy',createdAt:ago(295)},
  {kind:'Deployment',namespace:'argocd',name:'argocd-server',ready:1,desired:1,status:'Healthy',createdAt:ago(360)},
]

export const isDemoMode = import.meta.env.VITE_DATA_MODE === 'demo'

const scenarioEvidence:Record<string,{code:string;evidence:string;nextStep:string}>={
 probe:{code:'ProbeFailed',evidence:'Readiness probe returned HTTP 404 for /broken.',nextStep:'Compare the probe path with the previous revision. Restarting retains the broken path.'},
 crashloop:{code:'CrashLoopBackOff',evidence:'Container process exits with code 1 and is backing off.',nextStep:'Inspect previous logs and compare command/configuration changes.'},
 imagepull:{code:'ImagePullBackOff',evidence:'Registry returned manifest unknown for nginx:kubevista-nonexistent-image.',nextStep:'Correct the image reference or review the previous revision.'},
 oom:{code:'OOMKilled',evidence:'Container exceeded its 64Mi memory limit (exit code 137).',nextStep:'Compare memory usage and investigate the recent allocation change.'},
 scheduling:{code:'Unschedulable',evidence:'0/1 nodes available: insufficient cpu for a 1000 CPU request.',nextStep:'Compare resource requests with available node capacity.'},
}
let activeScenario:string|null=null
let labTimeline:Array<{source:string;detail:string;at:string}>=[]
let labGeneration=1
export function injectDemoIncident(scenario:string){
 if(!scenarioEvidence[scenario])return
 activeScenario=scenario;labGeneration++
 Object.assign(workloads[0],{ready:0,status:'Progressing'})
 labTimeline=[{source:'Simulated rollout',detail:`Revision ${labGeneration} introduced ${scenario} failure.`,at:new Date().toISOString()},{source:'Simulated Kubernetes evidence',detail:scenarioEvidence[scenario].evidence,at:new Date().toISOString()}]
}
export function recoverDemoIncident(action:string,target:string){
 if(!target.endsWith('/Deployment/probe-failure'))return
 labTimeline.push({source:'Simulated operator action',detail:`${action} accepted; this is a browser simulation.`,at:new Date().toISOString()})
 if(action==='rollback'){
  activeScenario=null;labGeneration++
  Object.assign(workloads[0],{ready:1,status:'Healthy'})
  labTimeline.push({source:'Simulated recovery',detail:'Previous template restored; 1/1 updated and available.',at:new Date().toISOString()})
 }
}

export function demoData(view: string): unknown {
  const observedAt = new Date().toISOString()
  const records: Record<string,unknown> = {
    overview:{cluster:'kubevista-dev',mode:'demo',nodes:{ready:2,total:2},namespaces:9,pods:{running:51,pending:0,failed:0,succeeded:2,unknown:0},observedAt},
    workloads:{items:workloads,observedAt},
    network:{services:[{namespace:'kubevista',name:'kubevista-api',type:'ClusterIP',clusterIp:'172.20.4.18',ports:['80/TCP']},{namespace:'kubevista',name:'kubevista-web',type:'ClusterIP',clusterIp:'172.20.8.42',ports:['80/TCP']},{namespace:'observability',name:'prometheus',type:'ClusterIP',clusterIp:'172.20.12.9',ports:['9090/TCP']}],ingresses:[{namespace:'kubevista',name:'kubevista',class:'alb',hosts:['kubevista.illuma.me'],address:'k8s-kubevist-5e18.us-west-2.elb.amazonaws.com'}],policies:[{namespace:'kubevista',name:'kubevista-api',ingressRules:3,egressRules:3},{namespace:'kubevista',name:'kubevista-web',ingressRules:2,egressRules:1},{namespace:'observability',name:'default-deny',ingressRules:0,egressRules:0}],observedAt},
    events:{items:[{type:'Normal',reason:'ScalingReplicaSet',namespace:'kubevista',object:'Deployment/kubevista-api',message:'Scaled up replica set kubevista-api-7d8c9b to 2.',count:1,lastSeen:ago(22)},{type:'Normal',reason:'SuccessfulCreate',namespace:'kubevista',object:'ReplicaSet/kubevista-api-7d8c9b',message:'Created pod kubevista-api-7d8c9b-x2m4p.',count:2,lastSeen:ago(21)},{type:'Normal',reason:'DisruptionTestComplete',namespace:'kubevista',object:'Deployment/kubevista-api',message:'Controlled pod-loss test completed with 896/896 HTTP 200 responses.',count:1,lastSeen:ago(18)}],observedAt},
    observability:{components:[{kind:'StatefulSet',name:'prometheus',ready:1,desired:1,status:'Healthy'},{kind:'StatefulSet',name:'loki',ready:1,desired:1,status:'Healthy'},{kind:'StatefulSet',name:'tempo',ready:1,desired:1,status:'Healthy'},{kind:'Deployment',name:'otel-gateway',ready:2,desired:2,status:'Healthy'},{kind:'Deployment',name:'grafana',ready:1,desired:1,status:'Healthy'}],signals:['Metrics / Prometheus','Logs / Loki','Traces / Tempo','Telemetry / OpenTelemetry'],namespace:'observability',observedAt},
    security:{findings:[{severity:'Warning',category:'Image policy',namespace:'demo',resource:'Pod/legacy-worker:worker',message:'Container references a mutable image tag; production workloads use immutable ECR digests.'}],podsEvaluated:51,networkPolicies:8,observedAt},
    cost:{nodes:[{name:'ip-10-0-11-42',instanceType:'t3.medium',capacityType:'SPOT',estimatedHourly:0.0146},{name:'ip-10-0-21-18',instanceType:'t3.medium',capacityType:'SPOT',estimatedHourly:0.0146}],controlPlaneHourly:0.10,loadBalancerHourly:0.0225,natGatewayHourly:0.045,estimatedHourly:0.1967,currency:'USD',disclaimer:'Estimate from the August 31 deployment. It excludes storage, data processing, logs, taxes, discounts, and free-tier credits.',observedAt},
    incidents:{items:[{id:'test-2026-08-31-api-loss',severity:'Controlled',status:'Resolved',title:'Controlled API pod-loss recovery',summary:'A running API pod was removed while Fortio generated traffic to validate disruption tolerance.',namespace:'kubevista',resource:'Deployment/kubevista-api',startedAt:ago(47),evidence:[{source:'Load generator',detail:'Fortio maintained 20 requests per second during the test',at:ago(47)},{source:'Kubernetes controller',detail:'Replacement API pod became Ready in 2 seconds',at:ago(46)},{source:'Verification',detail:'896/896 requests returned HTTP 200; p99 was approximately 2.78 ms',at:ago(43)}]}],observedAt},
    settings:{cluster:'kubevista-dev',environment:'demo data',version:'0.3.0',readOnly:true,operationsMode:'simulation',operationNamespaces:['kubevista','kubevista-lab'],minReplicas:1,maxReplicas:6,refreshSeconds:0},
  }
  return records[view]
}

export function demoWorkloadDetail(item: typeof workloads[number],enrich=false) {
  const observedAt=new Date().toISOString()
  const lab=item.name==='probe-failure'
  const actual=lab?workloads[0]:item
  if(lab) return {
    workload:{...actual},strategy:'RollingUpdate',selector:{app:'probe-failure'},labels:{app:'probe-failure'},
    images:[{name:'web',image:activeScenario==='imagepull'?'nginx:kubevista-nonexistent-image':'nginx:1.28-alpine'}],
    pods:[{name:'probe-failure-simulated',phase:activeScenario==='scheduling'?'Pending':'Running',ready:actual.ready,containers:1,restarts:activeScenario==='crashloop'||activeScenario==='oom'?3:0,node:activeScenario==='scheduling'?'':'simulated-node',createdAt:ago(1)}],
    services:[],policies:[],events:[],
    diagnoses:activeScenario?[{...scenarioEvidence[activeScenario],resource:'Pod/probe-failure-simulated'}]:[],
    timeline:[...labTimeline],
    recovery:{status:activeScenario?'Progressing':'Recovered',detail:`${actual.ready}/1 available (simulated).`,generation:labGeneration,observedGeneration:labGeneration},
    warnings:enrich?['Synthetic log excerpt for the browser lab. No Prometheus server was queried.']:[],
    logs:enrich&&activeScenario?[{pod:'probe-failure-simulated',container:'web',previous:false,text:'[SIMULATED] '+scenarioEvidence[activeScenario].evidence}]:[],
    metrics:[],observedAt,
  }
  return {diagnoses:lab&&activeScenario?[{...scenarioEvidence[activeScenario],resource:'Pod/probe-failure-simulated'}]:[],timeline:lab?labTimeline:[],warnings:[],logs:[],metrics:[],recovery:{status:actual.ready===actual.desired?'Recovered':'Progressing',detail:`${actual.ready}/${actual.desired} available (simulated).`,generation:labGeneration,observedGeneration:labGeneration},workload:actual,strategy:item.kind==='StatefulSet'?'RollingUpdate':'RollingUpdate',selector:{'app.kubernetes.io/name':item.name},labels:{'app.kubernetes.io/name':item.name,'app.kubernetes.io/managed-by':'argocd'},images:[{name:'app',image:`public.ecr.aws/kubevista/${item.name}:sha-8f04c2a`}],pods:Array.from({length:item.ready},(_,index)=>({name:`${item.name}-7d8c9b-${index?'z9q7n':'x2m4p'}`,phase:'Running',ready:1,containers:1,restarts:0,node:index?'ip-10-0-21-18':'ip-10-0-11-42',createdAt:ago(180)})),services:item.name.startsWith('kubevista')?[{namespace:item.namespace,name:item.name,type:'ClusterIP',clusterIp:'172.20.4.18',ports:['80/TCP']}]:[],policies:[{namespace:item.namespace,name:`${item.name}-access`,ingressRules:2,egressRules:1}],events:[{type:'Normal',reason:'ScalingReplicaSet',namespace:item.namespace,object:`${item.kind}/${item.name}`,message:`Reconciled ${item.desired} desired replicas.`,count:1,lastSeen:ago(22)}],observedAt}
}
