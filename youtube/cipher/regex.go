package cipher

import (
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type regexStep struct {
	op  string // rev, spl, swp
	arg int
}

var (
	regexParseMu    sync.Mutex
	regexParseCache = make(map[string][]regexStep)
)

func cacheKeyForJS(playerJS string) string {
	h := sha1.Sum([]byte(playerJS))
	return hex.EncodeToString(h[:])
}

// tryRegexDecipher attempts to parse player.js and decipher signature without JS execution.
func tryRegexDecipher(playerJS string, signature string) (string, bool) {
	key := cacheKeyForJS(playerJS)

	regexParseMu.Lock()
	steps, ok := regexParseCache[key]
	regexParseMu.Unlock()

	if !ok {
		var parsed []regexStep
		// 1) Find decipher function body: a=a.split(""); ...; return a.join("")
		decipherBodyRe := regexp.MustCompile(`function\s*[a-zA-Z0-9$]*\s*\(\s*a\s*\)\s*{\s*a\s*=\s*a\.split\(\s*""\s*\);([\s\S]*?)return\s+a\.join\(\s*""\s*\)\s*}`)
		m := decipherBodyRe.FindStringSubmatch(playerJS)
		if len(m) < 2 {
			return "", false
		}
		body := m[1]

		// 2) Guess transform object name from call sites like OBJ.fn(a, n)
		objNameRe := regexp.MustCompile(`([a-zA-Z0-9$]+)\.[a-zA-Z0-9$]+\(a(?:,\s*\d+)?\)`) // first object name occurrence
		om := objNameRe.FindStringSubmatch(body)
		if len(om) < 2 {
			return "", false
		}
		obj := om[1]

		// 3) Extract transform object literal
		objRe := regexp.MustCompile(`var\s+` + regexp.QuoteMeta(obj) + `\s*=\s*\{([\s\S]*?)\}\s*;`)
		om2 := objRe.FindStringSubmatch(playerJS)
		if len(om2) < 2 {
			return "", false
		}
		objBody := om2[1]

		// 4) Map transform names to operations
		// Reverse: contains a.reverse()
		// Splice: contains a.splice(0,b)
		// Swap: pattern a[0]=a[b%a.length]
		funcRe := regexp.MustCompile(`([a-zA-Z0-9$]+)\s*:\s*function\(a(?:,b)?\)\s*\{([\s\S]*?)\}`)
		nameToOp := make(map[string]string)
		for _, fm := range funcRe.FindAllStringSubmatch(objBody, -1) {
			fname := fm[1]
			fbody := fm[2]
			if strings.Contains(fbody, ".reverse()") {
				nameToOp[fname] = "rev"
				continue
			}
			if strings.Contains(fbody, ".splice(") {
				nameToOp[fname] = "spl"
				continue
			}
			if strings.Contains(fbody, "a[0]=a[") && strings.Contains(fbody, "%a.length]") {
				nameToOp[fname] = "swp"
				continue
			}
		}
		if len(nameToOp) == 0 {
			return "", false
		}

		// 5) Parse call sequence from function body
		callRe := regexp.MustCompile(regexp.QuoteMeta(obj) + `\.([a-zA-Z0-9$]+)\(a(?:,\s*(\d+))?\)`) // captures fn and optional number
		calls := callRe.FindAllStringSubmatch(body, -1)
		if len(calls) == 0 {
			return "", false
		}
		for _, c := range calls {
			fn := c[1]
			op, ok := nameToOp[fn]
			if !ok {
				return "", false
			}
			arg := 0
			if len(c) >= 3 && c[2] != "" {
				if v, err := strconv.Atoi(c[2]); err == nil {
					arg = v
				}
			}
			parsed = append(parsed, regexStep{op: op, arg: arg})
		}

		regexParseMu.Lock()
		regexParseCache[key] = parsed
		regexParseMu.Unlock()
		steps = parsed
	}

	// Apply transforms
	r := []rune(signature)
	for _, st := range steps {
		switch st.op {
		case "rev":
			r = regexReverse(r)
		case "spl":
			r = regexSplice(r, st.arg)
		case "swp":
			r = regexSwap(r, st.arg)
		default:
			return "", false
		}
	}
	return string(r), true
}

func regexReverse(s []rune) []rune {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func regexSplice(s []rune, n int) []rune {
	if n < 0 || n > len(s) {
		return s
	}
	return s[n:]
}

func regexSwap(s []rune, n int) []rune {
	if len(s) <= 1 {
		return s
	}
	n = n % len(s)
	if n < 0 {
		n += len(s)
	}
	s[0], s[n] = s[n], s[0]
	return s
}
