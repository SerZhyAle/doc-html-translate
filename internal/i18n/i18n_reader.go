package i18n

// Chrome of the converted page: the navigation bar and the reader controls injected into every
// generated HTML file. Short strings by design - this is the frame around a book, not an app.
//
// These words appear inside a page whose <html lang> is the *document's* language, so they also
// carry their own lang attribute; see internal/htmlgen.
//
// Order of the translations is always Codes[1:]: ru uk de it es fr pt ar hi bn ur zh.
func init() {
	Add("Search", "Поиск", "Пошук", "Suche", "Cerca", "Buscar", "Rechercher", "Pesquisar", "بحث", "खोजें", "অনুসন্ধান", "تلاش", "搜索")
	Add("Search text", "Искать текст", "Шукати текст", "Text suchen", "Cerca testo", "Buscar texto", "Rechercher du texte", "Pesquisar texto", "البحث في النص", "पाठ खोजें", "পাঠ খুঁজুন", "متن تلاش کریں", "搜索文字")
	Add("Scope", "Область", "Область", "Bereich", "Ambito", "Ámbito", "Portée", "Âmbito", "النطاق", "दायरा", "পরিসর", "دائرہ", "范围")
	Add("This page", "Эта страница", "Ця сторінка", "Diese Seite", "Questa pagina", "Esta página", "Cette page", "Esta página", "هذه الصفحة", "यह पृष्ठ", "এই পৃষ্ঠা", "یہ صفحہ", "本页")
	Add("Whole book", "Вся книга", "Уся книга", "Ganzes Buch", "Libro intero", "Libro completo", "Livre entier", "Livro inteiro", "الكتاب كله", "पूरी किताब", "সম্পূর্ণ বই", "پوری کتاب", "整本书")
	Add("Close", "Закрыть", "Закрити", "Schließen", "Chiudi", "Cerrar", "Fermer", "Fechar", "إغلاق", "बंद करें", "বন্ধ করুন", "بند کریں", "关闭")
	Add("Matches: {1} - {2}", "Найдено: {1} - {2}", "Знайдено: {1} - {2}", "Gefunden: {1} - {2}", "Trovati: {1} - {2}",
		"Encontrados: {1} - {2}", "Trouvés : {1} - {2}", "Encontrados: {1} - {2}", "النتائج: {1} - {2}",
		"परिणाम: {1} - {2}", "ফলাফল: {1} - {2}", "نتائج: {1} - {2}", "匹配：{1} - {2}")
	Add("OCR text plates", "Текстовые блоки OCR", "Текстові блоки OCR", "OCR-Textfelder", "Testo OCR", "Texto OCR", "Texte OCR", "Texto OCR", "نص التعرف الضوئي", "OCR पाठ", "OCR পাঠ", "OCR متن", "OCR 文字")
	Add("Scanned pages need OCR text plates", "Для поиска в сканах нужен распознанный текст", "Для пошуку в сканах потрібен розпізнаний текст", "Gescannte Seiten benötigen OCR-Textfelder", "Le pagine scansionate richiedono testo OCR", "Las páginas escaneadas requieren texto OCR", "Les pages numérisées nécessitent du texte OCR", "Páginas digitalizadas precisam de texto OCR", "تحتاج الصفحات الممسوحة إلى نص التعرف الضوئي", "स्कैन किए गए पृष्ठों के लिए OCR पाठ चाहिए", "স্ক্যান করা পৃষ্ঠায় OCR পাঠ প্রয়োজন", "اسکین شدہ صفحات کے لیے OCR متن درکار ہے", "扫描页面需要 OCR 文字")
	Add("Enter text to search", "Введите текст для поиска", "Введіть текст для пошуку", "Suchtext eingeben", "Inserisci il testo da cercare", "Escribe el texto a buscar", "Saisissez le texte à rechercher", "Digite o texto a pesquisar", "أدخل النص للبحث", "खोजने के लिए पाठ दर्ज करें", "খোঁজার পাঠ লিখুন", "تلاش کے لیے متن لکھیں", "输入搜索文字")
	Add("Searching whole book", "Поиск по всей книге", "Пошук в усій книзі", "Suche im ganzen Buch", "Ricerca nel libro intero", "Buscando en todo el libro", "Recherche dans le livre entier", "Pesquisando no livro inteiro", "البحث في الكتاب كله", "पूरी किताब में खोज", "সম্পূর্ণ বইয়ে খোঁজা হচ্ছে", "پوری کتاب میں تلاش", "正在搜索整本书")
	Add("Search index unavailable", "Поисковый индекс недоступен", "Пошуковий індекс недоступний", "Suchindex nicht verfügbar", "Indice di ricerca non disponibile", "Índice de búsqueda no disponible", "Index de recherche indisponible", "Índice de pesquisa indisponível", "فهرس البحث غير متاح", "खोज अनुक्रमणिका उपलब्ध नहीं है", "অনুসন্ধান সূচি অনুপলব্ধ", "تلاش کا اشاریہ دستیاب نہیں", "搜索索引不可用")
	Add("showing first 500", "показаны первые 500", "показано перші 500", "erste 500 angezeigt", "mostrati i primi 500", "se muestran los primeros 500", "500 premiers affichés", "mostrando os primeiros 500", "عرض أول 500", "पहले 500 दिखाए गए", "প্রথম ৫০০ দেখানো হয়েছে", "پہلے 500 دکھائے گئے", "仅显示前 500 个")
	// Paging names are media.previous / media.next qualified with their object, as ICON-SET
	// rule 3 allows ("Previous page"). They never borrow nav.back's word: the Russian "Назад"
	// and the German "Zurück" these links used to read are Back, a different meaning.
	Add("Previous page",
		"Предыдущая страница", "Попередня сторінка", "Vorherige Seite", "Pagina precedente",
		"Página anterior", "Page précédente", "Página anterior",
		"الصفحة السابقة", "पिछला पृष्ठ", "পূর্ববর্তী পৃষ্ঠা", "پچھلا صفحہ", "上一页")

	Add("Next page",
		"Следующая страница", "Наступна сторінка", "Nächste Seite", "Pagina successiva",
		"Página siguiente", "Page suivante", "Próxima página",
		"الصفحة التالية", "अगला पृष्ठ", "পরবর্তী পৃষ্ঠা", "اگلا صفحہ", "下一页")

	// nav.contents. The same words as the extension's ttToc message in every language
	// (tests/iconography_test.go). The Russian "Оглавление" is the record's declared form for a
	// book's table of contents (ICON-SET 0.15); "Содержание" is the vocabulary's word elsewhere.
	Add("Table of contents",
		"Оглавление", "Зміст", "Inhaltsverzeichnis", "Indice", "Índice", "Table des matières",
		"Sumário",
		"المحتويات", "विषय-सूची", "সূচিপত্র", "فہرست", "目录")

	Add("Smaller text",
		"Мельче", "Дрібніше", "Kleinerer Text", "Testo più piccolo", "Texto más pequeño",
		"Texte plus petit", "Texto menor",
		"نص أصغر", "छोटा पाठ", "ছোট লেখা", "چھوٹا متن", "缩小文字")

	Add("Larger text",
		"Крупнее", "Більше", "Größerer Text", "Testo più grande", "Texto más grande",
		"Texte plus grand", "Texto maior",
		"نص أكبر", "बड़ा पाठ", "বড় লেখা", "بڑا متن", "放大文字")

	// view.text-layer's name (ICON-SET 0.15): the toggle's label, tooltip and accessible name.
	Add("Text layer",
		"Текстовый слой", "Текстовий шар", "Textebene", "Livello di testo", "Capa de texto",
		"Calque de texte", "Camada de texto",
		"طبقة النص", "पाठ परत", "লেখার স্তর", "متن کی تہہ", "文字层")

	Add("Go to page",
		"Перейти к странице", "Перейти до сторінки", "Zu Seite springen",
		"Vai alla pagina", "Ir a la página", "Aller à la page", "Ir para a página",
		"الانتقال إلى صفحة", "पृष्ठ पर जाएँ", "পৃষ্ঠায় যান", "صفحے پر جائیں", "跳转到页面")

	Add("Font",
		"Шрифт", "Шрифт", "Schrift", "Carattere", "Fuente", "Police", "Fonte",
		"الخط", "फ़ॉन्ट", "ফন্ট", "فونٹ", "字体")

	Add("Theme",
		"Тема", "Тема", "Design", "Tema", "Tema", "Thème", "Tema",
		"المظهر", "थीम", "থিম", "تھیم", "主题")

	// The theme choices are words (app.theme's note, ICON-SET): ru and uk name each theme by the
	// record's adjective - Светлая / Сепия / Тёмная / Ночная, Світла / Сепія / Темна / Нічна.
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
		"Ночная", "Нічна", "Nacht", "Notte", "Noche", "Nuit", "Noite",
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

	// Reading-comfort controls (ticket 59). app.night-mode's name: the quick day/night
	// toggle beside the theme select, whose choices stay words.
	Add("Night mode",
		"Ночной режим", "Нічний режим", "Nachtmodus", "Modalità notte", "Modo nocturno",
		"Mode nuit", "Modo noturno",
		"الوضع الليلي", "नाइट मोड", "নাইট মোড", "نائٹ موڈ", "夜间模式")

	// action.reset qualified by its object (the catalog qualifies by target the same way):
	// the button returns the text size to the shipped default, not every setting.
	Add("Reset text size",
		"Сбросить размер текста", "Скинути розмір тексту", "Textgröße zurücksetzen",
		"Reimposta dimensione testo", "Restablecer el tamaño del texto",
		"Réinitialiser la taille du texte", "Redefinir o tamanho do texto",
		"إعادة تعيين حجم النص", "टेक्स्ट का आकार रीसेट करें", "টেক্সটের আকার রিসেট করুন",
		"ٹیکسٹ کا سائز ری سیٹ کریں", "重置文字大小")

	Add("Line spacing",
		"Межстрочный интервал", "Міжрядковий інтервал", "Zeilenabstand", "Interlinea",
		"Interlineado", "Interligne", "Espaçamento entre linhas",
		"تباعد الأسطر", "पंक्ति अंतराल", "লাইন স্পেসিং", "سطروں کا فاصلہ", "行距")

	Add("Column width",
		"Ширина колонки", "Ширина колонки", "Spaltenbreite", "Larghezza della colonna",
		"Ancho de columna", "Largeur de colonne", "Largura da coluna",
		"عرض العمود", "कॉलम की चौड़ाई", "কলামের প্রস্থ", "کالم کی چوڑائی", "栏宽")

	// The first option of the spacing and width selects: each surface's own shipped measure.
	Add("Default",
		"По умолчанию", "Типово", "Standard", "Predefinito", "Predeterminado",
		"Par défaut", "Padrão",
		"افتراضي", "डिफ़ॉल्ट", "ডিফল্ট", "ڈیفالٹ", "默认")

	// Column width options; they agree with "column" where the language marks gender.
	Add("Narrow",
		"Узкая", "Вузька", "Schmal", "Stretta", "Estrecha", "Étroite", "Estreita",
		"ضيقة", "संकरी", "সরু", "تنگ", "窄")

	Add("Normal",
		"Обычная", "Звичайна", "Normal", "Normale", "Normal", "Normale", "Normal",
		"عادية", "सामान्य", "সাধারণ", "عام", "标准")

	Add("Wide",
		"Широкая", "Широка", "Breit", "Larga", "Ancha", "Large", "Larga",
		"واسعة", "चौड़ी", "চওড়া", "چوڑی", "宽")

	Add("Full width",
		"Во всю ширину", "На всю ширину", "Volle Breite", "A tutta larghezza",
		"Ancho completo", "Pleine largeur", "Largura total",
		"العرض الكامل", "पूरी चौड़ाई", "পুরো প্রস্থ", "مکمل چوڑائی", "全宽")

	// The image-page zoom mode (comics, scans): the page fills the window's width.
	Add("Fit width",
		"По ширине окна", "За шириною вікна", "An Fensterbreite anpassen",
		"Adatta alla larghezza", "Ajustar al ancho", "Ajuster à la largeur",
		"Ajustar à largura",
		"ملاءمة العرض", "चौड़ाई पर फ़िट करें", "প্রস্থে ফিট করুন", "چوڑائی پر فٹ کریں", "适应窗口宽度")

	// The progress bar's accessible name; the readout on hover or focus is its number form.
	Add("Reading progress",
		"Прогресс чтения", "Прогрес читання", "Lesefortschritt", "Avanzamento della lettura",
		"Progreso de lectura", "Progression de la lecture", "Progresso da leitura",
		"تقدم القراءة", "पढ़ने की प्रगति", "পড়ার অগ্রগতি", "پڑھنے کی پیش رفت", "阅读进度")
}
