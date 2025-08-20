package cipher

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func reverseRunes(r []rune) []rune {
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return r
}

func spliceRunes(r []rune, n int) []rune {
	if n < 0 || n > len(r) {
		return r
	}
	return r[n:]
}

func TestDecipherWithOtto(t *testing.T) {
	playerJSContent, err := os.ReadFile("testdata/player.js")
	if err != nil {
		t.Fatalf("Failed to read test player.js: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(playerJSContent)
	}))
	defer server.Close()

	// Example of an encrypted signature
	encryptedSig := "ABCDEFGHIJKLMNabcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqr"

	// Compute the expected value using the same steps: reverse -> splice(26) -> reverse
	r := []rune(encryptedSig)
	r = reverseRunes(r)
	r = spliceRunes(r, 26)
	r = reverseRunes(r)
	expectedSig := string(r)

	deciphered, err := Decipher(server.Client(), server.URL, encryptedSig)
	if err != nil {
		t.Fatalf("Decipher returned an error: %v", err)
	}

	if deciphered != expectedSig {
		t.Errorf("Decipher() got = %v, want %v", deciphered, expectedSig)
	}
}

func TestDecipherN(t *testing.T) {
	playerJSContent, err := os.ReadFile("testdata/player.js")
	if err != nil {
		t.Fatalf("Failed to read test player.js: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(playerJSContent)
	}))
	defer server.Close()

	in := "abcdef"
	want := "fedcba"
	got, err := DecipherN(server.Client(), server.URL, in)
	if err != nil {
		t.Fatalf("DecipherN error: %v", err)
	}
	if got != want {
		t.Fatalf("DecipherN got=%q want=%q", got, want)
	}
}
