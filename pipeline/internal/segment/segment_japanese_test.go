package segment

import "testing"

// TestSegmentJapanese specifies the target behavior for a real Japanese
// Stage A — none of this passes yet (SegmentJapanese just delegates to
// the English-only Segment, see its own doc comment). Run this against
// whatever's implemented so far to see exactly which cases still fail;
// that diff is the spec for what's left to build.
//
// Hand-written sentences for now (deliberately simple, unambiguous
// grammar — same reasoning as Segment's own "basic multi-sentence"
// case).
func TestSegmentJapanese(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{
			// The core gap: Japanese sentences end in 。, not '.', and —
			// unlike English — there's no whitespace between one sentence
			// and the next for a boundary check to lean on at all.
			name: "basic multi-sentence, split on 。 with no whitespace between sentences",
			text: "猫が机の上に座った。犬は速く走った。今日雨が降ったか。",
			want: []string{
				"猫が机の上に座った。",
				"犬は速く走った。",
				"今日雨が降ったか。",
			},
		},
		{
			name: "、(comma) does not split, same as English commas",
			text: "雨が降ったが、彼は出かけた。",
			want: []string{
				"雨が降ったが、彼は出かけた。",
			},
		},
		{
			// Japanese dialogue quotes with 。 conventionally placed
			// inside the closing 」 — mirrors Segment's own "two dialogue
			// sentences do split" case.
			name: "「」 dialogue quotes: two quoted sentences do split",
			text: "「ごめんなさい。」と彼女は言った。「分かりません。」",
			want: []string{
				"「ごめんなさい。」と彼女は言った。",
				"「分かりません。」",
			},
		},
		{
			name: "fullwidth ？ and ！ also end a sentence",
			text: "本当ですか？信じられない！",
			want: []string{
				"本当ですか？",
				"信じられない！",
			},
		},
		{
			// Same rule as English's "[Illustration: ...]" case — a
			// paragraph break is a forced boundary even with no terminal
			// punctuation at all. A chapter heading is the realistic
			// Japanese analogue.
			name: "paragraph break forces a boundary even with no terminal punctuation",
			text: "第一章\n\n昔々、あるところに小さな村があった。",
			want: []string{
				"第一章",
				"昔々、あるところに小さな村があった。",
			},
		},
		{
			// Real source text: the opening three paragraphs of 羅生門
			// (Rashōmon) by 芥川龍之介 (Akutagawa Ryūnosuke) — Gutenberg
			// #1982, see pipeline/catalogs/JP_EN.txt. Pasted in with its
			// original hard line-wraps preserved rather than pre-joined
			// (same reasoning as TestSegment's Dracula excerpt): this is
			// what Stage A actually receives, wrap artifacts included, and
			// a wrap lands mid-sentence far more often than mid-boundary,
			// so each wrapped sentence's expected text keeps its embedded
			// newline exactly where the source has it.
			name: "羅生門 (Akutagawa Ryūnosuke) excerpt, opening three paragraphs",
			text: `或日の暮方の事である。一人の下人が、羅生門の下で雨やみを待っていた。　広い門
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
ある。`,

			want: []string{
				"或日の暮方の事である。",
				"一人の下人が、羅生門の下で雨やみを待っていた。",
				`広い門
の下には、この男の外に誰もいない。`,
				`ただ、所々丹塗の剥げた、大きな円柱に、きりぎ
りすが一匹とまっている。`,
				`羅生門が、朱雀大路にある以上は、この男の外にも、雨やみ
をする市女笠や揉烏帽子が、もう二三人はありそうなものである。`,
				`それが、この男の外
に誰もいない。`,
				`何故かと云うと、この二三年、京都には、地震とか辻風とか火事とか饑饉とか云う災
いがつづいて起こった。`,
				"そこで洛中のさびれ方は一通りでない。",
				`旧記によると、仏像や
仏具を打砕いて、その丹がついたり、金銀の箔（はく）がついたりした木を、路ばたに
つみ重ねて薪の料（しろ）に売っていたと云うことである。`,
				`洛中がその始末であるから、
羅生門の修理などは、元より誰も捨てて顧みる者がなかった。`,
				`するとその荒れ果てたの
をよい事にして、狐狸（こり）が棲む。`,
				"盗人が棲む。",
				`とうとうしまいには、引取り手の
ない死人を、この門へ持って来て、捨てて行くと云う習慣さえ出来た。`,
				`そこで、日の目
が見えなくなると、誰でも気味を悪がって、この門の近所へは足ぶみをしない事になっ
てしまったのである。`,
				"その代り又鴉が何処からか、たくさん集まって来た。",
				`昼間見ると、その鴉が何羽とな
く輪を描いて、高い鴟尾（しび）のまわりを啼きながら、飛びまわっている。`,
				`殊に門の
上の空が、夕焼けであかくなる時には、それが胡麻をまいたようにはっきり見えた。`,
				`鴉
は、勿論、門の上にある死人の肉を、啄みに来るのである。`,
				`ーー尤も今日は、刻限が遅
いせいか、一羽も見えない。`,
				`唯、所々、崩れかかった、そうしてその崩れ目に長い草の
はえた石段の上に、鴉の糞（くそ）が、点々と白くこびりついているのが見える。`,
				`下人
は七段ある石段の一番上の段に洗いざらした紺の襖（あお）の尻を据えて、右の頬に出
来た、大きな面皰（にきび）を気にしながら、ぼんやり、雨のふるのを眺めているので
ある。`,
			},
		},
		{
			name: "empty input",
			text: "",
			want: nil,
		},
		{
			name: "whitespace-only input",
			text: "   \n\t  ",
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SegmentJapanese(tc.text)

			gotText := make([]string, len(got))
			for i, s := range got {
				gotText[i] = s.Text
			}
			if !equalStrings(gotText, tc.want) {
				t.Fatalf("SegmentJapanese mismatch:\n%s", diffSentences(tc.want, gotText))
			}

			for i, s := range got {
				if s.Index != i {
					t.Errorf("sentence %d: Index = %d, want %d", i, s.Index, i)
				}
				if s.CharCount != len([]rune(s.Text)) {
					t.Errorf("sentence %d: CharCount = %d, want %d (len of %q)", i, s.CharCount, len([]rune(s.Text)), s.Text)
				}
			}
		})
	}
}
