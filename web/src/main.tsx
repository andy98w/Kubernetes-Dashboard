import React, { useCallback, useEffect, useMemo, useState } from 'react'
import ReactDOM from 'react-dom/client'
import './styles.css'

type Summary = {
  cluster: string
  mode: 'demo' | 'cluster'
  nodes: { ready: number; total: number }
  namespaces: number
  pods: { running: number; pending: number; failed: number; succeeded: number; unknown: number }
  observedAt: string
}

type RequestState = 'loading' | 'connected' | 'stale'
type IconName = 'overview' | 'workloads' | 'network' | 'events' | 'observability' | 'security' | 'cost' | 'settings' | 'refresh' | 'check' | 'warning' | 'node' | 'pod' | 'namespace' | 'clock'

const fallbackSummary: Summary = {
  cluster: 'kubevista-dev', mode: 'demo', nodes: { ready: 0, total: 0 }, namespaces: 0,
  pods: { running: 0, pending: 0, failed: 0, succeeded: 0, unknown: 0 }, observedAt: '',
}

const navigation: Array<{ label: string; items: Array<{ icon: IconName; name: string; active?: boolean; planned?: boolean }> }> = [
  { label: 'Cluster', items: [
    { icon: 'overview', name: 'Overview', active: true }, { icon: 'workloads', name: 'Workloads', planned: true },
    { icon: 'network', name: 'Network', planned: true }, { icon: 'events', name: 'Events', planned: true },
  ] },
  { label: 'Platform', items: [
    { icon: 'observability', name: 'Observability', planned: true }, { icon: 'security', name: 'Security', planned: true },
    { icon: 'cost', name: 'Cost', planned: true },
  ] },
]

function App() {
  const [summary, setSummary] = useState<Summary>(fallbackSummary)
  const [requestState, setRequestState] = useState<RequestState>('loading')
  const [error, setError] = useState<string | null>(null)
  const [refreshing, setRefreshing] = useState(false)

  const refresh = useCallback(async (signal?: AbortSignal) => {
    setRefreshing(true)
    try {
      const response = await fetch('/api/v1/summary', { signal })
      if (!response.ok) throw new Error(`API returned ${response.status}`)
      setSummary(await response.json() as Summary)
      setRequestState('connected')
      setError(null)
    } catch (caught) {
      if (caught instanceof DOMException && caught.name === 'AbortError') return
      setRequestState('stale')
      setError(caught instanceof Error ? caught.message : 'Cluster API unavailable')
    } finally { setRefreshing(false) }
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    void refresh(controller.signal)
    const timer = window.setInterval(() => void refresh(controller.signal), 15_000)
    return () => { controller.abort(); window.clearInterval(timer) }
  }, [refresh])

  const podTotal = useMemo(() => Object.values(summary.pods).reduce((total, count) => total + count, 0), [summary.pods])
  const healthyPods = podTotal > 0 ? Math.round((summary.pods.running / podTotal) * 100) : 0
  const nodeHealth = summary.nodes.total > 0 ? Math.round((summary.nodes.ready / summary.nodes.total) * 100) : 0
  const hasRisk = summary.pods.failed > 0 || summary.nodes.ready < summary.nodes.total
  const observed = summary.observedAt ? new Date(summary.observedAt) : null

  return <div className="app-shell">
    <aside className="sidebar">
      <div className="brand-mark" aria-label="KubeVista home"><Logo /><div><strong>KubeVista</strong><span>Operations console</span></div></div>
      <div className="environment-switcher">
        <span className={`status-dot ${requestState}`} aria-hidden="true" /><div><small>Environment</small><strong>Development</strong></div><span className="chevron">⌄</span>
      </div>
      <nav aria-label="Primary navigation">
        {navigation.map((section) => <div className="nav-section" key={section.label}><p>{section.label}</p>
          {section.items.map((item) => <button className={item.active ? 'active' : ''} disabled={item.planned} key={item.name}>
            <Icon name={item.icon} /><span>{item.name}</span>{item.planned && <small>Soon</small>}
          </button>)}
        </div>)}
      </nav>
      <div className="sidebar-footer">
        <button disabled><Icon name="settings" /><span>Settings</span><small>Soon</small></button>
        <div className="version-row"><span>KV</span><div><strong>Portfolio build</strong><small>v0.1.0</small></div></div>
      </div>
    </aside>

    <main>
      <header className="topbar">
        <div className="breadcrumbs"><span>Clusters</span><b>/</b><strong>{summary.cluster}</strong></div>
        <div className="topbar-actions">
          <span className={`connection-pill ${requestState}`}><span />{requestState === 'connected' ? 'API connected' : requestState === 'loading' ? 'Connecting' : 'API unavailable'}</span>
          <button className="icon-button" onClick={() => void refresh()} disabled={refreshing} aria-label="Refresh cluster summary"><Icon name="refresh" /></button>
        </div>
      </header>

      <div className="workspace">
        <section className="page-heading">
          <div><div className="eyebrow"><span>{summary.mode === 'cluster' ? 'Live cluster' : 'Demo inventory'}</span><i />us-west-2</div><h1>Cluster overview</h1><p>Health and inventory for the Kubernetes control plane.</p></div>
          <div className="observation"><Icon name="clock" /><div><small>Last observed</small><strong>{observed ? observed.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }) : 'Waiting for API'}</strong></div></div>
        </section>

        {requestState === 'stale' && <div className="notice" role="status"><Icon name="warning" /><div><strong>Live inventory is unavailable</strong><span>{error}. Showing the last successful response where available.</span></div><button onClick={() => void refresh()}>Retry connection</button></div>}

        <section className="metric-grid" aria-label="Cluster metrics">
          <Metric label="Node readiness" value={`${summary.nodes.ready}/${summary.nodes.total}`} detail={summary.nodes.total ? `${nodeHealth}% ready` : 'No inventory yet'} icon="node" status={summary.nodes.total > 0 && nodeHealth === 100 ? 'healthy' : 'neutral'} />
          <Metric label="Running pods" value={String(summary.pods.running)} detail={podTotal ? `${healthyPods}% of ${podTotal} pods` : 'No inventory yet'} icon="pod" status={summary.pods.failed > 0 ? 'critical' : summary.pods.pending > 0 ? 'warning' : 'healthy'} />
          <Metric label="Namespaces" value={String(summary.namespaces)} detail="Across the cluster" icon="namespace" status="neutral" />
          <Metric label="Active findings" value={hasRisk ? String(summary.pods.failed + (summary.nodes.total - summary.nodes.ready)) : '0'} detail={hasRisk ? 'Requires attention' : 'No critical findings'} icon={hasRisk ? 'warning' : 'check'} status={hasRisk ? 'critical' : 'healthy'} />
        </section>

        <section className="content-grid">
          <div className="panel health-panel"><PanelHeader title="Pod lifecycle" description="Current phase distribution" meta={`${podTotal} total`} />
            <div className="phase-bar" aria-label={`${healthyPods}% of pods are running`}>
              {podTotal > 0 ? <><span className="running" style={{ width: `${summary.pods.running / podTotal * 100}%` }} /><span className="pending" style={{ width: `${summary.pods.pending / podTotal * 100}%` }} /><span className="failed" style={{ width: `${summary.pods.failed / podTotal * 100}%` }} /><span className="succeeded" style={{ width: `${summary.pods.succeeded / podTotal * 100}%` }} /><span className="unknown" style={{ width: `${summary.pods.unknown / podTotal * 100}%` }} /></> : <span className="empty" />}
            </div>
            <div className="phase-table" role="table" aria-label="Pod phase counts"><PhaseRow label="Running" count={summary.pods.running} total={podTotal} tone="running" /><PhaseRow label="Pending" count={summary.pods.pending} total={podTotal} tone="pending" /><PhaseRow label="Failed" count={summary.pods.failed} total={podTotal} tone="failed" /><PhaseRow label="Succeeded" count={summary.pods.succeeded} total={podTotal} tone="succeeded" /><PhaseRow label="Unknown" count={summary.pods.unknown} total={podTotal} tone="unknown" /></div>
          </div>
          <div className="panel posture-panel"><PanelHeader title="Platform posture" description="Configured safeguards" meta="Baseline" /><PostureItem title="Read-only access" detail="Kubernetes RBAC" /><PostureItem title="Encrypted secrets" detail="AWS KMS" /><PostureItem title="Workload isolation" detail="NetworkPolicy" /><PostureItem title="Identity federation" detail="EKS Pod Identity" /><p className="posture-note">Configuration posture is declared by the platform baseline. Runtime policy evaluation is on the roadmap.</p></div>
        </section>

        <section className="panel inventory-panel"><PanelHeader title="Inventory status" description="Control-plane resources observed by the Go API" meta={summary.mode === 'cluster' ? 'Live' : 'Demo mode'} />
          <div className="inventory-table" role="table" aria-label="Cluster inventory status"><div className="inventory-row header" role="row"><span>Resource</span><span>Observed</span><span>State</span><span>Source</span></div><InventoryRow icon="node" resource="Nodes" observed={summary.nodes.total} state={summary.nodes.ready === summary.nodes.total && summary.nodes.total > 0 ? 'Ready' : 'Check'} source="core/v1" /><InventoryRow icon="pod" resource="Pods" observed={podTotal} state={summary.pods.failed > 0 ? 'Check' : podTotal > 0 ? 'Healthy' : 'Waiting'} source="core/v1" /><InventoryRow icon="namespace" resource="Namespaces" observed={summary.namespaces} state={summary.namespaces > 0 ? 'Healthy' : 'Waiting'} source="core/v1" /></div>
        </section>
      </div>
    </main>
  </div>
}

function Metric({ label, value, detail, icon, status }: { label: string; value: string; detail: string; icon: IconName; status: 'healthy' | 'warning' | 'critical' | 'neutral' }) { return <article className={`metric ${status}`}><div className="metric-label"><span>{label}</span><Icon name={icon} /></div><strong>{value}</strong><p>{detail}</p></article> }
function PanelHeader({ title, description, meta }: { title: string; description: string; meta: string }) { return <div className="panel-header"><div><h2>{title}</h2><p>{description}</p></div><span>{meta}</span></div> }
function PhaseRow({ label, count, total, tone }: { label: string; count: number; total: number; tone: string }) { return <div className="phase-row" role="row"><span><i className={tone} />{label}</span><strong>{count}</strong><span>{total ? `${Math.round(count / total * 100)}%` : '—'}</span></div> }
function PostureItem({ title, detail }: { title: string; detail: string }) { return <div className="posture-item"><span><Icon name="check" /></span><div><strong>{title}</strong><small>{detail}</small></div></div> }
function InventoryRow({ icon, resource, observed, state, source }: { icon: IconName; resource: string; observed: number; state: string; source: string }) { const tone = state === 'Healthy' || state === 'Ready' ? 'healthy' : state === 'Check' ? 'critical' : 'waiting'; return <div className="inventory-row" role="row"><span><Icon name={icon} /><strong>{resource}</strong></span><span>{observed}</span><span><i className={tone} />{state}</span><code>{source}</code></div> }
function Logo() { return <svg className="logo" viewBox="0 0 32 32" aria-hidden="true"><path d="M16 2.5 27.7 9v14L16 29.5 4.3 23V9L16 2.5Z"/><path d="m16 8 6.5 3.7v7.6L16 23l-6.5-3.7v-7.6L16 8Z"/><circle cx="16" cy="16" r="2.3"/></svg> }

function Icon({ name }: { name: IconName }) {
  const paths: Record<IconName, React.ReactNode> = {
    overview: <><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></>,
    workloads: <><path d="m12 2 8 4.5v9L12 20l-8-4.5v-9L12 2Z"/><path d="m4.3 6.7 7.7 4.4 7.7-4.4M12 11.1V20"/></>,
    network: <><circle cx="5" cy="6" r="2.5"/><circle cx="19" cy="6" r="2.5"/><circle cx="12" cy="18" r="2.5"/><path d="m7.4 7 3.4 8.5M16.6 7l-3.4 8.5M7.5 6h9"/></>,
    events: <><path d="M5 3h14v18H5z"/><path d="M8 8h8M8 12h8M8 16h5"/></>,
    observability: <><path d="M3 18V9m6 9V5m6 13v-7m6 7V2"/><path d="M2 21h20"/></>, security: <path d="M12 2.5 20 6v5.6c0 5-3.4 8.2-8 9.9-4.6-1.7-8-4.9-8-9.9V6l8-3.5Z"/>,
    cost: <><circle cx="12" cy="12" r="9"/><path d="M15.5 8.5c-.7-.7-1.7-1-3-1-1.7 0-3 .8-3 2s1 1.8 3.2 2.3c2 .5 3 1.2 3 2.5 0 1.4-1.3 2.3-3.2 2.3-1.3 0-2.5-.4-3.4-1.2M12.5 5.5v13"/></>,
    settings: <><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8l.1.1-2.9 2.9-.1-.1a1.6 1.6 0 0 0-1.8-.3 1.6 1.6 0 0 0-1 1.5V21h-4v-.1a1.6 1.6 0 0 0-1-1.5 1.6 1.6 0 0 0-1.8.3l-.1.1-2.9-2.9.1-.1a1.6 1.6 0 0 0 .3-1.8 1.6 1.6 0 0 0-1.5-1H3v-4h.1a1.6 1.6 0 0 0 1.5-1 1.6 1.6 0 0 0-.3-1.8l-.1-.1 2.9-2.9.1.1a1.6 1.6 0 0 0 1.8.3 1.6 1.6 0 0 0 1-1.5V3h4v.1a1.6 1.6 0 0 0 1 1.5 1.6 1.6 0 0 0 1.8-.3l.1-.1 2.9 2.9-.1.1a1.6 1.6 0 0 0-.3 1.8 1.6 1.6 0 0 0 1.5 1h.1v4h-.1a1.6 1.6 0 0 0-1.5 1Z"/></>,
    refresh: <><path d="M20 7v5h-5"/><path d="M19 12a7 7 0 1 0-1.3 5.3"/></>, check: <path d="m5 12 4.2 4.2L19 6.5"/>, warning: <><path d="M12 3 2.8 20h18.4L12 3Z"/><path d="M12 9v5m0 3h.01"/></>,
    node: <><rect x="3" y="5" width="18" height="14" rx="1"/><path d="M7 9h3m-3 4h3m6-4h1m-1 4h1"/></>, pod: <><path d="m12 2.5 8 4.7v9.6l-8 4.7-8-4.7V7.2l8-4.7Z"/><circle cx="12" cy="12" r="3"/></>, namespace: <><rect x="3" y="4" width="18" height="16" rx="1"/><path d="M3 9h18M8 9v11"/></>, clock: <><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></>,
  }
  return <svg className="icon" viewBox="0 0 24 24" aria-hidden="true">{paths[name]}</svg>
}

ReactDOM.createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>)
