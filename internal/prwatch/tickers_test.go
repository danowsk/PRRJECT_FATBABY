package prwatch

import "testing"

func TestExtractTickers(t *testing.T) {
	in := "( nasdaq : nvda ) and NYSE:BRK.B and (NYSE American: UEC) and TSX:SHOP"
	out := ExtractTickers(in)
	if len(out) != 4 {
		t.Fatalf("got %d", len(out))
	}
	if out[0].Exchange != "NASDAQ" || out[0].Symbol != "NVDA" {
		t.Fatalf("unexpected first %#v", out[0])
	}
}

func TestExtractFromHTMLMetaFirst(t *testing.T) {
	h := []byte(`<html><head><meta name="description" content="( nasdaq : nvda )"></head><body>NYSE:F</body></html>`)
	out := ExtractFromHTML(h)
	if len(out) != 1 || out[0].Symbol != "NVDA" {
		t.Fatalf("unexpected %+v", out)
	}
}

func BenchmarkExtractTickers(b *testing.B) {
	in := "Random text ( nasdaq : nvda ) plus NYSE:BRK.B TSX:SHOP OTCQX:ABCD"
	for i := 0; i < b.N; i++ {
		_ = ExtractTickers(in)
	}
}
