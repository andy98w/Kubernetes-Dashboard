export type WorkRole='terminal'|'collector'|'blueprint'|'metrics'|'archive'|'trace'
/** Visual metaphors only: names suggest a role, never prove live activity. */
export function workRole(name:string):WorkRole{
 if(/otel|exporter/i.test(name))return 'collector'
 if(/argocd/i.test(name))return 'blueprint'
 if(/prometheus/i.test(name))return 'metrics'
 if(/loki/i.test(name))return 'archive'
 if(/tempo/i.test(name))return 'trace'
 return 'terminal'
}
export const workDescription:Record<WorkRole,string>={
 terminal:'terminal work',collector:'instrument checks',blueprint:'blueprint comparison',
 metrics:'measurement console',archive:'log filing',trace:'trace inspection',
}
export function RobotWork({role,active}:{role:WorkRole;active:boolean}){
 return <g className={`robot-work work-${role} ${active?'work-active':''}`} aria-hidden="true">
  <ellipse cx="4" cy="12" rx="23" ry="7" fill="#081725" opacity=".3"/>
  <path d="M-16-3v15M22-3v15" stroke="#738b98" strokeWidth="3"/>
  <path d="m-23-7 27-9 25 12-28 10Z" fill="#d3b890" stroke="#344e60"/>
  <path d="m-23-7 24 13v4L-23-3Zm24 13 28-10v4L1 10Z" fill="#957d61"/>
  <g transform="translate(7 -19)">
   {role==='archive'?<g><path d="m-15 0 20-7 12 7v12l-20 7-12-7Z" fill="#7fa6a4"/><path className="work-tool" d="m-11-5 16-5 8 4-16 6Z" fill="#eee0bd"/></g>
   :role==='blueprint'?<g><path d="m-18 8 25-9 13 8-25 9Z" fill="#9fc7e7"/><path d="m-10 7 13-4 6 4-13 5Zm7 4 7 3" fill="none" stroke="#325470"/><path className="work-tool" d="m6 2 8-7" stroke="#edd28e" strokeWidth="3"/></g>
   :<g><path d="M-15 4v-19l29 5V9Z" fill="#8ba6b2" stroke="#344e60"/><path d="M-12 1v-12l23 4V5Z" fill="#20384c"/>
    {role==='metrics'||role==='collector'?<path className="work-tool" d="m-10-1 5-5 4 5 5-3 5 5" fill="none" stroke="#a6d9c1" strokeWidth="2"/>
    :role==='trace'?<g stroke="#b6abdf" fill="#b6abdf"><path d="M-8-6 0-2 8 1" fill="none"/><circle cx="-8" cy="-6" r="2"/><circle className="work-tool" cy="-2" r="2"/><circle cx="8" cy="1" r="2"/></g>
    :<path className="work-tool" d="m-9-6 4 3-4 2m8 1 7 1" fill="none" stroke="#a6d9c1" strokeWidth="2"/>}
   </g>}
  </g>
  <g className="work-hands" fill="#eadcba" stroke="#344e60"><ellipse cx="-9" cy="-9" rx="4" ry="2.5"/><ellipse cx="10" cy="-6" rx="4" ry="2.5"/></g>
 </g>
}
