const ago = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString()

const workloads = [
  {kind:'Deployment',namespace:'kubevista',name:'kubevista-api',ready:2,desired:2,status:'Healthy',createdAt:ago(240)},
  {kind:'Deployment',namespace:'kubevista',name:'kubevista-web',ready:2,desired:2,status:'Healthy',createdAt:ago(240)},
  {kind:'StatefulSet',namespace:'observability',name:'prometheus',ready:1,desired:1,status:'Healthy',createdAt:ago(310)},
  {kind:'StatefulSet',namespace:'observability',name:'loki',ready:1,desired:1,status:'Healthy',createdAt:ago(310)},
  {kind:'StatefulSet',namespace:'observability',name:'tempo',ready:1,desired:1,status:'Healthy',createdAt:ago(310)},
  {kind:'DaemonSet',namespace:'observability',name:'otel-agent',ready:2,desired:2,status:'Healthy',createdAt:ago(295)},
  {kind:'Deployment',namespace:'argocd',name:'argocd-server',ready:1,desired:1,status:'Healthy',createdAt:ago(360)},
]

export const isDemoMode = import.meta.env.VITE_DATA_MODE === 'demo'

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
    settings:{cluster:'kubevista-dev',environment:'demo data',version:'0.3.0',readOnly:true,operationsMode:'simulation',operationNamespaces:['kubevista'],minReplicas:1,maxReplicas:6,refreshSeconds:0},
  }
  return records[view]
}

export function demoWorkloadDetail(item: typeof workloads[number]) {
  const observedAt=new Date().toISOString()
  return {workload:item,strategy:item.kind==='StatefulSet'?'RollingUpdate':'RollingUpdate',selector:{'app.kubernetes.io/name':item.name},labels:{'app.kubernetes.io/name':item.name,'app.kubernetes.io/managed-by':'argocd'},images:[{name:'app',image:`public.ecr.aws/kubevista/${item.name}:sha-8f04c2a`}],pods:Array.from({length:item.ready},(_,index)=>({name:`${item.name}-7d8c9b-${index?'z9q7n':'x2m4p'}`,phase:'Running',ready:1,containers:1,restarts:0,node:index?'ip-10-0-21-18':'ip-10-0-11-42',createdAt:ago(180)})),services:item.name.startsWith('kubevista')?[{namespace:item.namespace,name:item.name,type:'ClusterIP',clusterIp:'172.20.4.18',ports:['80/TCP']}]:[],policies:[{namespace:item.namespace,name:`${item.name}-access`,ingressRules:2,egressRules:1}],events:[{type:'Normal',reason:'ScalingReplicaSet',namespace:item.namespace,object:`${item.kind}/${item.name}`,message:`Reconciled ${item.desired} desired replicas.`,count:1,lastSeen:ago(22)}],observedAt}
}
