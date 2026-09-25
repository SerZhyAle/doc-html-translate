import { createRequire } from "node:module";
import http from "node:http";
import { readFile } from "node:fs/promises";
import { join, extname } from "node:path";
const { chromium } = createRequire(process.env.PW_ROOT + "/")("playwright");
const ext = process.argv[2], pagePath = process.argv[3];
const types = { ".js": "text/javascript", ".css": "text/css", ".html": "text/html" };
const server = http.createServer(async (req, res) => {
  try {
    const p = req.url === "/" ? pagePath : join(ext, decodeURIComponent(req.url));
    const body = await readFile(p);
    res.writeHead(200, { "content-type": types[extname(p)] || "application/octet-stream" });
    res.end(body);
  } catch { res.writeHead(404); res.end(); }
}).listen(0);
const port = server.address().port;
const browser = await chromium.launch({ executablePath: "/opt/pw-browsers/chromium-1194/chrome-linux/chrome" });
const page = await browser.newPage({ viewport: { width: 1400, height: 1800 }, deviceScaleFactor: 1 });
await page.goto(`http://localhost:${port}/`);
await page.evaluate(async () => {
  const c = document.createElement("canvas"); c.width = 400; c.height = 200;
  const g = c.getContext("2d"); g.fillStyle = "rgb(255,0,0)"; g.fillRect(0, 0, 200, 200); g.fillStyle = "rgb(0,0,255)"; g.fillRect(200, 0, 200, 200);
  const url = c.toDataURL();
  await Promise.all([...document.images].map((i) => new Promise((ok) => { i.onload = ok; i.src = url; })));
});
await page.addScriptTag({ url: "/src/page-agent.js" });
// Ground truth: where the red is, per picture, before any plate exists.
const shot = async () => page.screenshot({ fullPage: true });
const rects = await page.evaluate(() => Object.fromEntries([...document.images].map((i) => {
  const r = i.getBoundingClientRect(); return [i.id, { x0: r.left, y0: r.top, x1: r.right, y1: r.bottom }];
})));
const before = (await shot()).toString("base64");
const ids = await page.evaluate(() => new Promise((ok) => {
  for (const f of __listeners) f({ dht: "page-ocr", t: "collect" }, {}, (r) => ok(r));
}));
const idOf = await page.evaluate(() => Object.fromEntries([...document.images].map((i) => [i.id, i.id])));
await page.evaluate((images) => {
  const spec = { text: "x", left: "0%", top: "0%", width: "50%", minHeight: "100%", fontSize: "1cqw", bg: "rgb(0,255,0)", ink: "rgb(0,255,0)" };
  for (const im of images) for (const f of __listeners) f({ dht: "page-ocr", t: "plates", id: im.id, specs: [spec] }, {}, () => {});
}, ids.images);
await page.waitForTimeout(800);
const after = (await shot()).toString("base64");
const unplaced = await page.evaluate(() => [...document.querySelectorAll(".dht-ocr-layer")].map((l) => l.dataset.unplaced === "1"));
const helper = await browser.newPage();
const out = await helper.evaluate(async ({ before, after, rects }) => {
  const load = async (b64) => {
    const im = new Image(); im.src = "data:image/png;base64," + b64; await im.decode();
    const c = document.createElement("canvas"); c.width = im.width; c.height = im.height;
    const g = c.getContext("2d"); g.drawImage(im, 0, 0); return g.getImageData(0, 0, im.width, im.height);
  };
  const B = await load(before), A = await load(after);
  const px = (img, x, y) => { const i = (y * img.width + x) * 4; return [img.data[i], img.data[i + 1], img.data[i + 2]]; };
  const isRed = ([r, g, b]) => r > 200 && g < 60 && b < 60;
  const isBlue = ([r, g, b]) => b > 200 && r < 60 && g < 60;
  const isGreen = ([r, g, b]) => g > 200 && r < 60 && b < 60;
  const res = {};
  for (const [name, r] of Object.entries(rects)) {
    const pad = 40;
    let redBefore = 0, redAfter = 0, greenOutside = 0, greenInside = 0, blueLost = 0;
    for (let y = Math.max(0, Math.floor(r.y0) - pad); y < Math.min(A.height, Math.ceil(r.y1) + pad); y++) {
      for (let x = Math.max(0, Math.floor(r.x0) - pad); x < Math.min(A.width, Math.ceil(r.x1) + pad); x++) {
        const wasRed = isRed(px(B, x, y));
        if (wasRed) redBefore++;
        if (isRed(px(A, x, y))) redAfter++;
        if (isGreen(px(A, x, y))) { if (wasRed) greenInside++; else greenOutside++; }
        if (isBlue(px(B, x, y)) && !isBlue(px(A, x, y))) blueLost++;
      }
    }
    res[name] = { redBefore, redAfter, greenInside, greenOutside, blueLost };
  }
  return res;
}, { before, after, rects });
for (const [n, name] of Object.keys(rects).entries()) console.log(name.padEnd(10), JSON.stringify({ ...out[name] }));
await browser.close(); server.close();
