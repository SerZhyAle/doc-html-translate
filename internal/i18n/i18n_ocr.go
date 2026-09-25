package i18n

// Console and GUI strings about installing OCR language data (internal/ocr/download.go).
//
// Order of the translations is always Codes[1:]: ru uk de it es fr pt ar hi bn ur zh.
func init() {
	Add("%q is not an OCR language this app offers. Choose one of: %s",
		"%q - не язык OCR, который предлагает это приложение. Выберите один из: %s",
		"%q - не мова OCR, яку пропонує цей застосунок. Виберіть одну з: %s",
		"%q ist keine OCR-Sprache, die diese App anbietet. Wählen Sie eine von: %s",
		"%q non è una lingua OCR offerta da questa app. Scegline una tra: %s",
		"%q no es un idioma de OCR que ofrezca esta aplicación. Elige uno de: %s",
		"%q n'est pas une langue OCR proposée par cette application. Choisissez parmi : %s",
		"%q não é um idioma de OCR oferecido por este aplicativo. Escolha um de: %s",
		"%q ليست لغة OCR يوفّرها هذا التطبيق. اختر واحدة من: %s",
		"%q इस ऐप द्वारा दी जाने वाली OCR भाषा नहीं है। इनमें से एक चुनें: %s",
		"%q এই অ্যাপের দেওয়া কোনো OCR ভাষা নয়। এগুলোর একটি বেছে নিন: %s",
		"%q اس ایپ کی پیش کردہ OCR زبان نہیں ہے۔ ان میں سے ایک منتخب کریں: %s",
		"%q 不是本应用提供的 OCR 语言。请从以下选择：%s")

	Add("Cannot write to the OCR language folder %s: %v",
		"Не удаётся записать в папку языков OCR %s: %v",
		"Не вдається записати до теки мов OCR %s: %v",
		"In den OCR-Sprachordner %s kann nicht geschrieben werden: %v",
		"Impossibile scrivere nella cartella delle lingue OCR %s: %v",
		"No se puede escribir en la carpeta de idiomas de OCR %s: %v",
		"Impossible d'écrire dans le dossier des langues OCR %s : %v",
		"Não é possível gravar na pasta de idiomas de OCR %s: %v",
		"تعذّرت الكتابة في مجلد لغات OCR %s: %v",
		"OCR भाषा फ़ोल्डर %s में लिखा नहीं जा सकता: %v",
		"OCR ভাষা ফোল্ডার %s-এ লেখা যাচ্ছে না: %v",
		"OCR زبانوں کے فولڈر %s میں لکھا نہیں جا سکتا: %v",
		"无法写入 OCR 语言文件夹 %s：%v")

	Add("Could not download the %s language data: %v",
		"Не удалось скачать языковые данные %s: %v",
		"Не вдалося завантажити мовні дані %s: %v",
		"Die Sprachdaten %s konnten nicht heruntergeladen werden: %v",
		"Impossibile scaricare i dati della lingua %s: %v",
		"No se pudieron descargar los datos del idioma %s: %v",
		"Impossible de télécharger les données de langue %s : %v",
		"Não foi possível baixar os dados do idioma %s: %v",
		"تعذّر تنزيل بيانات اللغة %s: %v",
		"%s भाषा डेटा डाउनलोड नहीं हो सका: %v",
		"%s ভাষার ডেটা ডাউনলোড করা যায়নি: %v",
		"%s زبان کا ڈیٹا ڈاؤن لوڈ نہیں ہو سکا: %v",
		"无法下载 %s 语言数据：%v")

	Add("The downloaded %s language data does not match the published file, so it was not installed. Try again later.",
		"Скачанные языковые данные %s не совпадают с опубликованным файлом, поэтому они не установлены. Повторите попытку позже.",
		"Завантажені мовні дані %s не збігаються з опублікованим файлом, тому їх не встановлено. Спробуйте пізніше.",
		"Die heruntergeladenen Sprachdaten %s stimmen nicht mit der veröffentlichten Datei überein und wurden daher nicht installiert. Versuchen Sie es später erneut.",
		"I dati della lingua %s scaricati non corrispondono al file pubblicato, quindi non sono stati installati. Riprova più tardi.",
		"Los datos del idioma %s descargados no coinciden con el archivo publicado, así que no se instalaron. Inténtalo más tarde.",
		"Les données de langue %s téléchargées ne correspondent pas au fichier publié ; elles n'ont donc pas été installées. Réessayez plus tard.",
		"Os dados do idioma %s baixados não correspondem ao arquivo publicado, por isso não foram instalados. Tente novamente mais tarde.",
		"بيانات اللغة %s التي نُزّلت لا تطابق الملف المنشور، لذا لم تُثبَّت. حاول مرة أخرى لاحقًا.",
		"डाउनलोड किया गया %s भाषा डेटा प्रकाशित फ़ाइल से मेल नहीं खाता, इसलिए इंस्टॉल नहीं किया गया। बाद में फिर कोशिश करें।",
		"ডাউনলোড করা %s ভাষার ডেটা প্রকাশিত ফাইলের সাথে মেলে না, তাই ইনস্টল করা হয়নি। পরে আবার চেষ্টা করুন।",
		"ڈاؤن لوڈ کیا گیا %s زبان کا ڈیٹا شائع شدہ فائل سے مطابقت نہیں رکھتا، اس لیے انسٹال نہیں کیا گیا۔ بعد میں دوبارہ کوشش کریں۔",
		"下载的 %s 语言数据与发布的文件不一致，因此未安装。请稍后重试。")
}
