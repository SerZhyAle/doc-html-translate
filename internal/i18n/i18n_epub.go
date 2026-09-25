package i18n

// Warnings the EPUB reader prints when a book names a file it cannot use. The item name comes
// first so a reader scanning the log sees which chapter is affected.
//
// Order of the translations is always Codes[1:]: ru uk de it es fr pt ar hi bn ur zh.
func init() {
	Add("Skipped book item %s: %v",
		"Пропущен элемент книги %s: %v",
		"Пропущено елемент книги %s: %v",
		"Buchelement %s übersprungen: %v",
		"Elemento del libro %s saltato: %v",
		"Elemento del libro %s omitido: %v",
		"Élément du livre %s ignoré : %v",
		"Item do livro %s ignorado: %v",
		"تم تخطي عنصر الكتاب %s: %v",
		"पुस्तक का आइटम %s छोड़ा गया: %v",
		"বইয়ের আইটেম %s বাদ দেওয়া হয়েছে: %v",
		"کتاب کا آئٹم %s چھوڑ دیا گیا: %v",
		"已跳过书籍条目 %s：%v")

	Add("Skipped book item %s: the file is not in the book",
		"Пропущен элемент книги %s: файла нет в книге",
		"Пропущено елемент книги %s: файлу немає в книзі",
		"Buchelement %s übersprungen: die Datei fehlt im Buch",
		"Elemento del libro %s saltato: il file non è nel libro",
		"Elemento del libro %s omitido: el archivo no está en el libro",
		"Élément du livre %s ignoré : le fichier est absent du livre",
		"Item do livro %s ignorado: o arquivo não está no livro",
		"تم تخطي عنصر الكتاب %s: الملف غير موجود في الكتاب",
		"पुस्तक का आइटम %s छोड़ा गया: फ़ाइल पुस्तक में नहीं है",
		"বইয়ের আইটেম %s বাদ দেওয়া হয়েছে: ফাইলটি বইয়ে নেই",
		"کتاب کا آئٹم %s چھوڑ دیا گیا: فائل کتاب میں موجود نہیں",
		"已跳过书籍条目 %s：书中没有该文件")

	Add("Book files %s and %s differ only in letter case; on Windows one replaces the other",
		"Файлы книги %s и %s различаются только регистром букв; в Windows один заменит другой",
		"Файли книги %s і %s відрізняються лише регістром літер; у Windows один замінить інший",
		"Die Buchdateien %s und %s unterscheiden sich nur in Groß-/Kleinschreibung; unter Windows ersetzt eine die andere",
		"I file del libro %s e %s differiscono solo per maiuscole/minuscole; su Windows uno sostituisce l'altro",
		"Los archivos del libro %s y %s solo difieren en mayúsculas/minúsculas; en Windows uno reemplaza al otro",
		"Les fichiers du livre %s et %s ne diffèrent que par la casse ; sous Windows, l'un remplace l'autre",
		"Os arquivos do livro %s e %s diferem apenas em maiúsculas/minúsculas; no Windows um substitui o outro",
		"ملفا الكتاب %s و%s يختلفان في حالة الأحرف فقط؛ في Windows يحل أحدهما محل الآخر",
		"पुस्तक की फ़ाइलें %s और %s केवल अक्षरों के केस में भिन्न हैं; Windows पर एक दूसरी की जगह ले लेती है",
		"বইয়ের ফাইল %s ও %s কেবল অক্ষরের কেসে আলাদা; Windows-এ একটি অন্যটিকে প্রতিস্থাপন করে",
		"کتاب کی فائلیں %s اور %s صرف حروف کے کیس میں مختلف ہیں؛ Windows پر ایک دوسری کی جگہ لے لیتی ہے",
		"书籍文件 %s 和 %s 仅大小写不同；在 Windows 上其中一个会覆盖另一个")
}
