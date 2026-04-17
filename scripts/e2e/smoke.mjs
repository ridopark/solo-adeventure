#!/usr/bin/env node
// Minimal visual smoke test. Exercises landing -> first page -> choose -> ending.
// Saves screenshots to scripts/e2e/out/.
import { chromium } from "playwright";
import { mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const __filename = fileURLToPath(import.meta.url);
const outDir = resolve(dirname(__filename), "out");
mkdirSync(outDir, { recursive: true });

const url = process.env.E2E_URL ?? "http://localhost:3004";
const shot = (page, name) => page.screenshot({ path: `${outDir}/${name}.png`, fullPage: true });

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext({ viewport: { width: 1280, height: 900 } });
const page = await ctx.newPage();

page.on("pageerror", (e) => console.error("PAGE ERROR:", e.message));
page.on("console", (msg) => {
  if (msg.type() === "error") console.error("BROWSER CONSOLE:", msg.text());
});

console.log("-> landing");
await page.goto(url, { waitUntil: "networkidle" });
await shot(page, "01-landing");

console.log("-> submit topic");
await page.getByPlaceholder(/lighthouse/i).fill("a lighthouse keeper in 1912");
await page.getByRole("button", { name: /begin/i }).click();

await page.waitForURL(/\/story\//, { timeout: 20000 });
await page.waitForSelector("article", { timeout: 20000 });
await page.waitForTimeout(500);
await shot(page, "02-first-page");

for (let i = 0; i < 3; i++) {
  console.log(`-> choice ${i + 1}`);
  const choice = page.getByRole("button").filter({ hasText: /step|turn|follow/i }).first();
  const endCard = page.getByText(/ending:/i);
  await Promise.race([
    choice.click(),
    endCard.waitFor({ timeout: 1000 }).catch(() => {}),
  ]);
  await page.waitForTimeout(800);
  await shot(page, `03-after-choice-${i + 1}`);
  if (await page.getByText(/start a new adventure/i).isVisible().catch(() => false)) break;
}

console.log("-> reach ending");
await page.waitForSelector("aside", { timeout: 20000 });
await shot(page, "04-ending");

console.log(`ok -- screenshots in ${outDir}`);
await browser.close();
