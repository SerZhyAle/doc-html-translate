import vm from "node:vm";
import { test, before } from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { createCanvas, ImageData, loadImage } from "@napi-rs/canvas";
import { parseHTML } from "linkedom";
import { isJxl, jxlSize, decodeJxl, imageBlob, normalizeJxlImages } from "../src/jxl.js";

const raw = new Uint8Array(await readFile(new URL("../../tests/fixtures/jxl/rgba.jxl", import.meta.url)));
const signature = [0,0,0,12,74,88,76,32,13,10,135,10];
function box(type, bytes) {
  const out = new Uint8Array(bytes.length + 8);
  new DataView(out.buffer).setUint32(0,out.length);
  out.set(new TextEncoder().encode(type),4); out.set(bytes,8);
  return out;
}
function concat(...parts) { return new Uint8Array(Buffer.concat(parts.map(p => Buffer.from(p)))); }
const container = concat(signature,box("ftyp",[106,120,108,32,0,0,0,0,106,120,108,32]),box("jxlc",raw));

before(async () => {
  globalThis.ImageData = ImageData;
  globalThis.OffscreenCanvas = class {
    constructor(w,h) { this.canvas = createCanvas(w,h); }
    getContext(kind) { return this.canvas.getContext(kind); }
    async convertToBlob() { return new Blob([await this.canvas.encode("png")],{type:"image/png"}); }
  };
  const { init } = await import("../vendor/jxl/decode.js");
  await init({ wasmBinary: await readFile(new URL("../vendor/jxl/codec/dec/jxl_dec.wasm",import.meta.url)) });
});

test("JPEG XL codestream and container decode losslessly including alpha", async () => {
  for (const bytes of [raw,container]) {
    assert.ok(isJxl(bytes));
    assert.deepEqual(jxlSize(bytes), {width:3,height:2});
    const im = await decodeJxl(bytes);
    assert.equal(im.width,3); assert.equal(im.height,2);
    for (let y=0;y<2;y++) for (let x=0;x<3;x++) {
      assert.deepEqual([...im.data.slice((y*3+x)*4,(y*3+x+1)*4)], [200,x*60,y*80,55+x*100]);
    }
    const png = await imageBlob(bytes,"image/jxl");
    assert.equal(png.type,"image/png");
    const image = await loadImage(Buffer.from(await png.arrayBuffer()));
    assert.equal(image.width,3); assert.equal(image.height,2);
  }
});

test("partial codestream boxes and extended box sizes probe correctly", () => {
  const parts = concat(signature,box("jxlp",concat([0,0,0,0],raw.slice(0,3))),box("jxlp",concat([128,0,0,1],raw.slice(3))));
  assert.deepEqual(jxlSize(parts),{width:3,height:2});
  const extended = new Uint8Array(raw.length+16);
  new DataView(extended.buffer).setUint32(0,1);
  extended.set(new TextEncoder().encode("jxlc"),4);
  new DataView(extended.buffer).setUint32(12,extended.length);
  extended.set(raw,16);
  assert.deepEqual(jxlSize(concat(signature,extended)),{width:3,height:2});
});

test("malformed input and pixel bombs are refused before raster decode", async () => {
  for (const bad of [new Uint8Array(), new Uint8Array([255,10]), concat(signature,[0,0,0,1,106,120,108,99])]) {
    assert.throws(()=>jxlSize(bad));
    await assert.rejects(decodeJxl(bad));
  }
  const bits=[];
  const put=(v,n)=>{for(let i=0;i<n;i++) bits.push((v>>>i)&1);};
  put(0,1); put(2,2); put(59999,18); put(1,3);
  const bomb=new Uint8Array(2+Math.ceil(bits.length/8)); bomb.set([255,10]);
  bits.forEach((v,i)=>bomb[2+(i>>3)]|=v<<(i&7));
  assert.throws(()=>jxlSize(bomb),/60000 x 60000/);
});

test("embedded FB2/data pictures become PNG and their URLs are released", async () => {
  const {document}=parseHTML("<html><body></body></html>");
  const frag=document.createDocumentFragment();
  const img=document.createElement("img");
  img.setAttribute("src",`data:image/jxl;base64,${Buffer.from(raw).toString("base64")}`);
  frag.append(img);
  let revoked=false;
  const book={sections:[{frag}],revoke:()=>{revoked=true;}};
  await normalizeJxlImages(book);
  const url=img.getAttribute("src");
  assert.ok(url.startsWith("blob:"));
  assert.equal((await (await fetch(url)).blob()).type,"image/png");
  book.revoke(); assert.ok(revoked);
  await assert.rejects(fetch(url));
});

test("viewer routes JPEG XL by signature even when the URL has no image suffix", async () => {
  const source = await readFile(new URL("../src/viewer.js",import.meta.url),"utf8");
  const routing = source.slice(source.indexOf("const FORMAT_EXT ="),source.indexOf("// Release the previous document"));
  const context = vm.createContext({ Uint8Array, isJxl, isMobiBytes:()=>false });
  vm.runInContext(routing,context);
  for (const bytes of [raw,container]) {
    const data=bytes.slice().buffer;
    assert.equal(context.detectFormat(data,"https://example.org/download?id=12"),"image");
    assert.equal(context.imageMime(data,"document.pdf"),"image/jxl");
  }
  assert.equal(context.detectFormat(new ArrayBuffer(0),"picture.JXL"),"image");
});
