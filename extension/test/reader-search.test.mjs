import test from 'node:test';
import assert from 'node:assert/strict';
import { occurrences } from '../src/reader-search.js';

test('search matches non-Latin and RTL text without changing its content', () => {
  assert.deepEqual(occurrences('مرحبًا بالعالم مرحبًا', 'مرحبًا', 'ar'), [0, 15]);
  assert.deepEqual(occurrences('東京と東京', '東京', 'ja'), [0, 3]);
});

test('search uses the document locale for case', () => {
  assert.deepEqual(occurrences('I ı İ i', 'ı', 'tr'), [0, 2]);
  assert.deepEqual(occurrences('I ı İ i', 'i', 'tr'), [4, 6]);
});

test('match offsets stay in the original string when lowering changes length', () => {
  // İ lowers to two code units under most locales, so the lowered haystack is longer
  // than its source; every reported offset must still point into the ORIGINAL string,
  // or the highlight would land one character short of the match.
  assert.deepEqual(occurrences('İstanbul izmir', 'izmir', 'en'), [9]);
});
