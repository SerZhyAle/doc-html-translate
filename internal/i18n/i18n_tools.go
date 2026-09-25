package i18n

// Console strings about the external helper programs the converter runs (pdftotext,
// Tesseract, Calibre, 7-Zip, ffmpeg/ImageMagick): where to get one that is missing.
//
// Order of the translations is always Codes[1:]: ru uk de it es fr pt ar hi bn ur zh.
func init() {
	Add("pdftotext not found - using the built-in PDF reader. For better text, install Poppler (package poppler-utils, or brew install poppler).",
		"pdftotext не найден - используется встроенный PDF-ридер. Для лучшего текста установите Poppler (пакет poppler-utils или brew install poppler).",
		"pdftotext не знайдено - використовується вбудований PDF-рідер. Для кращого тексту встановіть Poppler (пакет poppler-utils або brew install poppler).",
		"pdftotext nicht gefunden - der eingebaute PDF-Leser wird verwendet. Für besseren Text Poppler installieren (Paket poppler-utils oder brew install poppler).",
		"pdftotext non trovato - viene usato il lettore PDF integrato. Per un testo migliore installa Poppler (pacchetto poppler-utils oppure brew install poppler).",
		"No se encontró pdftotext - se usa el lector PDF integrado. Para un texto mejor, instala Poppler (paquete poppler-utils o brew install poppler).",
		"pdftotext introuvable - le lecteur PDF intégré est utilisé. Pour un meilleur texte, installez Poppler (paquet poppler-utils ou brew install poppler).",
		"pdftotext não encontrado - usando o leitor de PDF integrado. Para um texto melhor, instale o Poppler (pacote poppler-utils ou brew install poppler).",
		"لم يُعثر على pdftotext - يُستخدم قارئ PDF المدمج. للحصول على نص أفضل ثبّت Poppler (الحزمة poppler-utils أو brew install poppler).",
		"pdftotext नहीं मिला - अंतर्निहित PDF रीडर का उपयोग हो रहा है। बेहतर टेक्स्ट के लिए Poppler इंस्टॉल करें (पैकेज poppler-utils या brew install poppler)।",
		"pdftotext পাওয়া যায়নি - অন্তর্নির্মিত PDF রিডার ব্যবহার হচ্ছে। ভালো টেক্সটের জন্য Poppler ইনস্টল করুন (প্যাকেজ poppler-utils অথবা brew install poppler)।",
		"pdftotext نہیں ملا - اندرونی PDF ریڈر استعمال ہو رہا ہے۔ بہتر متن کے لیے Poppler انسٹال کریں (پیکیج poppler-utils یا brew install poppler)۔",
		"未找到 pdftotext - 正在使用内置 PDF 阅读器。如需更好的文本，请安装 Poppler（软件包 poppler-utils，或 brew install poppler）。")

	Add("pdftotext could not run (possibly blocked by antivirus), so the text was read with a less accurate method - ligatures and complex fonts may look wrong. Nothing is installed automatically. To restore full quality, install Poppler yourself, for example: winget install ossia.poppler",
		"pdftotext не удалось запустить (возможно, его заблокировал антивирус), поэтому текст прочитан менее точным способом - лигатуры и сложные шрифты могут отображаться неверно. Ничего не устанавливается автоматически. Чтобы вернуть полное качество, установите Poppler сами, например: winget install ossia.poppler",
		"pdftotext не вдалося запустити (можливо, його заблокував антивірус), тому текст прочитано менш точним способом - лігатури та складні шрифти можуть відображатися неправильно. Нічого не встановлюється автоматично. Щоб повернути повну якість, встановіть Poppler самостійно, наприклад: winget install ossia.poppler",
		"pdftotext konnte nicht gestartet werden (möglicherweise vom Virenschutz blockiert), daher wurde der Text mit einer ungenaueren Methode gelesen - Ligaturen und komplexe Schriften können falsch aussehen. Es wird nichts automatisch installiert. Für volle Qualität installieren Sie Poppler selbst, zum Beispiel: winget install ossia.poppler",
		"Impossibile avviare pdftotext (forse bloccato dall'antivirus), quindi il testo è stato letto con un metodo meno preciso - legature e font complessi potrebbero apparire errati. Nulla viene installato automaticamente. Per ripristinare la qualità piena, installa Poppler tu stesso, ad esempio: winget install ossia.poppler",
		"No se pudo ejecutar pdftotext (quizá lo bloqueó el antivirus), así que el texto se leyó con un método menos preciso - las ligaduras y las fuentes complejas pueden verse mal. No se instala nada automáticamente. Para recuperar la calidad completa, instala Poppler tú mismo, por ejemplo: winget install ossia.poppler",
		"pdftotext n'a pas pu s'exécuter (peut-être bloqué par l'antivirus), le texte a donc été lu avec une méthode moins précise - les ligatures et les polices complexes peuvent mal s'afficher. Rien n'est installé automatiquement. Pour retrouver la qualité complète, installez Poppler vous-même, par exemple : winget install ossia.poppler",
		"Não foi possível executar o pdftotext (talvez bloqueado pelo antivírus), então o texto foi lido com um método menos preciso - ligaduras e fontes complexas podem aparecer erradas. Nada é instalado automaticamente. Para recuperar a qualidade total, instale o Poppler você mesmo, por exemplo: winget install ossia.poppler",
		"تعذّر تشغيل pdftotext (ربما حظره برنامج مكافحة الفيروسات)، لذا قُرئ النص بطريقة أقل دقة - قد تظهر الحروف المركبة والخطوط المعقدة بشكل خاطئ. لا يُثبَّت أي شيء تلقائيًا. لاستعادة الجودة الكاملة ثبّت Poppler بنفسك، مثلًا: winget install ossia.poppler",
		"pdftotext नहीं चल सका (शायद एंटीवायरस ने रोका), इसलिए टेक्स्ट कम सटीक तरीके से पढ़ा गया - लिगेचर और जटिल फ़ॉन्ट गलत दिख सकते हैं। कुछ भी अपने आप इंस्टॉल नहीं होता। पूरी गुणवत्ता के लिए Poppler खुद इंस्टॉल करें, उदाहरण: winget install ossia.poppler",
		"pdftotext চালানো যায়নি (সম্ভবত অ্যান্টিভাইরাস আটকেছে), তাই টেক্সট কম নির্ভুল পদ্ধতিতে পড়া হয়েছে - লিগেচার ও জটিল ফন্ট ভুল দেখাতে পারে। কিছুই স্বয়ংক্রিয়ভাবে ইনস্টল হয় না। পূর্ণ মান ফেরাতে নিজে Poppler ইনস্টল করুন, যেমন: winget install ossia.poppler",
		"pdftotext نہیں چل سکا (شاید اینٹی وائرس نے روکا)، اس لیے متن کم درست طریقے سے پڑھا گیا - لگیچر اور پیچیدہ فونٹ غلط دکھ سکتے ہیں۔ کچھ بھی خودکار طور پر انسٹال نہیں ہوتا۔ پوری کوالٹی کے لیے Poppler خود انسٹال کریں، مثلاً: winget install ossia.poppler",
		"无法运行 pdftotext（可能被杀毒软件拦截），因此文本改用精度较低的方法读取 - 连字和复杂字体可能显示不正确。不会自动安装任何内容。如需恢复完整质量，请自行安装 Poppler，例如：winget install ossia.poppler")

	Add("PDF quality reduced - pdftotext unavailable",
		"Качество PDF снижено - pdftotext недоступен",
		"Якість PDF знижено - pdftotext недоступний",
		"PDF-Qualität eingeschränkt - pdftotext nicht verfügbar",
		"Qualità PDF ridotta - pdftotext non disponibile",
		"Calidad del PDF reducida - pdftotext no disponible",
		"Qualité PDF réduite - pdftotext indisponible",
		"Qualidade do PDF reduzida - pdftotext indisponível",
		"انخفضت جودة PDF - pdftotext غير متاح",
		"PDF गुणवत्ता कम - pdftotext उपलब्ध नहीं",
		"PDF-এর মান কমেছে - pdftotext উপলব্ধ নয়",
		"PDF کا معیار کم - pdftotext دستیاب نہیں",
		"PDF 质量降低 - pdftotext 不可用")

	Add("internal error while converting (details in the run log): %v",
		"внутренняя ошибка при конвертации (подробности в журнале запуска): %v",
		"внутрішня помилка під час конвертації (подробиці в журналі запуску): %v",
		"interner Fehler bei der Konvertierung (Details im Laufprotokoll): %v",
		"errore interno durante la conversione (dettagli nel registro di esecuzione): %v",
		"error interno durante la conversión (detalles en el registro de ejecución): %v",
		"erreur interne pendant la conversion (détails dans le journal d'exécution) : %v",
		"erro interno durante a conversão (detalhes no registro de execução): %v",
		"خطأ داخلي أثناء التحويل (التفاصيل في سجل التشغيل): %v",
		"रूपांतरण के दौरान आंतरिक त्रुटि (विवरण रन लॉग में): %v",
		"রূপান্তরের সময় অভ্যন্তরীণ ত্রুটি (বিস্তারিত রান লগে): %v",
		"تبدیلی کے دوران اندرونی خرابی (تفصیل رن لاگ میں): %v",
		"转换时发生内部错误（详情见运行日志）：%v")

	Add("%s did not finish within %s and was stopped (set DOCHT_TOOL_TIMEOUT_SCALE to allow more time)",
		"%s не завершился за %s и был остановлен (чтобы дать больше времени, задайте DOCHT_TOOL_TIMEOUT_SCALE)",
		"%s не завершився за %s і був зупинений (щоб дати більше часу, задайте DOCHT_TOOL_TIMEOUT_SCALE)",
		"%s wurde nicht innerhalb von %s fertig und wurde beendet (DOCHT_TOOL_TIMEOUT_SCALE setzen, um mehr Zeit zu geben)",
		"%s non ha terminato entro %s ed è stato fermato (imposta DOCHT_TOOL_TIMEOUT_SCALE per concedere più tempo)",
		"%s no terminó en %s y se detuvo (define DOCHT_TOOL_TIMEOUT_SCALE para dar más tiempo)",
		"%s ne s'est pas terminé en %s et a été arrêté (définissez DOCHT_TOOL_TIMEOUT_SCALE pour accorder plus de temps)",
		"%s não terminou em %s e foi interrompido (defina DOCHT_TOOL_TIMEOUT_SCALE para dar mais tempo)",
		"لم ينتهِ %s خلال %s فأُوقف (اضبط DOCHT_TOOL_TIMEOUT_SCALE لمنحه وقتًا أطول)",
		"%s %s में पूरा नहीं हुआ और रोक दिया गया (अधिक समय देने के लिए DOCHT_TOOL_TIMEOUT_SCALE सेट करें)",
		"%s %s-এর মধ্যে শেষ হয়নি, তাই থামানো হয়েছে (বেশি সময় দিতে DOCHT_TOOL_TIMEOUT_SCALE সেট করুন)",
		"%s %s میں مکمل نہیں ہوا اور روک دیا گیا (زیادہ وقت دینے کے لیے DOCHT_TOOL_TIMEOUT_SCALE سیٹ کریں)",
		"%s 未在 %s 内完成，已被停止（设置 DOCHT_TOOL_TIMEOUT_SCALE 可给予更多时间）")
}
