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
