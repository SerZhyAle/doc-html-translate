package i18n

// Chrome of the converted page: the navigation bar and the reader controls injected into every
// generated HTML file. Short strings by design - this is the frame around a book, not an app.
//
// These words appear inside a page whose <html lang> is the *document's* language, so they also
// carry their own lang attribute; see internal/htmlgen.
//
// Order of the translations is always Codes[1:]: ru uk de it es fr pt ar hi bn ur zh.
func init() {
	Add("Back",
		"Назад", "Назад", "Zurück", "Indietro", "Atrás", "Retour", "Voltar",
		"رجوع", "पिछला", "পূর্ববর্তী", "پیچھے", "上一页")

	Add("Forward",
		"Вперёд", "Вперед", "Weiter", "Avanti", "Adelante", "Suivant", "Avançar",
		"التالي", "अगला", "পরবর্তী", "آگے", "下一页")

	Add("Contents",
		"Оглавление", "Зміст", "Inhalt", "Indice", "Índice", "Sommaire", "Sumário",
		"المحتويات", "विषय-सूची", "সূচিপত্র", "فہرست", "目录")

	Add("Smaller text",
		"Мельче", "Дрібніше", "Kleinerer Text", "Testo più piccolo", "Texto más pequeño",
		"Texte plus petit", "Texto menor",
		"نص أصغر", "छोटा पाठ", "ছোট লেখা", "چھوٹا متن", "缩小文字")

	Add("Larger text",
		"Крупнее", "Більше", "Größerer Text", "Testo più grande", "Texto más grande",
		"Texte plus grand", "Texto maior",
		"نص أكبر", "बड़ा पाठ", "বড় লেখা", "بڑا متن", "放大文字")

	Add("Show or hide the recognized text layer",
		"Показать или скрыть слой распознанного текста",
		"Показати або сховати шар розпізнаного тексту",
		"Erkannte Textebene ein- oder ausblenden",
		"Mostra o nascondi il livello di testo riconosciuto",
		"Mostrar u ocultar la capa de texto reconocido",
		"Afficher ou masquer le calque de texte reconnu",
		"Mostrar ou ocultar a camada de texto reconhecido",
		"إظهار أو إخفاء طبقة النص المتعرف عليه",
		"पहचानी गई पाठ परत दिखाएँ या छिपाएँ",
		"শনাক্ত করা লেখার স্তর দেখান বা লুকান",
		"شناخت شدہ متن کی تہہ دکھائیں یا چھپائیں",
		"显示或隐藏识别的文字层")

	Add("Go to page",
		"Перейти к странице", "Перейти до сторінки", "Zu Seite springen",
		"Vai alla pagina", "Ir a la página", "Aller à la page", "Ir para a página",
		"الانتقال إلى الصفحة", "पृष्ठ पर जाएँ", "পৃষ্ঠায় যান", "صفحہ پر جائیں", "跳转到页面")

	Add("Font",
		"Шрифт", "Шрифт", "Schrift", "Carattere", "Fuente", "Police", "Fonte",
		"الخط", "फ़ॉन्ट", "ফন্ট", "فونٹ", "字体")

	Add("Theme",
		"Тема", "Тема", "Design", "Tema", "Tema", "Thème", "Tema",
		"المظهر", "थीम", "থিম", "تھیم", "主题")

	Add("Light",
		"Светлая", "Світла", "Hell", "Chiaro", "Claro", "Clair", "Claro",
		"فاتح", "हल्का", "উজ্জ্বল", "روشن", "浅色")

	Add("Sepia",
		"Сепия", "Сепія", "Sepia", "Seppia", "Sepia", "Sépia", "Sépia",
		"بني داكن", "सेपिया", "সেপিয়া", "سیپیا", "棕褐色")

	Add("Dark",
		"Тёмная", "Темна", "Dunkel", "Scuro", "Oscuro", "Sombre", "Escuro",
		"داكن", "गहरा", "গাঢ়", "گہرا", "深色")

	Add("Night",
		"Ночь", "Ніч", "Nacht", "Notte", "Noche", "Nuit", "Noite",
		"ليلي", "रात", "রাত", "رات", "夜间")

	Add("Serif",
		"С засечками", "З засічками", "Serif", "Con grazie", "Con serifa", "Serif", "Com serifa",
		"مذيل", "सेरिफ़", "সেরিফ", "سیرف", "衬线")

	Add("Sans",
		"Без засечек", "Без засічок", "Serifenlos", "Senza grazie", "Sin serifa", "Sans serif",
		"Sem serifa",
		"غير مذيل", "सैंस-सेरिफ़", "সান্স-সেরিফ", "بغیر سیرف", "无衬线")

	Add("Mono",
		"Моноширинный", "Моноширинний", "Monospace", "Monospaziato", "Monoespaciado",
		"Monospace", "Monoespaçada",
		"ثابت العرض", "मोनोस्पेस", "মনোস্পেস", "یکساں چوڑائی", "等宽")

	Add("Continue reading",
		"Продолжить чтение", "Продовжити читання", "Weiterlesen", "Continua a leggere",
		"Seguir leyendo", "Continuer la lecture", "Continuar a leitura",
		"متابعة القراءة", "पढ़ना जारी रखें", "পড়া চালিয়ে যান", "پڑھنا جاری رکھیں", "继续阅读")

	Add("Chapters: %d",
		"Глав: %d", "Розділів: %d", "Kapitel: %d", "Capitoli: %d", "Capítulos: %d",
		"Chapitres : %d", "Capítulos: %d",
		"الفصول: %d", "अध्याय: %d", "অধ্যায়: %d", "ابواب: %d", "章节：%d")
}
