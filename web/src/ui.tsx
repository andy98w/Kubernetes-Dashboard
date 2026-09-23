import {useRef,type SelectHTMLAttributes} from 'react'

const scenarios=[['probe','Failed readiness probe'],['crashloop','Crash loop'],['imagepull','Image pull failure'],['oom','Out of memory'],['scheduling','Insufficient capacity']]
export function ScenarioPicker({value,onChange}:{value:string;onChange:(value:string)=>void}){
 const root=useRef<HTMLDetailsElement>(null)
 return <details className="scenario-picker" ref={root} onKeyDown={e=>{if(e.key==='Escape'){e.stopPropagation();if(root.current){root.current.open=false;root.current.querySelector('summary')?.focus()}}}}><summary aria-label="Failure scenario">{scenarios.find(([id])=>id===value)?.[1]||'Choose scenario'}<span aria-hidden="true">⌄</span></summary><fieldset><legend>Failure scenario</legend>{scenarios.map(([id,label])=><label key={id}><input type="radio" name="failure-scenario" value={id} checked={id===value} onChange={()=>onChange(id)}/><span>{label}</span></label>)}</fieldset></details>
}

/** Native selection semantics with a shared visual treatment. */
export function Select({children,className='',...props}:SelectHTMLAttributes<HTMLSelectElement>){
  return <span className={`ui-select ${className}`}><select {...props}>{children}</select><svg viewBox="0 0 16 16" aria-hidden="true"><path d="m4 6 4 4 4-4"/></svg></span>
}

export function BotLogo(){return <svg className="bot-logo" viewBox="0 0 48 48" aria-hidden="true"><path d="M24 10V5" stroke="#adc5d1" strokeWidth="3"/><circle cx="24" cy="4" r="3" fill="#a6e5c7"/><rect x="5" y="11" width="38" height="28" rx="11" fill="#f0e2c4"/><rect x="10" y="16" width="28" height="16" rx="7" fill="#233b52"/><path d="M18 21v3m12-3v3m-9 3q3 3 6 0" stroke="#a6e5c7" strokeWidth="2.5" strokeLinecap="round" fill="none"/><path d="M17 40v4m14-4v4" stroke="#adbec7" strokeWidth="4" strokeLinecap="round"/></svg>}
