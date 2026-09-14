"""Bounded blue/green release lab; never targets AWS or the current kube context."""
import argparse
import json
import math
import subprocess
import time
from pathlib import Path

NAMESPACE = 'kubevista-release-lab'
ALLOWED_CONTEXTS = {'kind-kubevista-release-ci', 'kind-kubevista-release-lab'}
SAMPLE = r'''
import json,time,urllib.request,urllib.error
samples=[]
for _ in range(20):
 start=time.monotonic()
 try:
  with urllib.request.urlopen('http://TARGET:8080/',timeout=2) as response: status=response.status; body=response.read().decode()
 except urllib.error.HTTPError as error: status=error.code; body=error.read().decode()
 except Exception: status=0; body=''
 samples.append({'status':status,'seconds':time.monotonic()-start,'body':body})
print(json.dumps(samples))
'''

def evaluate(samples, baseline=None, expected_body=None):
    if len(samples) != 20 or any(not isinstance(s.get('seconds'), (float,int)) or not math.isfinite(s['seconds']) or s['seconds'] < 0 for s in samples):
        return {'pass': False, 'reason': 'incomplete or invalid measurements'}
    if expected_body is not None and any(s.get('body') != expected_body for s in samples):
        return {'pass':False,'reason':'unexpected serving revision','samples':len(samples)}
    errors = sum(s.get('status') != 200 for s in samples) / len(samples)
    p95 = sorted(s['seconds'] for s in samples)[18]
    limit = min(.25, max(.1, baseline['p95Seconds'] * 3)) if baseline else .25
    return {'pass': errors <= .05 and p95 <= limit, 'errorRate': errors, 'p95Seconds': p95, 'latencyLimitSeconds': limit, 'samples':len(samples)}

def route_patch(observed, target):
    return [
        {'op':'test','path':'/metadata/resourceVersion','value':observed['metadata']['resourceVersion']},
        {'op':'test','path':'/spec/selector','value':observed['spec']['selector']},
        {'op':'replace','path':'/spec/selector','value':{'app':'release-fixture','slot':target}},
    ]

class Lab:
    def __init__(self, context, output):
        if context not in ALLOWED_CONTEXTS:
            raise ValueError('Only dedicated release-lab kind contexts are allowed')
        self.context = context
        self.output = Path(output)
        self.output.mkdir(parents=True, exist_ok=True)
        self.events = []
    def kube(self, *args, data=None):
        result = subprocess.run(['kubectl','--context',self.context,'-n',NAMESPACE,*args],input=data,text=True,capture_output=True,timeout=180)
        if result.returncode:
            raise RuntimeError(result.stderr.strip())
        return result.stdout
    def apply(self, obj):
        self.kube('apply','-f','-',data=json.dumps(obj))
    def deployment(self, slot, mode):
        return {'apiVersion':'apps/v1','kind':'Deployment','metadata':{'name':slot},'spec':{'replicas':1,'selector':{'matchLabels':{'app':'release-fixture','slot':slot}},'template':{'metadata':{'labels':{'app':'release-fixture','slot':slot}},'spec':{'containers':[{'name':'app','image':'kubevista-release:lab','imagePullPolicy':'Never','env':[{'name':'MODE','value':mode},{'name':'SLOT','value':slot}],'ports':[{'containerPort':8080}],'readinessProbe':{'httpGet':{'path':'/healthz','port':8080},'periodSeconds':1},'resources':{'requests':{'cpu':'25m','memory':'32Mi'},'limits':{'cpu':'250m','memory':'64Mi'}}}]}}}}
    def service(self, name, slot):
        return {'apiVersion':'v1','kind':'Service','metadata':{'name':name},'spec':{'selector':{'app':'release-fixture','slot':slot},'ports':[{'port':8080,'targetPort':8080}]}}
    def setup(self):
        # Refuse to overwrite a pre-existing namespace.
        self.kube('create','namespace',NAMESPACE)
        for slot in ('stable','candidate'):
            self.apply(self.deployment(slot,'healthy'))
            self.apply(self.service(slot,slot))
        self.apply(self.service('traffic','stable'))
        self.apply({'apiVersion':'v1','kind':'Pod','metadata':{'name':'sampler'},'spec':{'containers':[{'name':'sampler','image':'kubevista-release:lab','imagePullPolicy':'Never','command':['python','-c','import time;time.sleep(3600)']}],'restartPolicy':'Never'}})
        self.kube('wait','pod/sampler','--for=condition=Ready','--timeout=120s')
        self.kube('rollout','status','deployment/stable','--timeout=120s')
    def sample(self, service):
        return json.loads(self.kube('exec','sampler','--','python','-c',SAMPLE.replace('TARGET',service)))
    def observe_route(self):
        return json.loads(self.kube('get','service','traffic','-o','json'))
    def switch(self, observed, slot):
        raw=self.kube('patch','service','traffic','--type=json','-p',json.dumps(route_patch(observed,slot)),'-o','json')
        changed=json.loads(raw)
        return changed
    def wait_route(self, slot, service='traffic'):
        # Wait for endpoint publication before collecting application measurements.
        expected=json.loads(self.kube('get','pods','-l',f'app=release-fixture,slot={slot}','-o','json'))['items']
        ips={p['status'].get('podIP') for p in expected if not p['metadata'].get('deletionTimestamp')}
        for _ in range(40):
            endpoints=json.loads(self.kube('get','endpoints',service,'-o','json'))
            actual={a['ip'] for subset in endpoints.get('subsets',[]) for a in subset.get('addresses',[])}
            if ips and actual == ips:
                time.sleep(.5)
                return
            time.sleep(.25)
        raise RuntimeError('Traffic endpoints did not converge')
    def scenario(self, mode):
        record={'mode':mode,'environment':'single-node kind; synthetic traffic','sampleSize':20}
        self.events.append(record)
        original=self.observe_route()
        if original['spec']['selector'] != {'app':'release-fixture','slot':'stable'}:
            raise RuntimeError('Expected stable traffic before release')
        self.apply(self.deployment('candidate',mode))
        self.kube('rollout','status','deployment/candidate','--timeout=120s')
        self.wait_route('candidate','candidate')
        self.wait_route('stable','stable')
        record['baseline']=evaluate(self.sample('stable'),expected_body='stable:healthy')
        if not record['baseline']['pass']:
            raise RuntimeError('Baseline unhealthy; refusing promotion')
        record['candidate']=evaluate(self.sample('candidate'),record['baseline'],f'candidate:{mode}')
        if not record['candidate']['pass']:
            record['outcome']='rejected-before-promotion'
        else:
            # Record the guarded mutation before waiting for propagation so any
            # later measurement error also triggers rollback.
            promoted=None
            try:
                promoted=self.switch(original,'candidate')
                self.wait_route('candidate')
                record['postPromotion']=evaluate(self.sample('traffic'),record['baseline'],f'candidate:{mode}')
                record['outcome']='promoted' if record['postPromotion']['pass'] else 'rolled-back'
            except Exception as error:
                record['outcome']='release-error'
                record['error']=str(error)
                raise
            finally:
                if promoted is not None:
                    self.switch(promoted,'stable')
                    self.wait_route('stable')
                    record['routeRestored']=True
        record['recovery']=evaluate(self.sample('traffic'),record['baseline'],'stable:healthy')
        if not record['recovery']['pass']:
            raise RuntimeError('Stable traffic recovery failed')
        # Success is reset to stable to isolate the next fixture, not because it failed.
        record['healthyReleaseResetForNextTest']=record['outcome']=='promoted'
    def run(self):
        try:
            self.setup()
            for mode in ('healthy','errors','slow','late-failure'):
                self.scenario(mode)
            observed=self.observe_route()
            self.kube('annotate','service','traffic','release-lab/concurrent-change=true','--overwrite')
            try:
                self.switch(observed,'candidate')
            except RuntimeError:
                if self.observe_route()['spec']['selector'] != observed['spec']['selector']:
                    raise RuntimeError('Stale mutation changed traffic')
                (self.output/'stale-release.json').write_text(json.dumps({'staleMutationRejected':True}))
            else:
                raise RuntimeError('Stale release was unexpectedly accepted')
            expected=['promoted','rejected-before-promotion','rejected-before-promotion','rolled-back']
            if [e['outcome'] for e in self.events] != expected:
                raise RuntimeError('Unexpected release decisions')
        finally:
            (self.output/'results.json').write_text(json.dumps(self.events,indent=2)+'\n')

if __name__=='__main__':
    parser=argparse.ArgumentParser()
    parser.add_argument('--context',required=True)
    parser.add_argument('--output',default='outputs/release-lab')
    args=parser.parse_args()
    Lab(args.context,args.output).run()
