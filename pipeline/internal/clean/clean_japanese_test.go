package clean

import "testing"

func TestCleanJapanese_StripsHeaderAndFooter(t *testing.T) {
	// Same Gutenberg wrapper as Clean's own test - this convention is
	// Gutenberg's own English-language transcription format, unaffected
	// by the book's language.
	raw := "The Project Gutenberg eBook of Some Book\n" +
		"\n" +
		"*** START OF THE PROJECT GUTENBERG EBOOK SOME BOOK ***\n" +
		"\n" +
		"或日の暮方の事である。\n" +
		"\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK SOME BOOK ***\n"

	got, err := CleanJapanese(raw)
	if err != nil {
		t.Fatalf("CleanJapanese: %v", err)
	}
	if got != "或日の暮方の事である。" {
		t.Errorf("CleanJapanese() = %q, want %q", got, "或日の暮方の事である。")
	}
}

func TestCleanJapanese_DewrapsWithoutInsertingASpace(t *testing.T) {
	// Mimics Gutenberg's fixed-width wrapping applied to Japanese
	// source text: a sentence broken mid-word across lines, the way
	// pipeline/catalogs/JP_EN.txt's 羅生門 (Rashōmon, Gutenberg #1982)
	// actually looks. Unlike Clean's equivalent English test
	// (TestClean_DewrapsHardWrappedLines), joining with a space would
	// be wrong here - Japanese doesn't put spaces between words at all.
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\n" +
		"或日の暮方の事である。一人の下人が、羅生門の下で雨やみを\n" +
		"待っていた。\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	got, err := CleanJapanese(raw)
	if err != nil {
		t.Fatalf("CleanJapanese: %v", err)
	}
	want := "或日の暮方の事である。一人の下人が、羅生門の下で雨やみを待っていた。"
	if got != want {
		t.Errorf("CleanJapanese() = %q, want %q", got, want)
	}
}

func TestCleanJapanese_PreservesParagraphBreaks(t *testing.T) {
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK X ***\n" +
		"第一章\n" +
		"\n" +
		"昔々、あるところに小さな村があった。\n" +
		"*** END OF THE PROJECT GUTENBERG EBOOK X ***"

	got, err := CleanJapanese(raw)
	if err != nil {
		t.Fatalf("CleanJapanese: %v", err)
	}
	want := "第一章\n\n昔々、あるところに小さな村があった。"
	if got != want {
		t.Errorf("CleanJapanese() = %q, want %q", got, want)
	}
}

// TestCleanJapanese_RealRashomonExcerpt runs CleanJapanese against the
// opening three paragraphs of 羅生門 (Rashōmon) by 芥川龍之介 (Akutagawa
// Ryūnosuke) — Gutenberg #1982, see pipeline/catalogs/JP_EN.txt — the
// same real excerpt segment_japanese_test.go's TestSegmentJapanese
// uses, here wrapped in a synthetic Gutenberg header/footer (the real
// downloaded file isn't checked in yet — see TestClean_RealPrideAndPrejudice
// for what that fixture looks like once it is).
//
// Notably, this excerpt's raw source has no *blank* line anywhere in
// it — each of what reads as three paragraphs is marked only by a
// leading 　(ideographic space) indent partway through a line, not a
// blank-line paragraph break. So the whole excerpt dewraps into one
// continuous block here, with no \n\n — that's correct given the real
// file's structure, not a bug. (The leading 　itself survives
// CleanJapanese untouched too, since it sits mid-line, not at a line
// boundary dewrapParagraphs trims; it's SegmentJapanese, not Clean,
// that later strips it as a sentence's own leading whitespace — see
// that package's tests.)
func TestCleanJapanese_RealRashomonExcerpt(t *testing.T) {
	raw := "*** START OF THE PROJECT GUTENBERG EBOOK RASHOMON ***\n" + `或日の暮方の事である。一人の下人が、羅生門の下で雨やみを待っていた。　広い門
の下には、この男の外に誰もいない。ただ、所々丹塗の剥げた、大きな円柱に、きりぎ
りすが一匹とまっている。羅生門が、朱雀大路にある以上は、この男の外にも、雨やみ
をする市女笠や揉烏帽子が、もう二三人はありそうなものである。それが、この男の外
に誰もいない。
　何故かと云うと、この二三年、京都には、地震とか辻風とか火事とか饑饉とか云う災
いがつづいて起こった。そこで洛中のさびれ方は一通りでない。旧記によると、仏像や
仏具を打砕いて、その丹がついたり、金銀の箔（はく）がついたりした木を、路ばたに
つみ重ねて薪の料（しろ）に売っていたと云うことである。洛中がその始末であるから、
羅生門の修理などは、元より誰も捨てて顧みる者がなかった。するとその荒れ果てたの
をよい事にして、狐狸（こり）が棲む。盗人が棲む。とうとうしまいには、引取り手の
ない死人を、この門へ持って来て、捨てて行くと云う習慣さえ出来た。そこで、日の目
が見えなくなると、誰でも気味を悪がって、この門の近所へは足ぶみをしない事になっ
てしまったのである。
　その代り又鴉が何処からか、たくさん集まって来た。昼間見ると、その鴉が何羽とな
く輪を描いて、高い鴟尾（しび）のまわりを啼きながら、飛びまわっている。殊に門の
上の空が、夕焼けであかくなる時には、それが胡麻をまいたようにはっきり見えた。鴉
は、勿論、門の上にある死人の肉を、啄みに来るのである。ーー尤も今日は、刻限が遅
いせいか、一羽も見えない。唯、所々、崩れかかった、そうしてその崩れ目に長い草の
はえた石段の上に、鴉の糞（くそ）が、点々と白くこびりついているのが見える。下人
は七段ある石段の一番上の段に洗いざらした紺の襖（あお）の尻を据えて、右の頬に出
来た、大きな面皰（にきび）を気にしながら、ぼんやり、雨のふるのを眺めているので
ある。` + "\n*** END OF THE PROJECT GUTENBERG EBOOK RASHOMON ***"

	got, err := CleanJapanese(raw)
	if err != nil {
		t.Fatalf("CleanJapanese: %v", err)
	}

	want := "或日の暮方の事である。一人の下人が、羅生門の下で雨やみを待っていた。　広い門の下には、この男の外に誰もいない。ただ、所々丹塗の剥げた、大きな円柱に、きりぎりすが一匹とまっている。羅生門が、朱雀大路にある以上は、この男の外にも、雨やみをする市女笠や揉烏帽子が、もう二三人はありそうなものである。それが、この男の外に誰もいない。何故かと云うと、この二三年、京都には、地震とか辻風とか火事とか饑饉とか云う災いがつづいて起こった。そこで洛中のさびれ方は一通りでない。旧記によると、仏像や仏具を打砕いて、その丹がついたり、金銀の箔（はく）がついたりした木を、路ばたにつみ重ねて薪の料（しろ）に売っていたと云うことである。洛中がその始末であるから、羅生門の修理などは、元より誰も捨てて顧みる者がなかった。するとその荒れ果てたのをよい事にして、狐狸（こり）が棲む。盗人が棲む。とうとうしまいには、引取り手のない死人を、この門へ持って来て、捨てて行くと云う習慣さえ出来た。そこで、日の目が見えなくなると、誰でも気味を悪がって、この門の近所へは足ぶみをしない事になってしまったのである。その代り又鴉が何処からか、たくさん集まって来た。昼間見ると、その鴉が何羽となく輪を描いて、高い鴟尾（しび）のまわりを啼きながら、飛びまわっている。殊に門の上の空が、夕焼けであかくなる時には、それが胡麻をまいたようにはっきり見えた。鴉は、勿論、門の上にある死人の肉を、啄みに来るのである。ーー尤も今日は、刻限が遅いせいか、一羽も見えない。唯、所々、崩れかかった、そうしてその崩れ目に長い草のはえた石段の上に、鴉の糞（くそ）が、点々と白くこびりついているのが見える。下人は七段ある石段の一番上の段に洗いざらした紺の襖（あお）の尻を据えて、右の頬に出来た、大きな面皰（にきび）を気にしながら、ぼんやり、雨のふるのを眺めているのである。"
	if got != want {
		t.Errorf("CleanJapanese() mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}
