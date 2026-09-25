import { createRequire } from "node:module";
const { chromium } = createRequire(process.env.PW_ROOT + "/")("playwright");
const url = "file://" + process.argv[2];
const browser = await chromium.launch({ executablePath: "/opt/pw-browsers/chromium-1194/chrome-linux/chrome" });
for (const js of [false, true]) {
  const ctx = await browser.newContext({ javaScriptEnabled: js, viewport: { width: 800, height: 600 } });
  const page = await ctx.newPage();
  await page.goto(url);
  await page.waitForTimeout(300);
  const r = await page.$eval(".ocr-box", (b) => ({
    overflow: getComputedStyle(b).overflow, height: b.style.height, fontSize: b.style.fontSize,
    scrollHeight: b.scrollHeight, clientHeight: b.clientHeight, scrollWidth: b.scrollWidth, clientWidth: b.clientWidth, imgH: document.querySelector("img").getBoundingClientRect().height, boxBottom: b.getBoundingClientRect().bottom,
  }));
  r.clipped = r.scrollHeight > r.clientHeight + 1;
  console.log(`js=${js}`, JSON.stringify(r));
  await page.screenshot({ path: `${process.argv[3]}/js-${js}.png`, fullPage: true });
  await ctx.close();
}
await browser.close();
