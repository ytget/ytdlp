package cipher

import "testing"

func TestTryRegexDecipher(t *testing.T) {
	js := `function X(a){a=a.split("");a.reverse();a.splice(0,26);a.reverse();return a.join("")};`
	in := "ABCDEFGHIJKLMNabcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqr"
	out, ok := tryRegexDecipher(js, in)
	if !ok {
		t.Fatalf("regex decipher not applied")
	}
	// Apply expected manually: B.B0(a,1) -> B.x9(a,26) -> B.B0(a,3)
	runes := []rune(in)
	// B.B0(a,1) - reverse
	runes = regexReverse(runes)
	// B.x9(a,26) - splice(0,26)
	runes = regexSplice(runes, 26)
	// B.B0(a,3) - reverse
	runes = regexReverse(runes)
	expected := string(runes)
	if out != expected {
		t.Fatalf("got %q want %q", out, expected)
	}
}
