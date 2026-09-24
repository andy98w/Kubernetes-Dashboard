// Local floor axes, before the node room's 1.2 scale. Keep feet inside
// the 14-tile room and use the aisle in front of the workstation rows.
export function breakRoute(slot:number,seed:number){
 const u=58+(slot%3)*64,v=58+Math.floor(slot/3)*64
 const aisleU=u+28,frontV=210,restU=58+(seed%3)*64
 const offset=(a:number,b:number)=>({x:(a-u)-(b-v),y:((a-u)+(b-v))/2})
 return {leave:offset(aisleU,v),aisle:offset(aisleU,frontV),rest:offset(restU,frontV)}
}
