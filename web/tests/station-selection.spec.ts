import {expect,test} from '@playwright/test'

test('robot selection highlights siblings and keeps workload inspection available',async({page})=>{
 await page.goto('/?fleet=1#/station')
 const pod=page.locator('[data-object="robot"]').filter({has:page.locator('title',{hasText:'illuma-api-demo-1 — Ready'})})
 await pod.focus()
 await pod.press('Enter')
 await expect(page.locator('[data-object="robot"][aria-pressed="true"]')).toHaveCount(2)
 await expect(page.getByText(/2 observed sibling pods across 2 nodes/)).toBeVisible()
 await expect(page.getByRole('button',{name:'Inspect workload ↗'})).toBeVisible()
 await page.getByRole('button',{name:'Dismiss object details'}).click()
 await expect(page.locator('[data-object="robot"][aria-pressed="true"]')).toHaveCount(0)
})
