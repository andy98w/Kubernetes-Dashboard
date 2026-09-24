export type ErrorEvent={service:string;release:string;type:string;frame:string;at:string;traceId:string}
export type Issue={key:string;service:string;type:string;frame:string;firstSeen:string;lastSeen:string;count:number;release:string;traceId:string;status:'Open'|'Resolved'|'Regressed'}
// Frames must already be normalized/source-mapped by a trusted ingestion pipeline.
// Message text is deliberately excluded: it can contain personal data and unstable IDs.
export function recordError(issues:Issue[],event:ErrorEvent):Issue[]{
  const key=JSON.stringify([event.service,event.type,event.frame])
  const previous=issues.find(i=>i.key===key)
  const issue:Issue={key,service:event.service,type:event.type,frame:event.frame,firstSeen:previous?.firstSeen??event.at,lastSeen:event.at,count:(previous?.count??0)+1,release:event.release,traceId:event.traceId,status:previous?.status==='Resolved'?'Regressed':previous?.status??'Open'}
  return [issue,...issues.filter(i=>i.key!==key)].slice(0,100)
}
export function resolveIssue(issues:Issue[],key:string):Issue[]{return issues.map(i=>i.key===key?{...i,status:'Resolved'}:i)}
