import { expect, test } from '@playwright/test'

test('labels the offline dataset honestly', async ({ page }) => {
  await page.goto('/#/overview')
  await expect(page.locator('.connection-pill')).toHaveText('Demo data')
  await expect(page.locator('.demo-banner')).toHaveCount(0)
  await expect(page.getByText('Soon', { exact: true })).toHaveCount(0)
})

test('drills from a workload into correlated operational state', async ({ page }) => {
  await page.goto('/#/workloads')
  await page.getByText('View & lab',{exact:true}).click()
  await page.getByRole('button', { name: 'Tables', exact: true }).click()
  await page.getByRole('row').filter({ hasText: 'kubevista-api' }).click()

  const drawer = page.getByRole('dialog', { name: /kubevista-api workload details/i })
  await expect(drawer).toBeVisible()
  await expect(drawer.getByText('Container images')).toBeVisible()
  await expect(drawer.getByText('public.ecr.aws/kubevista/kubevista-api:sha-8f04c2a')).toBeVisible()
  await expect(drawer.getByText(/0 restarts/).first()).toBeVisible()
  await expect(drawer.getByText('NetworkPolicies')).toBeVisible()

  await page.keyboard.press('Escape')
  await expect(drawer).toBeHidden()
})

test('presents measured resilience evidence as an incident timeline', async ({ page }) => {
  await page.goto('/#/incidents')
  await expect(page.getByRole('heading', { name: 'Controlled API pod-loss recovery' })).toBeVisible()
  await expect(page.getByText('896/896 requests returned HTTP 200; p99 was approximately 2.78 ms')).toBeVisible()
  await expect(page.getByText('Replacement API pod became Ready in 2 seconds')).toBeVisible()
})

test('does not overflow the viewport at supported breakpoints', async ({ page }) => {
  await page.goto('/#/workloads')
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)
  expect(overflow).toBe(false)
  await expect(page.locator('.station-world')).toBeVisible()
})
