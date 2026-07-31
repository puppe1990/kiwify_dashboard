/**
 * Capture README screenshots against a local server.
 * Usage: cais server & node scripts/take-screenshots.mjs
 * Requires: npm i -D playwright && npx playwright install chromium
 */
import { chromium } from "playwright";
import { mkdirSync } from "fs";
import { dirname, join } from "path";
import { fileURLToPath } from "url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const base = process.env.APP_URL || "http://127.0.0.1:8080";
const out = join(root, "docs/screenshots");
mkdirSync(out, { recursive: true });

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

async function shot(name) {
  const path = join(out, `${name}.png`);
  await page.screenshot({ path, fullPage: false });
  console.log("wrote", path, "→", page.url());
}

await page.goto(`${base}/login`, { waitUntil: "networkidle" });
await page.waitForTimeout(400);
await shot("01-login");

await page.locator('input[type="email"]').fill("demo@example.com");
await page.locator('input[type="password"]').fill("password");
await page.getByRole("button", { name: /log in/i }).click();
await page.waitForLoadState("networkidle");
await page.waitForTimeout(800);
await shot("02-after-login");

const paths = [
  ["/setup", "03-setup"],
  ["/dashboard", "04-dashboard"],
  ["/sales", "05-sales"],
  ["/settings", "06-settings"],
  ["/webhooks", "07-webhooks"],
  ["/account", "08-account"],
];

for (const [path, name] of paths) {
  await page.goto(`${base}${path}`, { waitUntil: "networkidle" });
  await page.waitForTimeout(700);
  await shot(name);
}

await browser.close();
console.log("done");
