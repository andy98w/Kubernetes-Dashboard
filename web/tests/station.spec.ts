import {test,expect} from '@playwright/test'

test('DaemonSets identify siblings without inventing node coverage',async({page})=>{
  await page.goto('/#/station')
  const bots=page.locator('[data-object="robot"]')
  const agent=page.getByRole('button',{name:'otel-agent-7d8c9b-x2m4p, namespace observability, DaemonSet: Ready. Open otel-agent',exact:true})
  await expect(page.locator('.station-daemon-badge')).toHaveCount(2)
  await agent.hover()
  await expect(bots.filter({has:page.locator('.station-daemon-badge')})).toHaveCount(2)
  await expect(page.locator('[data-object="robot"][opacity="1"]')).toHaveCount(2)
  await expect(page.locator('.station-inspect')).toContainText('2/2 ready (reported); pods loaded on 2 nodes')
  await expect(page.locator('.station-room')).toHaveCount(2)
  const scales=await page.locator('.station-room').evaluateAll(rooms=>rooms.map(room=>room.getAttribute('transform')?.match(/scale\(([^)]+)\)/)?.[1]))
  expect(new Set(scales).size).toBe(1)
  await page.getByRole('button',{name:'All',exact:true}).hover()
  await expect(page.locator('[data-object="robot"][opacity="0.18"]')).toHaveCount(0)
  await agent.focus()
  await expect(page.locator('[data-object="robot"][opacity="1"]')).toHaveCount(2)
})

for(const path of ['/', '/#/overview', '/#/station', '/#/workloads']) {
  test(`station is the default at ${path}`,async({page})=>{
    await page.goto(path)
    await expect(page.locator('.station-world')).toBeVisible()
    await page.getByText('View & lab',{exact:true}).click()
    await expect(page.getByRole('button',{name:'Station',exact:true})).toHaveAttribute('aria-pressed','true')
    await expect(page.getByRole('table')).toHaveCount(0)
  })
}

test('inspection preserves map filters and closes with Escape',async({page})=>{
  await page.goto('/')
  await page.getByText('Map options',{exact:true}).click()
  await page.getByRole('button',{name:'kubevista',exact:true}).click()
  await page.getByRole('navigation',{name:'Station tools'}).getByRole('button',{name:'Network',exact:true}).click()
  const panel=page.getByRole('dialog',{name:'Network inspection'})
  await expect(panel).toBeVisible()
  await expect(panel.getByRole('heading',{name:'Network',exact:true})).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(panel).toHaveCount(0)
  await expect(page.getByRole('button',{name:'kubevista',exact:true})).toHaveAttribute('aria-pressed','true')
  await expect(page.locator('.station-world')).toBeVisible()
})

test('deep-linked telemetry is an inspection over the station',async({page})=>{
  await page.goto('/#/observability')
  const panel=page.getByRole('dialog',{name:'Observability inspection'})
  await expect(panel).toBeVisible()
  await panel.getByRole('button',{name:'Return to station'}).click()
  await expect(page).toHaveURL(/#\/station$/)
  await expect(page.locator('.station-world')).toBeVisible()
})
