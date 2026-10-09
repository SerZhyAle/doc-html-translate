// make-demo-assets.mjs - the material the demo capture (make-demo.mjs) works from: a public-domain
// sample book, an original comic page, and the three pages of the demo's own that are not the
// product (the brand card, the console and the closing card).
//
// Everything here is authored for the demo, so a run needs no real book and no download, and the
// same input draws the same pixels. The only things on screen that are not drawn here are the
// product's own output: the converted pages and the console lines of the real binary.

// Site tokens, assets/sza-kit.css: --bg, --text, --text-2, --acc, --gold. The plate blue is the
// mark's own, as in tools/store/make-logos.ps1.
export const PALETTE = {
  bg: "#0a0f0a", text: "#f1f5ee", text2: "#c3ccbc", muted: "#94a08c",
  acc: "#3fb950", gold: "#e3b341", plate: "#1e3a8a",
};

export const BOOK_FILE = "Treasure Island.md";
export const COMIC_FILE = "Comic page.png";
export const COMIC_SIZE = { width: 1600, height: 880 };
// The lettering face: a bold sans that ships with Windows, so the page renders the same on any machine of
// this project. The bundled English OCR data has to read it, which is the test of the scene.
export const COMIC_FONT = "'Trebuchet MS', Arial, sans-serif";

// ---- the sample book ---------------------------------------------------------------------

// Treasure Island (R. L. Stevenson, 1883) is in the public domain. The file carries the book's real
// three-level structure (title, part, chapter) so the contents panel shows a genuine multi-level
// table; a chapter holds only its opening line where the full text is not needed on screen.
// Em dashes of the original are written as a spaced hyphen, the house style.
const PARTS = [
  ["Part One - The Old Buccaneer", [
    ['1. The Old Sea-dog at the "Admiral Benbow"', [
      "Squire Trelawney, Dr. Livesey, and the rest of these gentlemen having asked me to write down the whole particulars about Treasure Island, from the beginning to the end, keeping nothing back but the bearings of the island, and that only because there is still treasure not yet lifted, I take up my pen in the year of grace 17\\_\\_ and go back to the time when my father kept the Admiral Benbow inn and the brown old seaman with the sabre cut first took up his lodging under our roof.",
      "I remember him as if it were yesterday, as he came plodding to the inn door, his sea-chest following behind him in a hand-barrow - a tall, strong, heavy, nut-brown man, his tarry pigtail falling over the shoulders of his soiled blue coat, his hands ragged and scarred, with black, broken nails, and the sabre cut across one cheek, a dirty, livid white.",
      "I remember him looking round the cover and whistling to himself as he did so, and then breaking out in that old sea-song that he sang so often afterwards: \"Fifteen men on the dead man's chest - Yo-ho-ho, and a bottle of rum!\" in the high, old tottering voice that seemed to have been tuned and broken at the capstan bars.",
    ]],
    ["2. Black Dog Appears and Disappears", [
      "It was not very long after this that there occurred the first of the mysterious events that rid us at last of the captain, though not, as you will see, of his affairs.",
    ]],
    ["3. The Black Spot", [
      "About noon I stopped at the captain's door with some cooling drinks and medicines.",
    ]],
    ["4. The Sea-chest", [
      "I lost no time, of course, in telling my mother all that I knew, and perhaps should have told her long before, how we were now in a difficult and dangerous position.",
    ]],
    ["5. The Last of the Blind Man", []],
    ["6. The Captain's Papers", [
      "We rode hard all the way till we drew up before Dr. Livesey's door.",
    ]],
  ]],
  ["Part Two - The Sea-cook", [
    ["7. I Go to Bristol", [
      "It was longer than the squire imagined ere we were ready for the sea, and none of our first plans - not even Dr. Livesey's, of keeping me beside him - could be carried out as we intended.",
    ]],
    ['8. At the Sign of the "Spy-glass"', [
      "When I had done breakfasting the squire gave me a note addressed to John Silver, at the sign of the Spy-glass, and told me I should easily find the place by following the line of the docks and keeping a bright lookout for a little tavern with a large brass telescope for sign.",
    ]],
    ["9. Powder and Arms", [
      "The Hispaniola lay some way out, and we went under the figureheads and round the sterns of many other ships, and their cables sometimes grated underneath our keel, and sometimes swung above us.",
    ]],
    ["10. The Voyage", [
      "All that night we were in a great bustle getting things stowed in their place, and boatfuls of the squire's friends, Mr. Blandly and the like, coming off to wish him a good voyage and a safe return.",
    ]],
  ]],
  ["Part Three - My Shore Adventure", [
    ["13. How My Shore Adventure Began", [
      "The appearance of the island when I came on deck next morning was altogether changed.",
    ]],
    ["14. The First Blow", [
      "I was so pleased at having given the slip to Long John that I began to enjoy myself and look around me with some interest in the strange land that I was in.",
    ]],
    ["15. The Man of the Island", [
      "From the side of the hill, which was here steep and stony, a spout of gravel was dislodged and fell rattling and bounding through the trees.",
    ]],
  ]],
];

export function sampleBookMarkdown() {
  const out = ["# Treasure Island", "", "_by Robert Louis Stevenson_", ""];
  for (const [part, chapters] of PARTS) {
    out.push(`## ${part}`, "");
    for (const [title, paras] of chapters) {
      out.push(`### ${title}`, "");
      for (const p of paras) out.push(p, "");
    }
  }
  return out.join("\n");
}

// ---- the comic page ----------------------------------------------------------------------

// An original four-panel page: flat shapes, no borrowed art, English lettering that the bundled
// English OCR data reads. The only text in the picture is lettering, so every plate the converter
// draws is over a balloon or a caption box. Lettering is in capitals, the way comic lettering is.
const INK = "#171717";

function girl() {
  const arm = (d) => `<path d="${d}" fill="none" stroke="${INK}" stroke-width="30" stroke-linecap="round"/>` +
    `<path d="${d}" fill="none" stroke="#2a9d8f" stroke-width="22" stroke-linecap="round"/>`;
  return `<g id="girl" stroke-linejoin="round">
    <rect x="-32" y="-112" width="24" height="104" fill="#264653" stroke="${INK}" stroke-width="4"/>
    <rect x="8" y="-112" width="24" height="104" fill="#264653" stroke="${INK}" stroke-width="4"/>
    <ellipse cx="-20" cy="-8" rx="24" ry="11" fill="${INK}"/><ellipse cx="20" cy="-8" rx="24" ry="11" fill="${INK}"/>
    <path d="M-52,-250 Q-52,-264 -38,-264 L38,-264 Q52,-264 52,-250 L62,-100 L-62,-100 Z" fill="#2a9d8f" stroke="${INK}" stroke-width="4"/>
    <rect x="-10" y="-276" width="20" height="18" fill="#f4c7a1" stroke="${INK}" stroke-width="4"/>
    <circle cx="0" cy="-314" r="48" fill="#f4c7a1" stroke="${INK}" stroke-width="4"/>
    <path d="M-52,-322 Q-56,-380 0,-380 Q56,-380 52,-322 Q42,-352 0,-352 Q-42,-352 -52,-322 Z" fill="#5b3a29" stroke="${INK}" stroke-width="4"/>
    <circle cx="40" cy="-372" r="21" fill="#5b3a29" stroke="${INK}" stroke-width="4"/>
    <circle cx="-17" cy="-316" r="5.5" fill="${INK}"/><circle cx="17" cy="-316" r="5.5" fill="${INK}"/>
    <circle cx="-29" cy="-300" r="8" fill="#ef8f8f" opacity=".6"/><circle cx="29" cy="-300" r="8" fill="#ef8f8f" opacity=".6"/>
    <path d="M-14,-296 Q0,-282 14,-296" fill="none" stroke="${INK}" stroke-width="3" stroke-linecap="round"/>
    ${arm("M-46,-240 Q-74,-186 -16,-168")}${arm("M46,-240 Q74,-186 16,-168")}
    <g transform="translate(0,-182) rotate(-4)">
      <rect x="-48" y="-34" width="96" height="68" rx="5" fill="#b5412b" stroke="${INK}" stroke-width="4"/>
      <rect x="-48" y="-34" width="15" height="68" fill="#8a2f1d" stroke="${INK}" stroke-width="4"/>
      <rect x="-20" y="-16" width="56" height="9" fill="#e3b341"/>
    </g>
  </g>`;
}

const FONT = 40, LEAD = 48;

function letteredLines(cx, cy, lines) {
  const y0 = cy - ((lines.length - 1) * LEAD) / 2 + 14;
  const tspans = lines.map((l, i) => `<tspan x="${cx}" y="${y0 + i * LEAD}">${l}</tspan>`).join("");
  return `<text text-anchor="middle" font-size="${FONT}" font-weight="400" fill="${INK}">${tspans}</text>`;
}

// A balloon is a rounded box with a tail aimed at the speaker. The tail's base sits on a straight
// stretch of one edge (side and position `at`, so never on a rounded corner); the seam is then
// painted over so the box outline does not cross the join.
function balloon(cx, cy, w, h, lines, tail) {
  const hw = w / 2, hh = h / 2, k = 22;
  const ex = tail.side === "left" ? cx - hw : cx + hw;
  const [b1, b2] = [[ex, tail.at - k], [ex, tail.at + k]];
  const pts = (a) => a.map((p) => p.join(",")).join(" ");
  const tailSvg = `<polygon points="${pts([[tail.tip.x, tail.tip.y], b1, b2])}" fill="#fff" stroke="${INK}" stroke-width="5" stroke-linejoin="round"/>`;
  const box = `<rect x="${cx - hw}" y="${cy - hh}" width="${w}" height="${h}" rx="44" fill="#fff" stroke="${INK}" stroke-width="5"/>`;
  const patch = `<rect x="${ex + (tail.side === "left" ? -4 : -7)}" y="${tail.at - k + 4}" width="11" height="${2 * k - 8}" fill="#fff"/>`;
  return `${tailSvg}${box}${patch}${letteredLines(cx, cy, lines)}`;
}

// A narration box: the rectangular caption comics use for the narrator's line.
function narration(x, y, w, h, lines) {
  return `<rect x="${x}" y="${y}" width="${w}" height="${h}" fill="#ffd166" stroke="${INK}" stroke-width="5"/>${letteredLines(x + w / 2, y + h / 2, lines)}`;
}

function panel(x, y, w, h, id, art) {
  return `<clipPath id="${id}"><rect width="${w}" height="${h}"/></clipPath>
  <g transform="translate(${x},${y})"><g clip-path="url(#${id})">${art}</g><rect width="${w}" height="${h}" fill="none" stroke="${INK}" stroke-width="7"/></g>`;
}

const sprite = (x, y, s = 1) => `<use href="#girl" transform="translate(${x},${y}) scale(${s})"/>`;
const bars = (x, y, w, n, fill, step = 18) => Array.from({ length: n }, (_, i) =>
  `<rect x="${x}" y="${y + i * step}" width="${i === n - 1 ? w * 0.6 : w}" height="9" rx="4" fill="${fill}"/>`).join("");

// The four panels in page coordinates: A (30,30) 900x400, B (950,30) 620x400, C (30,450) 620x400,
// D (670,450) 900x400. Art is drawn in panel coordinates; lettering is drawn over the whole page.
export function comicHtml() {
  const A = `<rect width="900" height="400" fill="#bde0fe"/>
    <circle cx="840" cy="70" r="42" fill="#ffd166" stroke="${INK}" stroke-width="5"/>
    <g fill="#fff" stroke="${INK}" stroke-width="5"><ellipse cx="110" cy="70" rx="72" ry="28"/><ellipse cx="164" cy="52" rx="50" ry="26"/></g>
    <rect y="300" width="900" height="100" fill="#90be6d"/><rect y="360" width="900" height="40" fill="#c9ada7" stroke="${INK}" stroke-width="4"/>
    <rect x="670" y="214" width="210" height="104" fill="#e07a5f" stroke="${INK}" stroke-width="5"/>
    <path d="M656,218 L775,172 L894,218 Z" fill="#81405a" stroke="${INK}" stroke-width="5" stroke-linejoin="round"/>
    <rect x="768" y="254" width="48" height="64" fill="#3d405b" stroke="${INK}" stroke-width="4"/>
    <rect x="690" y="240" width="44" height="42" fill="#f4f1de" stroke="${INK}" stroke-width="4"/><rect x="836" y="240" width="32" height="42" fill="#f4f1de" stroke="${INK}" stroke-width="4"/>
    ${sprite(170, 386, 0.9)}`;
  const B = `<rect width="620" height="400" fill="#f6bd60"/><rect y="350" width="620" height="50" fill="#84a59d"/>
    <rect x="350" y="296" width="250" height="22" fill="#7f5539" stroke="${INK}" stroke-width="4"/><rect x="368" y="318" width="16" height="36" fill="#7f5539"/><rect x="566" y="318" width="16" height="36" fill="#7f5539"/>
    <rect x="384" y="276" width="170" height="20" rx="4" fill="#adb5bd" stroke="${INK}" stroke-width="4"/>
    <rect x="398" y="194" width="142" height="82" rx="4" fill="#343a40" stroke="${INK}" stroke-width="4"/><rect x="410" y="205" width="118" height="60" fill="#f8f9fa"/>${bars(420, 214, 98, 3, "#adb5bd", 16)}
    ${sprite(150, 390, 0.48)}`;
  const C = `<rect width="620" height="400" fill="#cdb4db"/>
    <rect x="190" y="206" width="400" height="156" rx="10" fill="#343a40" stroke="${INK}" stroke-width="5"/><rect x="206" y="220" width="368" height="128" fill="#f8f9fa"/>
    ${bars(224, 238, 124, 5, "#adb5bd", 20)}${bars(418, 238, 136, 5, "#52b788", 20)}
    <path d="M364,284 H396 M384,270 L398,284 L384,298" fill="none" stroke="${INK}" stroke-width="6" stroke-linecap="round" stroke-linejoin="round"/>
    <rect y="362" width="620" height="38" fill="#a2d2ff"/>${sprite(100, 390, 0.48)}`;
  const D = `<rect width="900" height="400" fill="#355070"/><rect y="350" width="900" height="50" fill="#6d597a"/>
    <circle cx="660" cy="64" r="30" fill="#f4f1de" stroke="${INK}" stroke-width="4"/>
    <circle cx="590" cy="120" r="3" fill="#e5e5e5"/><circle cx="560" cy="40" r="4" fill="#e5e5e5"/><circle cx="470" cy="130" r="3" fill="#e5e5e5"/>
    <rect x="40" y="290" width="26" height="62" fill="#7f5539" stroke="${INK}" stroke-width="4"/><path d="M22,292 L84,292 L72,238 L34,238 Z" fill="#ffd166" stroke="${INK}" stroke-width="4" stroke-linejoin="round"/>
    ${sprite(740, 386, 0.85)}`;

  // Panel D's head: x 670+740, tail tip just left of it.
  const lettering = [
    balloon(650, 165, 520, 240, ["I FOUND A BOOK", "ABOUT TREASURE.", "BUT I CAN'T READ", "A WORD OF IT!"], { side: "left", at: 185, tip: { x: 252, y: 150 } }),
    narration(970, 50, 580, 170, ["NO PROBLEM. FIRST,", "TURN THE BOOK INTO", "A WEB PAGE."]),
    narration(50, 470, 580, 170, ["THEN MY BROWSER", "CAN TRANSLATE IT", "FOR ME."]),
    narration(690, 470, 560, 76, ["LATER THAT EVENING"]),
    balloon(1000, 700, 540, 160, ["NOW I CAN READ", "THE WHOLE STORY!"], { side: "right", at: 700, tip: { x: 1358, y: 606 } }),
  ].join("\n    ");

  return `<!doctype html><meta charset="utf-8"><style>html,body{margin:0;background:#fffdf5}svg{display:block}</style>
<svg xmlns="http://www.w3.org/2000/svg" width="${COMIC_SIZE.width}" height="${COMIC_SIZE.height}" viewBox="0 0 ${COMIC_SIZE.width} ${COMIC_SIZE.height}" font-family="${COMIC_FONT}">
<defs>${girl()}</defs><rect width="100%" height="100%" fill="#fffdf5"/>
${panel(30, 30, 900, 400, "pa", A)}${panel(950, 30, 620, 400, "pb", B)}${panel(30, 450, 620, 400, "pc", C)}${panel(670, 450, 900, 400, "pd", D)}
    ${lettering}</svg>`;
}

// ---- the demo's own pages ----------------------------------------------------------------

const BASE_CSS = `
  *{box-sizing:border-box}html,body{margin:0;width:1280px;height:720px;overflow:hidden}
  body{background:${PALETTE.bg};color:${PALETTE.text};font-family:"Segoe UI",system-ui,sans-serif;position:relative}
  body::before{content:"";position:absolute;inset:0;
    background:radial-gradient(ellipse 75% 110% at 5% 0%,rgba(63,185,80,.33),transparent 70%),
               radial-gradient(ellipse 60% 90% at 100% 100%,rgba(227,179,65,.22),transparent 70%)}
  .plate{position:relative;flex:none;background:${PALETTE.plate};border:3px solid rgba(255,255,255,.27);
    display:flex;flex-direction:column;align-items:center;justify-content:center;color:#fff;font-weight:700;line-height:.98}
  .gold{color:${PALETTE.gold}}
`;

// The mark is the one in tools/store/make-logos.ps1: white bold DOC over HTML on the brand blue.
function plate(size) {
  const fs = Math.round(size * 0.236);
  return `<div class="plate" style="width:${size}px;height:${size}px;border-radius:${Math.round(size * 0.157)}px;font-size:${fs}px"><span>DOC</span><span>HTML</span></div>`;
}

export function introHtml() {
  return `<!doctype html><meta charset="utf-8"><title>demo intro</title><style>${BASE_CSS}
  main{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;gap:64px;padding:0 90px}
  h1{margin:0;font-size:84px;line-height:1.05;font-weight:700;letter-spacing:-.5px;white-space:nowrap}
  .rule{width:96px;height:6px;border-radius:3px;background:${PALETTE.acc};margin:16px 0 28px}
  p{margin:0;font-size:38px;line-height:1.3;font-weight:600;color:${PALETTE.text2};max-width:700px}
</style><main>${plate(280)}<div><h1>doc-html-translate</h1><div class="rule"></div>
<p>Turn any book, document or comic into a local web page.</p></div></main>`;
}

// The closing card states the free path; it does not draw the browser's own translate bar, which is
// native browser chrome and cannot be captured from a headless page.
export const CLOSING_LINE = "Open the page in Chrome or Edge, then use the browser's own Translate page - free, no key, no account.";
export const CLOSING_ACCENT = "free, no key, no account";

export function closingHtml() {
  const at = CLOSING_LINE.indexOf(CLOSING_ACCENT);
  const line = `${CLOSING_LINE.slice(0, at)}<span class="gold">${CLOSING_ACCENT}</span>${CLOSING_LINE.slice(at + CLOSING_ACCENT.length)}`;
  return `<!doctype html><meta charset="utf-8"><title>demo closing</title><style>${BASE_CSS}
  main{position:absolute;inset:0;display:flex;flex-direction:column;align-items:center;justify-content:center;padding:0 110px;text-align:center}
  .brand{display:flex;align-items:center;gap:22px;margin-bottom:46px}
  .brand b{font-size:44px;letter-spacing:-.3px}
  p{margin:0;font-size:56px;line-height:1.28;font-weight:600;color:${PALETTE.text}}
</style><main><div class="brand">${plate(92)}<b>doc-html-translate</b></div><p>${line}</p></main>`;
}

export function consoleHtml() {
  return `<!doctype html><meta charset="utf-8"><title>demo console</title><style>${BASE_CSS}
  .win{position:absolute;left:90px;top:70px;width:1100px;height:450px;border-radius:14px;background:#0c120d;
    border:1px solid rgba(255,255,255,.14);box-shadow:0 18px 50px rgba(0,0,0,.55);overflow:hidden}
  .bar{height:46px;display:flex;align-items:center;padding:0 22px;font-size:17px;color:${PALETTE.muted};
    border-bottom:1px solid rgba(255,255,255,.08);background:rgba(255,255,255,.03)}
  pre{margin:0;padding:22px 28px;font:26px/1.55 "Cascadia Mono",Consolas,monospace;color:${PALETTE.text2};white-space:pre-wrap}
  .p{color:${PALETTE.acc}}.c{color:${PALETTE.text}}.ok{color:${PALETTE.acc};font-weight:700}
  .caret{display:inline-block;width:13px;height:28px;background:${PALETTE.text};vertical-align:-5px;margin-left:2px}
</style><div class="win"><div class="bar">doc-html-translate</div><pre id="t"></pre></div>
<script>
  // setTerm draws the command typed so far and the console lines printed so far.
  window.setTerm = function (typed, lines, caret) {
    var esc = function (s) { return s.replace(/&/g, "&amp;").replace(/</g, "&lt;"); };
    var out = '<span class="p">&gt;</span> <span class="c">' + esc(typed) + '</span>' + (caret ? '<span class="caret"></span>' : '');
    lines.forEach(function (l) { out += "\\n" + (l === "Done." ? '<span class="ok">' + l + '</span>' : esc(l)); });
    document.getElementById("t").innerHTML = out;
  };
</script>`;
}

// ---- overlays injected into the pages being captured -------------------------------------

// The product's version stamp is hidden so the capture does not date itself; nothing else of the
// converted page is altered.
export const HIDE_VERSION_CSS = ".nav-version{display:none !important}";

// captionScript returns an expression that shows (or replaces) the scene caption. `accent` is the
// part of the text set in the gold of the brand palette.
export function captionScript(text, accent = "") {
  const esc = (s) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;");
  const at = accent ? text.indexOf(accent) : -1;
  const html = at < 0 ? esc(text)
    : `${esc(text.slice(0, at))}<span style="color:${PALETTE.gold}">${esc(accent)}</span>${esc(text.slice(at + accent.length))}`;
  return `(() => {
    let s = document.getElementById("demo-hide"); if (!s) { s = document.createElement("style"); s.id = "demo-hide"; s.textContent = ${JSON.stringify(HIDE_VERSION_CSS)}; document.head.appendChild(s); }
    let c = document.getElementById("demo-cap");
    if (!c) { c = document.createElement("div"); c.id = "demo-cap"; document.documentElement.appendChild(c); }
    c.style.cssText = "position:fixed;left:50%;bottom:22px;transform:translateX(-50%);z-index:2147483647;max-width:1120px;" +
      "padding:10px 24px;border-radius:14px;background:rgba(10,15,10,.9);border:1px solid rgba(255,255,255,.2);" +
      "color:${PALETTE.text};font:600 28px/1.25 'Segoe UI',system-ui,sans-serif;text-align:center;white-space:nowrap;box-shadow:0 8px 28px rgba(0,0,0,.45)";
    c.innerHTML = ${JSON.stringify(html)};
  })()`;
}

export const clearCaptionScript = `(() => { const c = document.getElementById("demo-cap"); if (c) c.remove(); })()`;
