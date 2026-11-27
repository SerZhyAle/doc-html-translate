# Releases

<!-- Rendered by the sza release-ledger from channels.json, builds.json and history.jsonl (RELEASE-LEDGER rule 10). Do not edit: the next write replaces this file. -->

What each package was built as last, what each channel carries, and every publication before it.

## Channels

| Package | Channel | Kind | Live | Since | In flight | Ref |
| --- | --- | --- | --- | --- | --- | --- |
| app | github | github | 26.1126.2358 | 2026-11-27 | - | [v26.1126.2358](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.1126.2358) |
| app | winget | winget | 26.1117.1450 | 2026-11-18 | 26.1126.2358 submitted (2026-11-27) | [PR#490566](https://github.com/microsoft/winget-pkgs/pull/490566) |
| app | msstore | msstore | 26.1117.1450.0 | 2026-11-18 | 26.1126.2358 pending (2026-11-27) | v26.1126.2358 |
| app | installer (Universal Windows installer (GitHub Release asset)) | manual | 26.1126.2358 | 2026-11-27 | - | [v26.1126.2358](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.1126.2358) |
| extension | chrome | chrome | 26.1117.1543 | 2026-11-18 | 26.1127.133 submitted (2026-11-27); 26.1126.2356 blocked (2026-11-27) | ext-cws-v26.1127.133 |
| extension | edge | edge | 26.1113.22 | 2026-11-18 | 26.1127.133 blocked (2026-11-27); 26.1126.2356 submitted (2026-11-27); 26.1117.1543 blocked (2026-11-18) | ext-edge-v26.1127.133 |

## Last builds

| Package | Flavor | Kind | Version | Built | Artifact |
| --- | --- | --- | --- | --- | --- |
| app | cli | debug | 26.1127.0150 | 2026-11-27 01:50:38+01:00 | build/doc-html-translate.exe |
| app | installer | debug | 26.1126.2321 | 2026-11-26 23:21:53+01:00 | dist/doc-html-translate-setup-26.1126.2321.exe |
| app | installer | release | 26.1126.2358 | 2026-11-27 00:34:53+01:00 | temp/release-26.1126.2358/dist/doc-html-translate-setup-26.1126.2358.exe |
| app | msix | release | 26.1126.2358 | 2026-11-27 00:34:43+01:00 | msix/out/SZA.Doc-HTML-Translate_26.1126.2358.0_x64.msix |
| app | msix-selftest | test | 26.1126.2333 | 2026-11-26 23:33:08+01:00 | msix/out/SerZhyAle.DocHtmlTranslate_26.1126.2333.0_x64_selftest.msix |
| app | ui | debug | 26.1127.0150 | 2026-11-27 01:50:41+01:00 | build/doc-html-ui.exe |
| extension | unpacked | debug | 26.1126.2333 | 2026-11-26 23:33:58+01:00 | - |

## History

| When | Package | Channel | Version | State | Source | Ref | Note |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 2026-11-27 02:00:27+01:00 | extension | edge | 26.1127.133 | blocked | release | ext-edge-v26.1127.133 | package uploaded; publish refused InProgressSubmission while 26.1126.2356 is in certification; rerun run 45919944729 --failed after it clears |
| 2026-11-27 02:00:27+01:00 | extension | chrome | 26.1127.133 | submitted | release | ext-cws-v26.1127.133 | no downloads permission; PENDING_REVIEW |
| 2026-11-27 00:47:25+01:00 | app | msstore | 26.1126.2358 | pending | release | v26.1126.2358 | package and listing import folder built from the tag in msix/out; machine channel not usable (metadata JSON above the command-line limit) and the Partner Center product page did not load; manual upload owed |
| 2026-11-27 00:47:24+01:00 | extension | chrome | 26.1126.2356 | blocked | release | ext-cws-v26.1126.2356 | package uploaded; publish refused INVALID_ITEM_METADATA: the downloads permission justification is owed in the developer dashboard, which asked for a Google sign-in |
| 2026-11-27 00:38:03+01:00 | extension | edge | 26.1126.2356 | submitted | release | ext-edge-v26.1126.2356 | - |
| 2026-11-27 00:38:02+01:00 | app | winget | 26.1126.2358 | submitted | release | [PR#490566](https://github.com/microsoft/winget-pkgs/pull/490566) | - |
| 2026-11-27 00:34:54+01:00 | app | installer | 26.1126.2358 | live | manual | [v26.1126.2358](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.1126.2358) | - |
| 2026-11-27 00:34:54+01:00 | app | github | 26.1126.2358 | live | release | [v26.1126.2358](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.1126.2358) | - |
| 2026-11-18 23:02:06+01:00 | extension | edge | 26.1117.1543 | blocked | manual | [ext-edge-v26.1117.1543](https://microsoftedge.microsoft.com/addons/detail/anokfnnfiboaccbbpfkdaphejnkkhajh) | Migrated from DEV/RELEASE_STATE.md (2026-11-17 15:45): publish-edge run 44106343944 FAILED at publish (InProgressSubmission while 26.1113.22 cert was in progress) |
| 2026-11-18 23:01:32+01:00 | extension | edge | 26.1113.22 | live | import | [link](https://microsoftedge.microsoft.com/addons/detail/anokfnnfiboaccbbpfkdaphejnkkhajh) | - |
| 2026-11-18 23:01:32+01:00 | extension | chrome | 26.1117.1543 | live | import | [link](https://chromewebstore.google.com/detail/nmcckamdocainafmmompkbmelkpbnmic) | - |
| 2026-11-18 23:01:32+01:00 | app | msstore | 26.1117.1450.0 | live | import | [Submission 14](https://apps.microsoft.com/detail/9PMHSWQPR6V1) | Store status Published |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.1117.1450 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.1117.1450) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.1112.2325 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.1112.2325) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.1021.0300 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.1021.0300) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.1010.0142 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.1010.0142) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.1007.0054 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.1007.0054) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0930.1107 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0930.1107) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0912.2026 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0912.2026) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0815.2020 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0815.2020) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0813.1917 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0813.1917) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0729.0134 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0729.0134) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0718.0252 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0718.0252) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0715.2158 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0715.2158) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0714.1658 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0714.1658) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0709.2245 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0709.2245) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0702.1530 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0702.1530) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0612.0124 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0612.0124) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0511.0342 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0511.0342) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0428.0237 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0428.0237) | - |
| 2026-11-18 23:01:32+01:00 | app | winget | 26.0427.1713 | live | import | [link](https://github.com/microsoft/winget-pkgs/tree/master/manifests/s/SerZhyAle/DocHtmlTranslate/26.0427.1713) | - |
| 2026-11-17 14:35:55Z | app | installer | 26.1117.1450 | live | manual | [v26.1117.1450](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.1117.1450) | doc-html-translate-setup-26.1117.1450.exe uploaded to GitHub Release |
| 2026-11-17 14:35:55Z | app | github | 26.1117.1450 | live | import | [v26.1117.1450](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.1117.1450) | - |
| 2026-11-12 23:16:13Z | app | github | 26.1112.2325 | live | import | [v26.1112.2325](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.1112.2325) | - |
| 2026-10-21 00:36:59Z | app | github | 26.1021.0300 | live | import | [v26.1021.0300](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.1021.0300) | - |
| 2026-10-09 23:57:19Z | app | github | 26.1010.0142 | live | import | [v26.1010.0142](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.1010.0142) | - |
| 2026-10-06 23:41:46Z | app | github | 26.1007.0054 | live | import | [v26.1007.0054](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.1007.0054) | - |
| 2026-09-30 09:54:56Z | app | github | 26.0930.1107 | live | import | [v26.0930.1107](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0930.1107) | - |
| 2026-09-12 18:27:57Z | app | github | 26.0912.2026 | live | import | [v26.0912.2026](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0912.2026) | - |
| 2026-08-15 18:35:22Z | app | github | 26.0815.2020 | live | import | [v26.0815.2020](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0815.2020) | - |
| 2026-08-13 17:18:46Z | app | github | 26.0813.1917 | live | import | [v26.0813.1917](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0813.1917) | - |
| 2026-07-29 00:03:36Z | app | github | 26.0729.0134 | live | import | [v26.0729.0134](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0729.0134) | - |
| 2026-07-18 00:54:05Z | app | github | 26.0718.0252 | live | import | [v26.0718.0252](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0718.0252) | - |
| 2026-07-15 20:34:32Z | app | github | 26.0715.2158 | live | import | [v26.0715.2158](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0715.2158) | - |
| 2026-07-14 15:06:35Z | app | github | 26.0714.1658 | live | import | [v26.0714.1658](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0714.1658) | - |
| 2026-07-09 20:53:24Z | app | github | 26.0709.2245 | live | import | [v26.0709.2245](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0709.2245) | - |
| 2026-07-02 13:34:17Z | app | github | 26.0702.1530 | live | import | [v26.0702.1530](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0702.1530) | - |
| 2026-06-11 23:27:20Z | app | github | 26.0612.0124 | live | import | [v26.0612.0124](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0612.0124) | - |
| 2026-05-15 18:55:09Z | app | github | 26.0515.2046 | live | import | [v26.0515.2046](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0515.2046) | - |
| 2026-05-11 01:46:49Z | app | github | 26.0511.0342 | live | import | [v26.0511.0342](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0511.0342) | - |
| 2026-04-28 00:39:46Z | app | github | 26.0428.0237 | live | import | [v26.0428.0237](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0428.0237) | - |
| 2026-04-27 15:15:24Z | app | github | 26.0427.1713 | live | import | [v26.0427.1713](https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0427.1713) | - |
