import {expect,test} from '@playwright/test'

for (const [scenario,code] of [['probe','ProbeFailed'],['crashloop','CrashLoopBackOff'],['imagepull','ImagePullBackOff'],['oom','OOMKilled'],['scheduling','Unschedulable']]) {
 test(`diagnoses and recovers simulated ${scenario}`,async({page})=>{
  await page.goto('/#/workloads')
  await page.getByLabel('Failure scenario').selectOption(scenario)
  await page.getByRole('button',{name:'Inject simulated failure'}).click()
  await page.getByRole('row').filter({hasText:'probe-failure'}).click()
  const drawer=page.getByRole('dialog',{name:'probe-failure workload details'})
  await expect(drawer.getByRole('heading',{name:code,exact:true})).toBeVisible()
  await drawer.getByRole('button',{name:'Roll back deployment Review the previous owned pod template'}).click()
  await expect(drawer.getByRole('button',{name:'Review operation'})).toBeDisabled()
  await drawer.getByLabel('Reason for this operation').fill('Recover the injected failure')
  await drawer.getByRole('button',{name:'Review operation'}).click()
  await expect(drawer.getByText('Simulated review only; no Kubernetes admission checks ran.')).toBeVisible()
  await drawer.getByRole('button',{name:'Run simulation'}).click()
  await drawer.getByRole('button',{name:'Check recovery'}).click()
  await expect(drawer.getByText('Recovered',{exact:true})).toBeVisible()
  await expect(drawer.getByRole('heading',{name:code,exact:true})).toHaveCount(0)
  await expect(drawer.getByText('Simulated recovery',{exact:true})).toBeVisible()
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth)).toBe(true)
 })
}

test('a restart does not pretend to fix the broken probe',async({page})=>{
 await page.goto('/#/workloads')
 await page.getByRole('button',{name:'Inject simulated failure'}).click()
 await page.getByRole('row').filter({hasText:'probe-failure'}).click()
 const drawer=page.getByRole('dialog',{name:'probe-failure workload details'})
 await drawer.getByRole('button',{name:'Restart rollout Recreate pods through the current strategy'}).click()
 await drawer.getByLabel('Reason for this operation').fill('Check that restart retains the bad probe')
 await drawer.getByRole('button',{name:'Review operation'}).click()
 await drawer.getByRole('button',{name:'Run simulation'}).click()
 await drawer.getByRole('button',{name:'Check recovery'}).click()
 await expect(drawer.getByRole('heading',{name:'ProbeFailed',exact:true})).toBeVisible()
 await expect(drawer.getByText('Recovered',{exact:true})).toHaveCount(0)
})
