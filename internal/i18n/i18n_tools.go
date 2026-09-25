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
}
