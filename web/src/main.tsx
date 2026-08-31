import React from 'react'
import ReactDOM from 'react-dom/client'
import './styles.css'

const services = [
  ['checkout-api', 'commerce', '12 / 12', '99.99%', 'Healthy'],
  ['catalog-api', 'commerce', '8 / 8', '99.97%', 'Healthy'],
  ['events-worker', 'platform', '5 / 6', '98.41%', 'Degraded'],
  ['otel-collector', 'observability', '3 / 3', '100%', 'Healthy']
]

function App() {
  return <div className="shell">
    <aside>
      <div className="brand"><span className="hex">⬡</span><div>KubeVista<small>PLATFORM CONSOLE</small></div></div>
      <nav><b>OVERVIEW</b><a className="active">◫ Dashboard</a><a>◉ Workloads</a><a>⌁ Network flows</a><b>OPERATIONS</b><a>⌁ Deployments</a><a>◈ Events</a><a>◌ Cost</a><b>PLATFORM</b><a>◎ Observability</a><a>◇ Security posture</a></nav>
      <div className="cluster"><i></i><div><strong>portfolio-eks</strong><span>us-west-2 · healthy</span></div></div>
    </aside>
    <main>
      <header><div><p>PRODUCTION / US-WEST-2</p><h1>Cluster overview</h1></div><button>Last 30 minutes⌄</button></header>
      <section className="stats">
        <Card title="NODE HEALTH" value="3 / 3" detail="All nodes ready" tone="green" />
        <Card title="RUNNING PODS" value="42" detail="1 pending · 0 failed" tone="cyan" />
        <Card title="CPU UTILIZATION" value="38%" detail="12.4 of 32 cores" tone="amber" />
        <Card title="MEMORY" value="61%" detail="47.8 of 78 GiB" tone="violet" />
      </section>
      <section className="grid">
        <div className="panel wide"><div className="panelHead"><div><small>RESOURCE UTILIZATION</small><h2>Cluster capacity</h2></div><span className="legend">● CPU &nbsp; <em>● Memory</em></span></div><div className="chart"><div className="y">100%<span>75%</span><span>50%</span><span>25%</span><span>0%</span></div><svg viewBox="0 0 700 210" preserveAspectRatio="none"><defs><linearGradient id="fill" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stopColor="#22d3ee" stopOpacity=".24"/><stop offset="1" stopColor="#22d3ee" stopOpacity="0"/></linearGradient></defs><path className="area" d="M0 160 C80 145,90 110,170 130 S280 80,355 110 S450 140,520 90 S630 70,700 72 L700 210 L0 210Z"/><path className="cpu" d="M0 160 C80 145,90 110,170 130 S280 80,355 110 S450 140,520 90 S630 70,700 72"/><path className="mem" d="M0 115 C100 105,145 92,220 100 S330 70,410 82 S530 55,610 60 S670 47,700 50"/></svg><div className="x"><span>14:00</span><span>14:06</span><span>14:12</span><span>14:18</span><span>14:24</span><span>14:30</span></div></div></div>
        <div className="panel"><div className="panelHead"><div><small>WORKLOAD STATUS</small><h2>Pod health</h2></div></div><div className="donut"><div><strong>43</strong><span>TOTAL PODS</span></div></div><div className="keys"><span><i className="green"></i>Running <b>42</b></span><span><i className="yellow"></i>Pending <b>1</b></span><span><i className="red"></i>Failed <b>0</b></span></div></div>
      </section>
      <section className="panel table"><div className="panelHead"><div><small>LIVE INVENTORY</small><h2>Workloads</h2></div><button>View all workloads →</button></div><div className="rows heading"><span>WORKLOAD</span><span>NAMESPACE</span><span>READY</span><span>SLO</span><span>STATUS</span></div>{services.map((s) => <div className="rows" key={s[0]}><span><i className="cube">◆</i><strong>{s[0]}</strong></span><span>{s[1]}</span><span>{s[2]}</span><span>{s[3]}</span><span className={s[4] === 'Healthy' ? 'healthy' : 'degraded'}>● {s[4]}</span></div>)}</section>
    </main>
  </div>
}

function Card({title,value,detail,tone}:{title:string,value:string,detail:string,tone:string}) { return <div className={`card ${tone}`}><small>{title}</small><strong>{value}</strong><span>{detail}</span><div className="spark">╱╲╱╲╲╱╲</div></div> }

ReactDOM.createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>)
