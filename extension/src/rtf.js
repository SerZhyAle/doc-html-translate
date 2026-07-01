// rtf.js - RTF reader. Ports internal/rtf/extract.go: a lightweight stripper that
// removes control words and braces, decodes \'XX hex escapes as Windows-1251
// (common for Russian RTF), and splits the plain text into paragraphs.

import { paragraphsToBook } from "./txt.js";

// stripRtf turns RTF bytes into plain text. Pure (uses TextDecoder, no DOM) -
// unit-tested. Bytes are read as latin1 so each byte maps to one char (matching
// the Go string(data)); \'XX hex bytes are decoded as Windows-1251.
export function stripRtf(bytes) {
  const input = new TextDecoder("latin1").decode(bytes);
  const cp1251 = new TextDecoder("windows-1251");
  let out = "";
  let i = 0;
  let depth = 0;

  while (i < input.length) {
    const ch = input[i];
    if (ch === "{") {
      depth++; i++;
    } else if (ch === "}") {
      depth--; if (depth < 0) depth = 0; i++;
    } else if (ch === "\\") {
      i++;
      if (i >= input.length) break;
      const next = input[i];
      if (next === "\\") { out += "\\"; i++; }
      else if (next === "{") { out += "{"; i++; }
      else if (next === "}") { out += "}"; i++; }
      else if (next === "'") {
        if (i + 2 < input.length) {
          const b = parseInt(input.slice(i + 1, i + 3), 16);
          if (!Number.isNaN(b)) {
            out += b < 0x80 ? String.fromCharCode(b) : cp1251.decode(new Uint8Array([b]));
          }
          i += 3;
        } else {
          i++;
        }
      } else if (next === "\n" || next === "\r") {
        out += "\n\n"; i++;
      } else {
        const [word, ni] = readControlWord(input, i);
        i = ni;
        if (word === "par" || word === "line") out += "\n\n";
        else if (word === "tab") out += "\t";
        else if (word === "u") {
          const [num, ni2] = readNumber(input, i);
          i = ni2;
          if (num >= 0 && num <= 0x10ffff) out += String.fromCodePoint(num);
          if (i < input.length && input[i] !== "\\" && input[i] !== "{" && input[i] !== "}") i++;
        }
      }
    } else {
      out += ch; i++;
    }
  }
  return out;
}

// readControlWord reads a control word (letters), skips an optional numeric
// parameter and one trailing space; returns [word, newIndex].
function readControlWord(input, i) {
  const start = i;
  while (i < input.length && /[a-zA-Z]/.test(input[i])) i++;
  const word = input.slice(start, i);
  while (i < input.length && (/[0-9]/.test(input[i]) || input[i] === "-")) i++;
  if (i < input.length && input[i] === " ") i++;
  return [word, i];
}

// readNumber reads a (possibly negative) decimal and one optional space delimiter.
function readNumber(input, i) {
  let neg = false;
  if (i < input.length && input[i] === "-") { neg = true; i++; }
  let num = 0;
  let found = false;
  while (i < input.length && input[i] >= "0" && input[i] <= "9") {
    num = num * 10 + (input.charCodeAt(i) - 48);
    found = true; i++;
  }
  if (!found) return [0, i];
  if (neg) num = -num;
  if (i < input.length && input[i] === " ") i++;
  return [num, i];
}

// splitRtfParagraphs groups consecutive non-blank lines into paragraphs, matching
// the Go rtf splitParagraphs (blank-line separated, joined with a space).
function splitRtfParagraphs(text) {
  const norm = String(text).replace(/\r\n?/g, "\n");
  const paras = [];
  let cur = "";
  for (const line of norm.split("\n")) {
    const t = line.trim();
    if (t === "") {
      if (cur) { paras.push(cur); cur = ""; }
    } else {
      cur = cur ? `${cur} ${t}` : t;
    }
  }
  if (cur) paras.push(cur);
  return paras;
}

// parseRtf strips the RTF and returns the render-ready book shape.
export async function parseRtf(data) {
  const text = stripRtf(data);
  return paragraphsToBook(splitRtfParagraphs(text), "rtf-sec");
}
