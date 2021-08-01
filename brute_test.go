package main

import "testing"

func origEscape(text []rune) []byte {
	for _, r := range text {
		if r > 127 {
			var s = string(text)
			var t = make([]byte, 3*len(s))
			j := 0
			for i := 0; i < len(s); i++ {
				c := s[i]
				if c > 127 {
					t[j] = '%'
					t[j+1] = "0123456789abcdef"[c>>4]
					t[j+2] = "0123456789abcdef"[c&15]
					j += 3
				} else {
					t[j] = c
					j++
				}
			}
			return t[:j]
		}
	}
	return []byte(string(text))
}

func TestIsWord(t *testing.T) {
	for _, word := range wordlist {
		if !isWord([]rune(word)) {
			t.Errorf("%s should be matched as possible word but it is not", word)
		}
	}
}

func TestEscape(t *testing.T) {
	isSame := func(a, b []byte) bool {
		for i, ch := range a {
			if ch != b[i] {
				return false
			}
		}
		return len(a) == len(b)

	}
	data := map[string]string{
		"kočka":    "ko%c4%8dka",
		"kočička":  "ko%c4%8di%c4%8dka",
		"ščřžýáíé": "%c5%a1%c4%8d%c5%99%c5%be%c3%bd%c3%a1%c3%ad%c3%a9",
	}

	buf := make([]byte, 100)

	for key, val := range data {
		escaped := escape([]rune(key), buf)
		if !isSame([]byte(val), escaped) {
			t.Errorf("esape of %s is not %s but %v", key, string(escaped), val)
		}
	}
}

func BenchmarkEscape(b *testing.B) {
	buf := make([]byte, 100)
	for i := 0; i < b.N; i++ {
		for _, word := range wordlist {
			escape([]rune(word), buf)
		}
	}
}

func BenchmarkOldescape(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, word := range wordlist {
			origEscape([]rune(word))
		}
	}
}
