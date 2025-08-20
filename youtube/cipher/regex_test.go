package cipher

import "testing"

func TestTryRegexDecipher(t *testing.T) {
	js := `function X(a){a=a.split("");B.B0(a,1);B.x9(a,26);B.B0(a,3);return a.join("")};var B={B0:function(a){a.reverse()},x9:function(a,b){a.splice(0,b)},yG:function(a,b){var c=a[0];a[0]=a[b%a.length];a[b%a.length]=c}};`
	in := "ABCDEFGHIJKLMNabcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqr"
	out, ok := tryRegexDecipher(js, in)
	if !ok {
		t.Fatalf("regex decipher not applied")
	}
	// Apply expected manually: reverse -> splice(26) -> reverse
	runes := []rune(in)
	runes = regexReverse(runes)
	runes = regexSplice(runes, 26)
	runes = regexReverse(runes)
	expected := string(runes)
	if out != expected {
		t.Fatalf("got %q want %q", out, expected)
	}
}
