export function usagePercent(used:number|null|undefined,capacity:number|undefined){
 return used==null||!Number.isFinite(used)||!capacity||!Number.isFinite(capacity)||capacity<=0?null:Math.max(0,Math.min(100,used/capacity*100))
}
