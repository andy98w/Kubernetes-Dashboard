import React, { useCallback, useEffect, useMemo, useState } from 'react'
import ReactDOM from 'react-dom/client'
import './styles.css'

type View = 'overview' | 'workloads' | 'network' | 'events' | 'observability' | 'security' | 'cost' | 'settings'
type RequestState = 'loading' | 'connected' | 'stale'
type IconName = View | 'refresh' | 'check' | 'warning' | 'node' | 'pod' | 'namespace' | 'clock'
type Summary = { cluster:string; mode:'demo'|'cluster'; nodes:{ready:number;total:number}; namespaces:number; pods:{running:number;pending:number;failed:number;succeeded:number;unknown:number}; observedAt:string }
type Workload = { kind:string; namespace:string; name:string; ready:number; desired:number; status:string; createdAt:string }
type Workloads = { items:Workload[]; observedAt:string }
type Service = { namespace:string; name:string; type:string; clusterIp:string; ports:string[] }
type Ingress = { namespace:string; name:string; class:string; hosts:string[]; address:string }
type Policy = { namespace:string; name:string; ingressRules:number; egressRules:number }
type Network = { services:Service[]; ingresses:Ingress[]; policies:Policy[]; observedAt:string }
type ClusterEvent = { type:string; reason:string; namespace:string; object:string; message:string; count:number; lastSeen:string }
type Events = { items:ClusterEvent[]; observedAt:string }
type Component = { kind:string; name:string; ready:number; desired:number; status:string }
type Observability = { components:Component[]; signals:string[]; namespace:string; observedAt:string }
type Finding = { severity:string; category:string; namespace:string; resource:string; message:string }
type Security = { findings:Finding[]; podsEvaluated:number; networkPolicies:number; observedAt:string }
type NodeCost = { name:string; instanceType:string; capacityType:string; estimatedHourly:number }
type Cost = { nodes:NodeCost[]; controlPlaneHourly:number; loadBalancerHourly:number; natGatewayHourly:number; estimatedHourly:number; currency:string; disclaimer:string; observedAt:string }
type Settings = { cluster:string; environment:string; version:string; readOnly:boolean; refreshSeconds:number }

const fallbackSummary: Summary = { cluster:'kubevista-dev',mode:'demo',nodes:{ready:0,total:0},namespaces:0,pods:{running:0,pending:0,failed:0,succeeded:0,unknown:0},observedAt:'' }
const navigation: Array<{label:string;items:Array<{icon:IconName;name:string;view:View}>}> = [
  {label:'Cluster',items:[{icon:'overview',name:'Overview',view:'overview'},{icon:'workloads',name:'Workloads',view:'workloads'},{icon:'network',name:'Network',view:'network'},{icon:'events',name:'Events',view:'events'}]},
  {label:'Platform',items:[{icon:'observability',name:'Observability',view:'observability'},{icon:'security',name:'Security',view:'security'},{icon:'cost',name:'Cost',view:'cost'}]},
]

function initialView(): View {
  const candidate = window.location.hash.replace('#/','') as View
  return [...navigation.flatMap(section=>section.items.map(item=>item.view)),'settings'].includes(candidate) ? candidate : 'overview'
}

function App() {
  const [view,setView] = useState<View>(initialView)
  const [cache,setCache] = useState<Partial<Record<View,unknown>>>({overview:fallbackSummary})
  const [requestState,setRequestState] = useState<RequestState>('loading')
  const [error,setError] = useState<string|null>(null)
  const [refreshing,setRefreshing] = useState(false)
  const refresh = useCallback(async (signal?:AbortSignal) => {
    setRefreshing(true)
    try {
      const response = await fetch(`/api/v1/${view === 'overview' ? 'summary' : view}`,{signal})
      if(!response.ok) throw new Error(`API returned ${response.status}`)
      const value=await response.json(); setCache(current=>({...current,[view]:value})); setRequestState('connected'); setError(null)
    } catch(caught) {
      if(caught instanceof DOMException && caught.name==='AbortError') return
      setRequestState('stale'); setError(caught instanceof Error?caught.message:'Cluster API unavailable')
    } finally { setRefreshing(false) }
  },[view])
  useEffect(()=>{
    const controller=new AbortController(); setRequestState('loading'); void refresh(controller.signal)
    const timer=window.setInterval(()=>void refresh(controller.signal),15_000)
    return()=>{controller.abort();window.clearInterval(timer)}
  },[refresh])
  useEffect(()=>{ const listener=()=>setView(initialView()); window.addEventListener('hashchange',listener); return()=>window.removeEventListener('hashchange',listener) },[])
  useEffect(()=>{ document.title=`KubeVista · ${view[0].toUpperCase()+view.slice(1)}` },[view])
  const navigate=(next:View)=>{ setRequestState('loading'); window.location.hash=`/${next}`; setView(next) }
  const activeName = view[0].toUpperCase()+view.slice(1)
  return <div className="app-shell">
    <aside className="sidebar">
      <button className="brand-mark" onClick={()=>navigate('overview')} aria-label="KubeVista home"><Logo/><div><strong>KubeVista</strong><span>Operations console</span></div></button>
      <div className="environment-switcher"><span className={`status-dot ${requestState}`}/><div><small>Environment</small><strong>Development</strong></div><span className="chevron">⌄</span></div>
      <nav aria-label="Primary navigation">{navigation.map(section=><div className="nav-section" key={section.label}><p>{section.label}</p>{section.items.map(item=><button className={view===item.view?'active':''} onClick={()=>navigate(item.view)} key={item.view}><Icon name={item.icon}/><span>{item.name}</span></button>)}</div>)}</nav>
      <div className="sidebar-footer"><button className={view==='settings'?'active':''} onClick={()=>navigate('settings')}><Icon name="settings"/><span>Settings</span></button><div className="version-row"><span>KV</span><div><strong>Portfolio build</strong><small>v0.2.0</small></div></div></div>
    </aside>
    <main>
      <header className="topbar"><div className="breadcrumbs"><span>Clusters</span><b>/</b><strong>kubevista-dev</strong><b>/</b><span>{activeName}</span></div><div className="topbar-actions"><span className={`connection-pill ${requestState}`}><span/>{requestState==='connected'?'API connected':requestState==='loading'?'Connecting':'API unavailable'}</span><button className="icon-button" onClick={()=>void refresh()} disabled={refreshing} aria-label="Refresh current view"><Icon name="refresh"/></button></div></header>
      <div className="workspace">{requestState==='stale'&&<div className="notice" role="status"><Icon name="warning"/><div><strong>Live inventory is unavailable</strong><span>{error}. Showing the last successful response where available.</span></div><button onClick={()=>void refresh()}>Retry connection</button></div>}{requestState==='loading'||!validData(view,cache[view])?<LoadingPage/>:<Page view={view} data={cache[view]}/>}</div>
    </main>
  </div>
}

function validData(view:View,data:unknown):boolean {
  if(!data||typeof data!=='object') return false
  const value=data as Record<string,unknown>
  if(view==='overview') return typeof value.cluster==='string'&&typeof value.pods==='object'
  if(view==='workloads'||view==='events') return Array.isArray(value.items)
  if(view==='network') return Array.isArray(value.services)&&Array.isArray(value.ingresses)&&Array.isArray(value.policies)
  if(view==='observability') return Array.isArray(value.components)&&Array.isArray(value.signals)
  if(view==='security') return Array.isArray(value.findings)
  if(view==='cost') return Array.isArray(value.nodes)&&typeof value.estimatedHourly==='number'
  return view==='settings'&&typeof value.cluster==='string'
}

function Page({view,data}:{view:View;data:unknown}) {
  switch(view){
    case 'overview': return <OverviewPage summary={data as Summary}/>
    case 'workloads': return <WorkloadsPage data={data as Workloads}/>
    case 'network': return <NetworkPage data={data as Network}/>
    case 'events': return <EventsPage data={data as Events}/>
    case 'observability': return <ObservabilityPage data={data as Observability}/>
    case 'security': return <SecurityPage data={data as Security}/>
    case 'cost': return <CostPage data={data as Cost}/>
    case 'settings': return <SettingsPage data={data as Settings}/>
  }
}

function PageHeading({eyebrow,title,description,observedAt}:{eyebrow:string;title:string;description:string;observedAt?:string}) {
  const observed=observedAt?new Date(observedAt):null
  return <section className="page-heading"><div><div className="eyebrow"><span>{eyebrow}</span><i/>us-west-2</div><h1>{title}</h1><p>{description}</p></div>{observed&&<div className="observation"><Icon name="clock"/><div><small>Last observed</small><strong>{observed.toLocaleTimeString([],{hour:'2-digit',minute:'2-digit',second:'2-digit'})}</strong></div></div>}</section>
}

function OverviewPage({summary}:{summary:Summary}) {
  const podTotal=Object.values(summary.pods).reduce((a,b)=>a+b,0), healthyPods=podTotal?Math.round(summary.pods.running/podTotal*100):0, nodeHealth=summary.nodes.total?Math.round(summary.nodes.ready/summary.nodes.total*100):0
  const hasRisk=summary.pods.failed>0||summary.nodes.ready<summary.nodes.total
  return <><PageHeading eyebrow={summary.mode==='cluster'?'Live cluster':'Demo inventory'} title="Cluster overview" description="Health and inventory for the Kubernetes control plane." observedAt={summary.observedAt}/>
    <section className="metric-grid"><Metric label="Node readiness" value={`${summary.nodes.ready}/${summary.nodes.total}`} detail={`${nodeHealth}% ready`} icon="node" status={nodeHealth===100?'healthy':'warning'}/><Metric label="Running pods" value={String(summary.pods.running)} detail={`${healthyPods}% of ${podTotal} pods`} icon="pod" status={summary.pods.failed?'critical':'healthy'}/><Metric label="Namespaces" value={String(summary.namespaces)} detail="Across the cluster" icon="namespace" status="neutral"/><Metric label="Active findings" value={hasRisk?String(summary.pods.failed+summary.nodes.total-summary.nodes.ready):'0'} detail={hasRisk?'Requires attention':'No critical findings'} icon={hasRisk?'warning':'check'} status={hasRisk?'critical':'healthy'}/></section>
    <section className="content-grid"><div className="panel health-panel"><PanelHeader title="Pod lifecycle" description="Current phase distribution" meta={`${podTotal} total`}/><div className="phase-bar">{podTotal>0?<><span className="running" style={{width:`${summary.pods.running/podTotal*100}%`}}/><span className="pending" style={{width:`${summary.pods.pending/podTotal*100}%`}}/><span className="failed" style={{width:`${summary.pods.failed/podTotal*100}%`}}/></>:<span className="empty"/>}</div><div className="phase-table"><PhaseRow label="Running" count={summary.pods.running} total={podTotal} tone="running"/><PhaseRow label="Pending" count={summary.pods.pending} total={podTotal} tone="pending"/><PhaseRow label="Failed" count={summary.pods.failed} total={podTotal} tone="failed"/><PhaseRow label="Succeeded" count={summary.pods.succeeded} total={podTotal} tone="succeeded"/><PhaseRow label="Unknown" count={summary.pods.unknown} total={podTotal} tone="unknown"/></div></div><div className="panel posture-panel"><PanelHeader title="Platform posture" description="Configured safeguards" meta="Baseline"/><PostureItem title="Read-only access" detail="Kubernetes RBAC"/><PostureItem title="Encrypted secrets" detail="AWS KMS"/><PostureItem title="Workload isolation" detail="NetworkPolicy"/><PostureItem title="Identity federation" detail="EKS Pod Identity"/><p className="posture-note">Configuration posture is declared by the GitOps platform baseline.</p></div></section>
    <section className="panel inventory-panel"><PanelHeader title="Inventory status" description="Control-plane resources observed by the Go API" meta="Live"/><div className="inventory-table"><div className="inventory-row header"><span>Resource</span><span>Observed</span><span>State</span><span>Source</span></div><InventoryRow icon="node" resource="Nodes" observed={summary.nodes.total} state={summary.nodes.ready===summary.nodes.total?'Ready':'Check'} source="core/v1"/><InventoryRow icon="pod" resource="Pods" observed={podTotal} state={summary.pods.failed?'Check':'Healthy'} source="core/v1"/><InventoryRow icon="namespace" resource="Namespaces" observed={summary.namespaces} state="Healthy" source="core/v1"/></div></section></>
}

function WorkloadsPage({data}:{data:Workloads}) {
  const [query,setQuery]=useState(''),[namespace,setNamespace]=useState('all')
  const namespaces=useMemo(()=>['all',...Array.from(new Set(data.items.map(i=>i.namespace))).sort()],[data.items])
  const filtered=data.items.filter(item=>(namespace==='all'||item.namespace===namespace)&&`${item.name} ${item.kind}`.toLowerCase().includes(query.toLowerCase()))
  const healthy=data.items.filter(item=>item.status==='Healthy').length
  return <><PageHeading eyebrow="Live controllers" title="Workloads" description="Replica health across Deployments, StatefulSets, and DaemonSets." observedAt={data.observedAt}/><section className="metric-grid three"><Metric label="Controllers" value={String(data.items.length)} detail="Observed resources" icon="workloads" status="neutral"/><Metric label="Healthy" value={String(healthy)} detail={`${data.items.length?Math.round(healthy/data.items.length*100):0}% available`} icon="check" status="healthy"/><Metric label="Needs attention" value={String(data.items.length-healthy)} detail="Progressing or unavailable" icon="warning" status={healthy===data.items.length?'healthy':'warning'}/></section><section className="panel data-panel"><PanelHeader title="Controller inventory" description="Read-only live state from apps/v1" meta={`${filtered.length} shown`}/><Filters><input aria-label="Search workloads" placeholder="Search workload or kind" value={query} onChange={e=>setQuery(e.target.value)}/><select aria-label="Filter namespace" value={namespace} onChange={e=>setNamespace(e.target.value)}>{namespaces.map(ns=><option key={ns} value={ns}>{ns==='all'?'All namespaces':ns}</option>)}</select></Filters><DataTable headers={['Workload','Namespace','Kind','Ready','Status','Age']}>{filtered.map(item=><tr key={`${item.namespace}/${item.kind}/${item.name}`}><td><strong>{item.name}</strong></td><td><code>{item.namespace}</code></td><td>{item.kind}</td><td>{item.ready}/{item.desired}</td><td><Status value={item.status}/></td><td>{age(item.createdAt)}</td></tr>)}</DataTable>{!filtered.length&&<Empty message="No workloads match the current filters."/>}</section></>
}

function NetworkPage({data}:{data:Network}) { return <><PageHeading eyebrow="Cluster connectivity" title="Network" description="Services, ingress exposure, and workload isolation policies." observedAt={data.observedAt}/><section className="metric-grid three"><Metric label="Services" value={String(data.services.length)} detail="Cluster service endpoints" icon="network" status="neutral"/><Metric label="Ingresses" value={String(data.ingresses.length)} detail="External entry points" icon="network" status="healthy"/><Metric label="NetworkPolicies" value={String(data.policies.length)} detail="Traffic controls" icon="security" status="healthy"/></section><section className="split-panels"><div className="panel data-panel"><PanelHeader title="Ingress exposure" description="Hosts reconciled by the AWS Load Balancer Controller" meta={`${data.ingresses.length} routes`}/><DataTable headers={['Ingress','Namespace','Class','Hosts','Address']}>{data.ingresses.map(i=><tr key={`${i.namespace}/${i.name}`}><td><strong>{i.name}</strong></td><td><code>{i.namespace}</code></td><td>{i.class}</td><td>{i.hosts.join(', ')||'—'}</td><td className="truncate" title={i.address}>{i.address}</td></tr>)}</DataTable>{!data.ingresses.length&&<Empty message="No ingress resources found."/>}</div><div className="panel data-panel"><PanelHeader title="Services" description="Stable virtual IPs and ports" meta={`${data.services.length} services`}/><DataTable headers={['Service','Namespace','Type','Cluster IP','Ports']}>{data.services.map(s=><tr key={`${s.namespace}/${s.name}`}><td><strong>{s.name}</strong></td><td><code>{s.namespace}</code></td><td>{s.type}</td><td><code>{s.clusterIp}</code></td><td>{s.ports.join(', ')}</td></tr>)}</DataTable></div></section><section className="panel data-panel"><PanelHeader title="NetworkPolicy inventory" description="Declared ingress and egress rule counts" meta={`${data.policies.length} policies`}/><DataTable headers={['Policy','Namespace','Ingress rules','Egress rules','Coverage']}>{data.policies.map(p=><tr key={`${p.namespace}/${p.name}`}><td><strong>{p.name}</strong></td><td><code>{p.namespace}</code></td><td>{p.ingressRules}</td><td>{p.egressRules}</td><td><Status value={p.ingressRules+p.egressRules>0?'Enforced':'Empty'}/></td></tr>)}</DataTable></section></> }

function EventsPage({data}:{data:Events}) {
  const [onlyWarnings,setOnlyWarnings]=useState(false); const items=onlyWarnings?data.items.filter(e=>e.type==='Warning'):data.items; const warnings=data.items.filter(e=>e.type==='Warning').length
  return <><PageHeading eyebrow="Control-plane feed" title="Events" description="The 100 most recent Kubernetes events, newest first." observedAt={data.observedAt}/><section className="metric-grid three"><Metric label="Recent events" value={String(data.items.length)} detail="Current API retention window" icon="events" status="neutral"/><Metric label="Warnings" value={String(warnings)} detail="Potentially actionable" icon="warning" status={warnings?'warning':'healthy'}/><Metric label="Normal" value={String(data.items.length-warnings)} detail="Expected operations" icon="check" status="healthy"/></section><section className="panel data-panel"><PanelHeader title="Event stream" description="Reasons and messages emitted by Kubernetes controllers" meta={`${items.length} shown`}/><Filters><label className="toggle"><input type="checkbox" checked={onlyWarnings} onChange={e=>setOnlyWarnings(e.target.checked)}/><span>Warnings only</span></label></Filters><DataTable headers={['Severity','Reason','Object','Namespace','Message','Count','Last seen']}>{items.map((event,index)=><tr key={`${event.namespace}/${event.object}/${event.reason}/${index}`}><td><Status value={event.type}/></td><td><strong>{event.reason}</strong></td><td><code>{event.object}</code></td><td><code>{event.namespace}</code></td><td className="message-cell">{event.message}</td><td>{event.count||1}</td><td>{relative(event.lastSeen)}</td></tr>)}</DataTable>{!items.length&&<Empty message="No events match the current filter."/>}</section></>
}

function ObservabilityPage({data}:{data:Observability}) {
  const healthy=data.components.filter(c=>c.status==='Healthy').length
  return <><PageHeading eyebrow="Telemetry pipeline" title="Observability" description="Runtime health for metrics, logs, traces, and OpenTelemetry collectors." observedAt={data.observedAt}/><section className="signal-grid">{data.signals.map((signal,index)=><article className="signal-card" key={signal}><span>{String(index+1).padStart(2,'0')}</span><Icon name="observability"/><div><strong>{signal.split(' / ')[0]}</strong><small>{signal.split(' / ')[1]}</small></div><Status value="Connected"/></article>)}</section><section className="panel data-panel"><PanelHeader title="Observability components" description={`Workloads in the ${data.namespace} namespace`} meta={`${healthy}/${data.components.length} healthy`}/><DataTable headers={['Component','Kind','Ready','Status','Purpose']}>{data.components.map(c=><tr key={`${c.kind}/${c.name}`}><td><strong>{c.name}</strong></td><td>{c.kind}</td><td>{c.ready}/{c.desired}</td><td><Status value={c.status}/></td><td>{componentPurpose(c.name)}</td></tr>)}</DataTable></section><aside className="callout"><Icon name="security"/><div><strong>Administrative UIs remain private</strong><span>Grafana and Argo CD are intentionally accessed through authenticated port forwarding instead of public load balancers.</span></div></aside></>
}

function SecurityPage({data}:{data:Security}) {
  const critical=data.findings.filter(f=>f.severity==='Critical').length, warnings=data.findings.filter(f=>f.severity==='Warning').length
  return <><PageHeading eyebrow="Read-only posture scan" title="Security" description="Runtime checks derived from Pod security contexts and NetworkPolicy coverage." observedAt={data.observedAt}/><section className="metric-grid three"><Metric label="Pods evaluated" value={String(data.podsEvaluated)} detail="Container security contexts" icon="pod" status="neutral"/><Metric label="Critical findings" value={String(critical)} detail="Privileged containers" icon="warning" status={critical?'critical':'healthy'}/><Metric label="NetworkPolicies" value={String(data.networkPolicies)} detail="Declared traffic boundaries" icon="security" status="healthy"/></section><section className="panel data-panel"><PanelHeader title="Runtime findings" description="Informational analysis; no resources are mutated" meta={`${critical} critical · ${warnings} warning`}/><DataTable headers={['Severity','Category','Resource','Namespace','Finding']}>{data.findings.map((f,index)=><tr key={`${f.resource}/${f.category}/${index}`}><td><Status value={f.severity}/></td><td><strong>{f.category}</strong></td><td><code>{f.resource}</code></td><td><code>{f.namespace}</code></td><td className="message-cell">{f.message}</td></tr>)}</DataTable>{!data.findings.length&&<Empty message="No runtime findings detected."/>}</section><aside className="callout"><Icon name="check"/><div><strong>Designed for least privilege</strong><span>KubeVista can only get, list, and watch approved Kubernetes resource types. It cannot create, update, delete, exec, or read Secrets.</span></div></aside></>
}

function CostPage({data}:{data:Cost}) {
  const monthly=data.estimatedHourly*730
  return <><PageHeading eyebrow="Directional model" title="Cost" description="A transparent hourly estimate based on live node types and fixed EKS infrastructure." observedAt={data.observedAt}/><section className="cost-hero"><div><small>Estimated monthly run rate</small><strong>${monthly.toFixed(0)}</strong><span>≈ ${data.estimatedHourly.toFixed(3)}/hour · {data.currency}</span></div><div className="cost-breakdown"><CostLine label="EKS control plane" value={data.controlPlaneHourly}/><CostLine label="Application Load Balancer" value={data.loadBalancerHourly}/><CostLine label="NAT Gateway" value={data.natGatewayHourly}/><CostLine label="Worker compute" value={data.nodes.reduce((sum,n)=>sum+n.estimatedHourly,0)}/></div></section><section className="panel data-panel"><PanelHeader title="Worker compute" description="Capacity observed from Kubernetes node labels" meta={`${data.nodes.length} nodes`}/><DataTable headers={['Node','Instance type','Capacity','Hourly estimate','Monthly estimate']}>{data.nodes.map(n=><tr key={n.name}><td className="truncate"><strong>{n.name}</strong></td><td><code>{n.instanceType}</code></td><td><Status value={n.capacityType}/></td><td>${n.estimatedHourly.toFixed(4)}</td><td>${(n.estimatedHourly*730).toFixed(2)}</td></tr>)}</DataTable></section><p className="disclaimer">{data.disclaimer}</p></>
}

function SettingsPage({data}:{data:Settings}) { return <><PageHeading eyebrow="Console configuration" title="Settings" description="Runtime identity and operating behavior for this read-only deployment."/><section className="settings-grid"><Setting label="Cluster" value={data.cluster}/><Setting label="Environment" value={data.environment}/><Setting label="Application version" value={data.version}/><Setting label="Access mode" value={data.readOnly?'Read only':'Write enabled'}/><Setting label="Refresh interval" value={`${data.refreshSeconds} seconds`}/><Setting label="Region" value="us-west-2"/></section><section className="panel settings-panel"><PanelHeader title="Operational guarantees" description="Controls enforced outside the browser" meta="Production baseline"/><PostureItem title="Cognito authentication" detail="Hosted sign-in and mandatory TOTP MFA"/><PostureItem title="TLS at the edge" detail="ACM certificate with HTTP-to-HTTPS redirect"/><PostureItem title="GitOps delivery" detail="Argo CD automated reconciliation and self-healing"/><PostureItem title="Immutable application images" detail="ECR digests produced with SBOM and provenance"/></section></> }

function LoadingPage(){return <><div className="page-heading skeleton-heading"><div><span/><span/></div></div><div className="loading-grid">{[1,2,3,4].map(i=><span key={i}/>)}</div><div className="panel loading-panel"/></>}
function Metric({label,value,detail,icon,status}:{label:string;value:string;detail:string;icon:IconName;status:'healthy'|'warning'|'critical'|'neutral'}){return <article className={`metric ${status}`}><div className="metric-label"><span>{label}</span><Icon name={icon}/></div><strong>{value}</strong><p>{detail}</p></article>}
function PanelHeader({title,description,meta}:{title:string;description:string;meta:string}){return <div className="panel-header"><div><h2>{title}</h2><p>{description}</p></div><span>{meta}</span></div>}
function Filters({children}:{children:React.ReactNode}){return <div className="filters">{children}</div>}
function DataTable({headers,children}:{headers:string[];children:React.ReactNode}){return <div className="table-scroll"><table><thead><tr>{headers.map(h=><th key={h}>{h}</th>)}</tr></thead><tbody>{children}</tbody></table></div>}
function Status({value}:{value:string}){const lower=value.toLowerCase();const tone=lower.includes('critical')||lower.includes('unavailable')?'critical':lower.includes('warning')||lower.includes('progress')||lower.includes('pending')?'warning':lower.includes('info')||lower.includes('spot')?'info':'healthy';return <span className={`status-badge ${tone}`}><i/>{value}</span>}
function Empty({message}:{message:string}){return <div className="empty-state"><Icon name="check"/><span>{message}</span></div>}
function PhaseRow({label,count,total,tone}:{label:string;count:number;total:number;tone:string}){return <div className="phase-row"><span><i className={tone}/>{label}</span><strong>{count}</strong><span>{total?`${Math.round(count/total*100)}%`:'—'}</span></div>}
function PostureItem({title,detail}:{title:string;detail:string}){return <div className="posture-item"><span><Icon name="check"/></span><div><strong>{title}</strong><small>{detail}</small></div></div>}
function InventoryRow({icon,resource,observed,state,source}:{icon:IconName;resource:string;observed:number;state:string;source:string}){return <div className="inventory-row"><span><Icon name={icon}/><strong>{resource}</strong></span><span>{observed}</span><span><Status value={state}/></span><code>{source}</code></div>}
function CostLine({label,value}:{label:string;value:number}){return <div><span>{label}</span><strong>${value.toFixed(4)}<small>/hr</small></strong></div>}
function Setting({label,value}:{label:string;value:string}){return <article><small>{label}</small><strong>{value}</strong></article>}
function age(value:string){const days=Math.floor((Date.now()-new Date(value).getTime())/86_400_000);if(days>0)return `${days}d`;const hours=Math.floor((Date.now()-new Date(value).getTime())/3_600_000);return hours>0?`${hours}h`:'<1h'}
function relative(value:string){const minutes=Math.max(0,Math.floor((Date.now()-new Date(value).getTime())/60_000));if(minutes<1)return 'just now';if(minutes<60)return `${minutes}m ago`;const hours=Math.floor(minutes/60);return hours<24?`${hours}h ago`:`${Math.floor(hours/24)}d ago`}
function componentPurpose(name:string){const n=name.toLowerCase();if(n.includes('grafana'))return 'Dashboards';if(n.includes('prometheus'))return 'Metrics';if(n.includes('loki'))return 'Logs';if(n.includes('tempo'))return 'Traces';if(n.includes('otel'))return 'Telemetry routing';if(n.includes('alert'))return 'Alert routing';return 'Platform telemetry'}

function Logo(){return <svg className="logo" viewBox="0 0 32 32" aria-hidden="true"><path d="M16 2.5 27.7 9v14L16 29.5 4.3 23V9L16 2.5Z"/><path d="m16 8 6.5 3.7v7.6L16 23l-6.5-3.7v-7.6L16 8Z"/><circle cx="16" cy="16" r="2.3"/></svg>}
function Icon({name}:{name:IconName}){const paths:Record<IconName,React.ReactNode>={overview:<><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></>,workloads:<><path d="m12 2 8 4.5v9L12 20l-8-4.5v-9L12 2Z"/><path d="m4.3 6.7 7.7 4.4 7.7-4.4M12 11.1V20"/></>,network:<><circle cx="5" cy="6" r="2.5"/><circle cx="19" cy="6" r="2.5"/><circle cx="12" cy="18" r="2.5"/><path d="m7.4 7 3.4 8.5M16.6 7l-3.4 8.5M7.5 6h9"/></>,events:<><path d="M5 3h14v18H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></>,observability:<><path d="M3 18V9m6 9V5m6 13v-7m6 7V2"/><path d="M2 21h20"/></>,security:<path d="M12 2.5 20 6v5.6c0 5-3.4 8.2-8 9.9-4.6-1.7-8-4.9-8-9.9V6l8-3.5Z"/>,cost:<><circle cx="12" cy="12" r="9"/><path d="M15.5 8.5c-.7-.7-1.7-1-3-1-1.7 0-3 .8-3 2s1 1.8 3.2 2.3c2 .5 3 1.2 3 2.5 0 1.4-1.3 2.3-3.2 2.3-1.3 0-2.5-.4-3.4-1.2M12.5 5.5v13"/></>,settings:<><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8l.1.1-2.9 2.9-.1-.1a1.6 1.6 0 0 0-1.8-.3 1.6 1.6 0 0 0-1 1.5V21h-4v-.1a1.6 1.6 0 0 0-1-1.5 1.6 1.6 0 0 0-1.8.3l-.1.1-2.9-2.9.1-.1a1.6 1.6 0 0 0 .3-1.8 1.6 1.6 0 0 0-1.5-1H3v-4h.1a1.6 1.6 0 0 0 1.5-1 1.6 1.6 0 0 0-.3-1.8l-.1-.1 2.9-2.9.1.1a1.6 1.6 0 0 0 1.8.3 1.6 1.6 0 0 0 1-1.5V3h4v.1a1.6 1.6 0 0 0 1 1.5 1.6 1.6 0 0 0 1.8-.3l.1-.1 2.9 2.9-.1.1a1.6 1.6 0 0 0-.3 1.8 1.6 1.6 0 0 0 1.5 1h.1v4h-.1a1.6 1.6 0 0 0-1.5 1Z"/></>,refresh:<><path d="M20 7v5h-5"/><path d="M19 12a7 7 0 1 0-1.3 5.3"/></>,check:<path d="m5 12 4.2 4.2L19 6.5"/>,warning:<><path d="M12 3 2.8 20h18.4L12 3Z"/><path d="M12 9v5m0 3h.01"/></>,node:<><rect x="3" y="5" width="18" height="14" rx="1"/><path d="M7 9h3m-3 4h3m6-4h1m-1 4h1"/></>,pod:<><path d="m12 2.5 8 4.7v9.6l-8 4.7-8-4.7V7.2l8-4.7Z"/><circle cx="12" cy="12" r="3"/></>,namespace:<><rect x="3" y="4" width="18" height="16" rx="1"/><path d="M3 9h18M8 9v11"/></>,clock:<><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></>};return <svg className="icon" viewBox="0 0 24 24" aria-hidden="true">{paths[name]}</svg>}

ReactDOM.createRoot(document.getElementById('root')!).render(<React.StrictMode><App/></React.StrictMode>)
