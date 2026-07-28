package i18n

// Console strings of the CLI: the Windows-registration blocks, the first-run notice, the
// default-handler prompt and the closing pause. The welcome screen itself is not here - it is a
// per-language resource under internal/app/splash.
//
// Order of the translations is always Codes[1:]: ru uk de it es fr pt ar hi bn ur zh.
func init() {
	Add("  Windows registration: DONE",
		"  Регистрация в Windows: ВЫПОЛНЕНА",
		"  Реєстрація у Windows: ВИКОНАНА",
		"  Windows-Registrierung: FERTIG",
		"  Registrazione in Windows: FATTA",
		"  Registro en Windows: HECHO",
		"  Enregistrement dans Windows : TERMINÉ",
		"  Registro no Windows: CONCLUÍDO",
		"  التسجيل في Windows: تم",
		"  Windows में पंजीकरण: पूर्ण",
		"  Windows-এ নিবন্ধন: সম্পন্ন",
		"  Windows میں اندراج: مکمل",
		"  Windows 注册：完成")

	Add("  Program set as default handler for:",
		"  Программа назначена обработчиком по умолчанию для:",
		"  Програму призначено обробником за замовчуванням для:",
		"  Das Programm ist jetzt Standardanwendung für:",
		"  Il programma è ora l'applicazione predefinita per:",
		"  El programa es ahora la aplicación predeterminada para:",
		"  Le programme est désormais l'application par défaut pour :",
		"  O programa agora é o aplicativo padrão para:",
		"  تم تعيين البرنامج كبرنامج افتراضي لـ:",
		"  प्रोग्राम अब इनके लिए डिफ़ॉल्ट है:",
		"  প্রোগ্রামটি এখন এগুলোর জন্য ডিফল্ট:",
		"  پروگرام اب ان کے لیے طے شدہ ہے:",
		"  程序已设为以下类型的默认打开方式：")

	Add("  Double-clicking a file will now open it with this program.",
		"  Теперь двойной клик на файле открывает эту программу.",
		"  Тепер подвійний клік на файлі відкриває цю програму.",
		"  Ein Doppelklick auf eine Datei öffnet sie nun mit diesem Programm.",
		"  Ora un doppio clic su un file lo apre con questo programma.",
		"  Ahora un doble clic en un archivo lo abre con este programa.",
		"  Un double-clic sur un fichier l'ouvre maintenant avec ce programme.",
		"  Agora um clique duplo em um arquivo o abre com este programa.",
		"  الآن النقر المزدوج على ملف يفتحه بهذا البرنامج.",
		"  अब फ़ाइल पर डबल क्लिक करने से वह इसी प्रोग्राम में खुलेगी।",
		"  এখন ফাইলে ডাবল ক্লিক করলে তা এই প্রোগ্রামেই খুলবে।",
		"  اب فائل پر ڈبل کلک کرنے سے وہ اسی پروگرام میں کھلے گی۔",
		"  现在双击文件就会用本程序打开。")

	Add(`  Added a right-click "Convert to HTML" entry and "Open with" for:`,
		"  Добавлен пункт правого клика «Convert to HTML» и «Открыть с помощью» для:",
		"  Додано пункт правого кліку «Convert to HTML» та «Відкрити за допомогою» для:",
		`  Kontextmenüeintrag "Convert to HTML" und "Öffnen mit" hinzugefügt für:`,
		`  Aggiunta la voce "Convert to HTML" nel menu contestuale e "Apri con" per:`,
		`  Añadida la entrada "Convert to HTML" del menú contextual y "Abrir con" para:`,
		`  Entrée "Convert to HTML" du menu contextuel et "Ouvrir avec" ajoutées pour :`,
		`  Adicionados o item "Convert to HTML" do menu de contexto e "Abrir com" para:`,
		"  تمت إضافة أمر «Convert to HTML» في قائمة الزر الأيمن و«فتح باستخدام» لـ:",
		`  दायाँ-क्लिक मेनू में "Convert to HTML" और "इसके साथ खोलें" जोड़ा गया:`,
		`  ডান-ক্লিক মেনুতে "Convert to HTML" এবং "দিয়ে খুলুন" যোগ করা হয়েছে:`,
		`  دائیں کلک مینو میں "Convert to HTML" اور "کے ساتھ کھولیں" شامل کیا گیا:`,
		"  已为以下类型添加右键菜单项「Convert to HTML」和「打开方式」：")

	Add("  Your default handlers were NOT changed - association is optional.",
		"  Ассоциация по умолчанию НЕ изменена - это по желанию.",
		"  Асоціацію за замовчуванням НЕ змінено - це за бажанням.",
		"  Ihre Standardanwendungen wurden NICHT geändert - die Zuordnung ist freiwillig.",
		"  Le applicazioni predefinite NON sono state cambiate - l'associazione è facoltativa.",
		"  Las aplicaciones predeterminadas NO se han cambiado - la asociación es opcional.",
		"  Vos applications par défaut n'ont PAS été modifiées - l'association est facultative.",
		"  Os aplicativos padrão NÃO foram alterados - a associação é opcional.",
		"  لم يتم تغيير البرامج الافتراضية - الربط اختياري.",
		"  आपके डिफ़ॉल्ट प्रोग्राम बदले नहीं गए - यह वैकल्पिक है।",
		"  আপনার ডিফল্ট প্রোগ্রাম বদলানো হয়নি - এটি ঐচ্ছিক।",
		"  آپ کے طے شدہ پروگرام تبدیل نہیں کیے گئے - یہ اختیاری ہے۔",
		"  未更改您的默认打开方式 - 关联是可选的。")

	Add("Nothing to remove: the program was not the default handler.",
		"Нечего снимать: программа не была обработчиком по умолчанию.",
		"Нічого знімати: програма не була обробником за замовчуванням.",
		"Nichts zu entfernen: das Programm war nicht die Standardanwendung.",
		"Niente da rimuovere: il programma non era l'applicazione predefinita.",
		"Nada que quitar: el programa no era la aplicación predeterminada.",
		"Rien à retirer : le programme n'était pas l'application par défaut.",
		"Nada a remover: o programa não era o aplicativo padrão.",
		"لا شيء لإزالته: لم يكن البرنامج البرنامج الافتراضي.",
		"हटाने के लिए कुछ नहीं: प्रोग्राम डिफ़ॉल्ट नहीं था।",
		"সরানোর কিছু নেই: প্রোগ্রামটি ডিফল্ট ছিল না।",
		"ہٹانے کے لیے کچھ نہیں: پروگرام طے شدہ نہیں تھا۔",
		"无需移除：本程序不是默认打开方式。")

	Add("Default-handler association removed for:",
		"Ассоциация обработчика по умолчанию снята для:",
		"Асоціацію обробника за замовчуванням знято для:",
		"Standardzuordnung entfernt für:",
		"Associazione predefinita rimossa per:",
		"Asociación predeterminada eliminada para:",
		"Association par défaut supprimée pour :",
		"Associação padrão removida para:",
		"تمت إزالة الربط الافتراضي لـ:",
		"इनके लिए डिफ़ॉल्ट संबद्धता हटाई गई:",
		"এগুলোর জন্য ডিফল্ট সংযোগ সরানো হয়েছে:",
		"ان کے لیے طے شدہ وابستگی ہٹا دی گئی:",
		"已移除以下类型的默认关联：")

	Add("  Make DOC-HTML-TRANSLATE the default handler for these file types? [y/N]: ",
		"  Сделать DOC-HTML-TRANSLATE обработчиком по умолчанию для этих типов? [y/N]: ",
		"  Зробити DOC-HTML-TRANSLATE обробником за замовчуванням для цих типів? [y/N]: ",
		"  DOC-HTML-TRANSLATE zur Standardanwendung für diese Dateitypen machen? [j/N]: ",
		"  Rendere DOC-HTML-TRANSLATE l'applicazione predefinita per questi tipi? [s/N]: ",
		"  ¿Hacer que DOC-HTML-TRANSLATE sea la aplicación predeterminada para estos tipos? [s/N]: ",
		"  Faire de DOC-HTML-TRANSLATE l'application par défaut pour ces types ? [o/N] : ",
		"  Tornar o DOC-HTML-TRANSLATE o aplicativo padrão para esses tipos? [s/N]: ",
		"  هل تريد جعل DOC-HTML-TRANSLATE البرنامج الافتراضي لهذه الأنواع؟ [y/N]: ",
		"  क्या DOC-HTML-TRANSLATE को इन फ़ाइल प्रकारों के लिए डिफ़ॉल्ट बनाएँ? [y/N]: ",
		"  DOC-HTML-TRANSLATE কে এই ফাইল ধরনের জন্য ডিফল্ট করবেন? [y/N]: ",
		"  کیا DOC-HTML-TRANSLATE کو ان فائل اقسام کے لیے طے شدہ بنائیں؟ [y/N]: ",
		"  是否将 DOC-HTML-TRANSLATE 设为这些文件类型的默认打开方式？[y/N]: ")

	Add("  Press Enter to close.. (we both know you'll close the window anyway)",
		"  Нажмите Enter для закрытия.. (хотя вы всё равно закроете окно крестиком)",
		"  Натисніть Enter, щоб закрити.. (хоча ви все одно закриєте вікно хрестиком)",
		"  Zum Schließen Enter drücken.. (wir wissen beide, dass Sie das Fenster ohnehin schließen)",
		"  Premi Invio per chiudere.. (tanto la finestra la chiuderai comunque tu)",
		"  Pulsa Intro para cerrar.. (los dos sabemos que cerrarás la ventana igualmente)",
		"  Appuyez sur Entrée pour fermer.. (nous savons tous deux que vous fermerez la fenêtre)",
		"  Pressione Enter para fechar.. (nós dois sabemos que você vai fechar a janela mesmo assim)",
		"  اضغط Enter للإغلاق.. (كلانا يعلم أنك ستغلق النافذة على أي حال)",
		"  बंद करने के लिए Enter दबाएँ.. (वैसे भी आप विंडो खुद ही बंद कर देंगे)",
		"  বন্ধ করতে Enter চাপুন.. (আমরা দুজনেই জানি আপনি জানালাটা নিজেই বন্ধ করবেন)",
		"  بند کرنے کے لیے Enter دبائیں.. (ہم دونوں جانتے ہیں کہ آپ ونڈو خود ہی بند کریں گے)",
		"  按 Enter 关闭..（反正你也会直接关掉窗口）")

	// The affirmative answer in each language, so a user typing their own "yes" is understood.
	// "y"/"yes" are always accepted on top of these - the prompt shows [y/N] in most languages.
	Add("y", "д", "т", "j", "s", "s", "o", "s", "ن", "ह", "হ", "ج", "是")
	Add("yes", "да", "так", "ja", "sì", "sí", "oui", "sim", "نعم", "हाँ", "হ্যাঁ", "جی", "是的")
}
