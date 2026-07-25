// Package main demonstrates registering a TrueType font and drawing UTF-8 text with it.
package main

import (
	"context"
	"log"

	"github.com/avdoseferovic/paper/pkg/consts"
	"github.com/avdoseferovic/paper/pkg/core"

	"github.com/avdoseferovic/paper"
	"github.com/avdoseferovic/paper/pkg/decorator"

	"github.com/avdoseferovic/paper/pkg/fontrepository"

	"github.com/avdoseferovic/paper/pkg/components/text"
	"github.com/avdoseferovic/paper/pkg/consts/fontstyle"

	"github.com/avdoseferovic/paper/pkg/config"
	"github.com/avdoseferovic/paper/pkg/props"
)

func main() {
	m := GetPaper("docs/assets/fonts/arial-unicode-ms.ttf")
	document, err := m.Generate(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.Save("docs/assets/pdf/customfont.pdf")
	if err != nil {
		log.Fatal(err.Error())
	}

	err = document.GetReport().Save("docs/assets/text/customfont.txt")
	if err != nil {
		log.Fatal(err.Error())
	}
}

func GetPaper(customFontFile string) core.Paper {
	customFont := "arial-unicode-ms"

	customFonts, err := fontrepository.New().
		AddUTF8Font(customFont, fontstyle.Normal, customFontFile).
		AddUTF8Font(customFont, fontstyle.Italic, customFontFile).
		AddUTF8Font(customFont, fontstyle.Bold, customFontFile).
		AddUTF8Font(customFont, fontstyle.BoldItalic, customFontFile).
		Load()
	if err != nil {
		log.Fatal(err.Error())
	}

	builder := config.NewBuilder().
		WithCustomFonts(customFonts)

	cfg := builder.WithDefaultFont(&props.Font{Family: customFont}).
		Build()

	mrt := paper.New(cfg)
	m := decorator.NewMetrics(mrt)

	header, contents := getLanguageSample()

	m.AddRow(8,
		text.NewCol(4, header[0], props.Text{Style: fontstyle.Bold, Family: consts.FontFamilyArial, Align: consts.AlignCenter}),
		text.NewCol(8, header[1], props.Text{Style: fontstyle.Bold, Family: consts.FontFamilyArial, Align: consts.AlignCenter}),
	)

	grey := props.Color{Red: 200, Green: 200, Blue: 200}
	for i, content := range contents {
		r := m.AddRow(5,
			text.NewCol(4, content[0], props.Text{Align: consts.AlignCenter}),
			text.NewCol(8, content[1], props.Text{Align: consts.AlignCenter}),
		)

		if i%2 == 0 {
			r.WithStyle(&props.Cell{
				BackgroundColor: &grey,
			})
		}
	}

	longText := "聲音之道㣲矣天地有自然之聲人聲有自然之節古之聖人得其節之自然者而為之依永和聲至於八音諧而神人和胥是道也文字之作無不講求音韻顧南北異" +
		"其風土古今殊其轉變喉舌唇齒清濁輕重之分辨在毫釐動多訛舛樊然淆混不可究極自西域梵僧定字母為三十六分五音以總天下之聲而翻切之學興儒者若司馬光鄭樵" +
		"皆宗之其法有音和類隔互用借聲類例不一後人苦其委曲繁重難以驟曉往往以類隔互用之切改從音和而終莫能得其原也我聖祖仁皇帝"

	m.AddRows(text.NewRow(10, "long text without spaces", props.Text{
		Top:    5,
		Style:  fontstyle.Bold,
		Family: consts.FontFamilyArial,
	}))

	m.AddRow(80,
		text.NewCol(4, longText, props.Text{Align: consts.AlignCenter, BreakLineStrategy: consts.BreakLineDash}),
		text.NewCol(4, longText, props.Text{Align: consts.AlignLeft, BreakLineStrategy: consts.BreakLineDash}),
		text.NewCol(4, longText, props.Text{Align: consts.AlignRight, BreakLineStrategy: consts.BreakLineDash}),
	)

	return m
}

func getLanguageSample() ([]string, [][]string) {
	header := []string{"Language", "Phrase: Talk is cheap. Show me the code."}

	contents := [][]string{
		{"Africâner", "Praat is goedkoop. Wys my die kode."},
		{"Albanês", "Biseda është e lirë. Më trego kodin."},
		{"Alemão", "Reden ist billig. Zeig mir den Code."},
		{"Amárico", "ወሬ ርካሽ ነው ፡፡ ኮዱን አሳዩኝ ፡፡"},
		{"Árabe", "كلام رخيص. أرني الكود."},
		{"Armênio", "Խոսակցությունն էժան է: Showույց տվեք ինձ ծածկագիրը:"},
		{"Azerbaijano", "Danışıq ucuzdur. Kodu göstərin."},
		{"Basco", "Eztabaida merkea da. Erakutsi kodea."},
		{"Bengali", "টক সস্তা। আমাকে কোডটি দেখান"},
		{"Bielorusso", "Размовы танныя. Пакажыце мне код."},
		{"Birmanês", "ဟောပြောချက်ကစျေးပေါတယ် ကုဒ်ကိုပြပါ။"},
		{"Bósnio", "Govor je jeftin. Pokaži mi šifru."},
		{"Búlgaro", "Разговорите са евтини. Покажи ми кода."},
		{"Canarim", "ಮಾತುಕತೆ ಅಗ್ಗವಾಗಿದೆ. ನನಗೆ ಕೋಡ್ ತೋರಿಸಿ."},
		{"Catalão", "Parlar és barat. Mostra’m el codi."},
		{"Cazeque", "Сөйлесу арзан. Маған кодты көрсетіңіз."},
		{"Cebuano", "Barato ra ang sulti. Ipakita kanako ang code."},
		{"Chinês Simplificado", "谈话很便宜。给我看代码。"},
		{"Chinês Tradicional", "談話很便宜。給我看代碼。"},
		{"Cingalês", "කතාව ලාභයි. කේතය මට පෙන්වන්න."},
		{"Coreano", "토크는 싸다. 코드를 보여주세요."},
		{"Corso", "Parlà hè bonu. Mostrami u codice."},
		{"Croata", "Razgovor je jeftin. Pokaži mi šifru."},
		{"Curdo", "Axaftin erzan e. Kodê nîşanî min bidin."},
		{"Dinamarquês", "Tal er billig. Vis mig koden."},
		{"Eslovaco", "Hovor je lacný. Ukáž mi kód."},
		{"Esloveno", "Pogovor je poceni. Pokaži mi kodo."},
		{"Espanhol", "Hablar es barato. Enséñame el código."},
		{"Esperanto", "Babilado estas malmultekosta. Montru al mi la kodon."},
		{"Estoniano", "Rääkimine on odav. Näita mulle koodi."},
		{"Filipino", "Mura ang usapan. Ipakita sa akin ang code."},
		{"Finlandês", "Puhe on halpaa. Näytä koodi."},
		{"Francês", "Parler n'est pas cher. Montre-moi le code."},
		{"Frísio Ocidental", "Prate is goedkeap. Lit my de koade sjen."},
		{"Gaélico Escocês", "Tha còmhradh saor. Seall dhomh an còd."},
		{"Galego", "Falar é barato. Móstrame o código."},
		{"Galês", "Mae siarad yn rhad. Dangoswch y cod i mi."},
		{"Georgiano", "აუბარი იაფია. მაჩვენე კოდი."},
		{"Grego", "Η συζήτηση είναι φθηνή. Δείξε μου τον κωδικό."},
		{"Guzerate", "વાતો કરવી સસ્તી છે. મને કોડ બતાવો."},
		{"Haitiano", "Pale bon mache. Montre m kòd la."},
		{"Hauçá", "Magana tana da arha. Nuna min lambar."},
		{"Havaiano", "Kūʻai ke kamaʻilio. E hōʻike mai iaʻu i ke pāʻālua."},
		{"Hebraico", "הדיבורים זולים. הראה לי את הקוד."},
		{"Híndi", "बोलना आसान है। मुझे कोड दिखाओ।"},
		{"Hmong", "Kev hais lus yog pheej yig. Qhia kuv cov code."},
		{"Holandês", "Praten is goedkoop. Laat me de code zien."},
		{"Húngaro", "Beszélni olcsó. Mutasd meg a kódot."},
		{"Igbo", "Okwu dị ọnụ ala. Gosi m koodu."},
		{"Lídiche", "רעדן איז ביליק. ווייַזן מיר דעם קאָד."},
		{"Indonésio", "Berbicara itu murah. Tunjukkan kodenya."},
		{"Inglês", "Talk is cheap. Show me the code."},
		{"Iorubá", "Ọrọ jẹ olowo poku. Fi koodu naa han mi."},
		{"Irlandês", "Tá caint saor. Taispeáin dom an cód."},
		{"Islandês", "Tal er ódýrt. Sýndu mér kóðann."},
		{"Italiano", "Parlare è economico. Mostrami il codice."},
		{"Japonês", "口で言うだけなら簡単です。コードを見せてください。"},
		{"Javanês", "Omongan iku murah. Tampilake kode kasebut."},
		{"Khmer", "ការនិយាយគឺថោក។ បង្ហាញលេខកូដមកខ្ញុំ"},
		{"Laosiano", "ການສົນທະນາແມ່ນລາຄາຖືກ. ສະແດງລະຫັດໃຫ້ຂ້ອຍ."},
		{"Latim", "Disputatio vilis est. Ostende mihi codice."},
		{"Letão", "Saruna ir lēta. Parādiet man kodu."},
		{"Lituano", "Kalbėti pigu. Parodyk man kodą."},
		{"Luxemburguês", "Schwätzen ass bëlleg. Weist mir de Code."},
		{"Macedônio", "Зборувањето е ефтино. Покажи ми го кодот."},
		{"Malaiala", "സംസാരം വിലകുറഞ്ഞതാണ്. എനിക്ക് കോഡ് കാണിക്കുക."},
		{"Malaio", "Perbincangan murah. Tunjukkan kod saya."},
		{"Malgaxe", "Mora ny resaka. Asehoy ahy ny kaody."},
		{"Maltês", "It-taħdita hija rħisa. Urini l-kodiċi."},
		{"Maori", "He iti te korero. Whakaatuhia mai te tohu."},
		{"Marati", "चर्चा स्वस्त आहे. मला कोड दाखवा."},
		{"Mongol", "Яриа хямд. Надад кодоо харуул."},
		{"Nepalês", "कुरा सस्तो छ। मलाई कोड देखाउनुहोस्।"},
		{"Nianja", "Kulankhula ndikotsika mtengo. Ndiwonetseni nambala"},
		{"Norueguês", "Snakk er billig. Vis meg koden."},
		{"Oriá", "କଥାବାର୍ତ୍ତା ଶସ୍ତା ଅଟେ | ମୋତେ କୋଡ୍ ଦେଖାନ୍ତୁ |"},
		{"Panjabi", "ਗੱਲ ਸਸਤਾ ਹੈ. ਮੈਨੂੰ ਕੋਡ ਦਿਖਾਓ."},
		{"Pashto", "خبرې ارزانه دي. ما ته کوډ وښایاست"},
		{"Persa", "بحث ارزان است. کد را به من نشان دهید"},
		{"Polonês", "Rozmowa jest tania. Pokaż mi kod."},
		{"Português", "Falar é fácil. Mostre-me o código."},
		{"Quiniaruanda", "Ibiganiro birahendutse. Nyereka kode."},
		{"Quirguiz", "Сүйлөшүү арзан. Мага кодду көрсөтүңүз."},
		{"Romeno", "Vorbirea este ieftină. Arată-mi codul."},
		{"Russo", "Обсуждение дешево. Покажи мне код."},
		{"Samoano", "E taugofie talanoaga. Faʻaali mai le code."},
		{"Sérvio", "Причање је јефтино. Покажи ми шифру."},
		{"Sindi", "ڳالهه سستا آهي. مونکي ڪوڊ ڏيکاريو."},
		{"Somali", "Hadalku waa jaban yahay. I tus lambarka."},
		{"Soto do Sul", "Puo e theko e tlase. Mpontshe khoutu."},
		{"Suaíli", "Mazungumzo ni ya bei rahisi. Nionyeshe nambari."},
		{"Sueco", "Prat är billigt. Visa mig koden."},
		{"Sundanês", "Omongan mirah. Tunjukkeun kode na."},
		{"Tadjique", "Сӯҳбат арзон аст. Рамзро ба ман нишон диҳед."},
		{"Tailandês", "พูดคุยราคาถูก แสดงรหัส"},
		{"Tâmil", "பேச்சு மலிவானது. குறியீட்டை எனக்குக் காட்டு."},
		{"Tártaro", "Сөйләшү арзан. Миңа код күрсәтегез."},
		{"Tcheco", "Mluvení je levné. Ukaž mi kód."},
		{"Télugo", "చర్చ చౌకగా ఉంటుంది. నాకు కోడ్ చూపించు."},
		{"Turco", "Konuşma ucuz. Bana kodu göster."},
		{"Turcomeno", "Gepleşik arzan. Kody görkez"},
		{"Ucraniano", "Розмова дешева. Покажи мені код."},
		{"Uigur", "پاراڭ ئەرزان. ماڭا كودنى كۆرسەت."},
		{"Urdu", "بات گھٹیا ہے. مجھے کوڈ دکھائیں۔"},
		{"Uzbeque", "Gapirish arzon. Menga kodni ko'rsating."},
		{"Vietnamita", "Nói chuyện là rẻ. Cho tôi xem mã."},
		{"Xhosa", "Ukuthetha akubizi. Ndibonise ikhowudi."},
		{"Xona", "Kutaura kwakachipa. Ndiratidze kodhi."},
		{"Zulu", "Ukukhuluma kushibhile. Ngikhombise ikhodi."},
	}

	return header, contents
}
