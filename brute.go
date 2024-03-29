package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"runtime/pprof"

	"github.com/spaolacci/murmur3"
)

const CURSUP = "\033[A"

type Progress struct {
	word    []rune
	counter int
}

// var CHARS = []rune("oenatvsilkrdpmuzjycbhfg")             // -xwq
var CHARS = []rune("oenatvsilkrdpímuázjyěcbéhřýžčšůfgúňxťóďwqχ") // -xťóďwq

var DECANON_MAP = map[rune]string{
	'a': "aá",
	'á': "aá",
	'b': "b",
	'c': "cčχ",
	'č': "cčχ",
	'χ': "cčχ",
	'd': "dď",
	'ď': "dď",
	'e': "eéě",
	'é': "eéě",
	'ě': "eéě",
	'f': "f",
	'g': "g",
	'h': "h",
	'i': "iíyý",
	'í': "iíyý",
	'y': "iíyý",
	'ý': "iíyý",
	'j': "j",
	'k': "k",
	'l': "l",
	'm': "m",
	'n': "nň",
	'ň': "nň",
	'o': "oó",
	'ó': "oó",
	'p': "p",
	'r': "rř",
	'ř': "rř",
	's': "sš",
	'š': "sš",
	't': "tť",
	'ť': "tť",
	'u': "uůú",
	'ů': "uůú",
	'ú': "uůú",
	'v': "vw",
	'w': "vw",
	'x': "x",
	'z': "zž",
	'ž': "zž",
}

func decanon(word []rune, hint []int) []rune {
	decanWord := make([]rune, len(word))
	copy(decanWord, word)
	for i, r := range decanWord {
		alts, exists := DECANON_MAP[r]
		ralts := []rune(alts)
		if exists {
			decanWord[i] = ralts[hint[i]%len(ralts)]
		}
	}
	return decanWord
}

var timeStart = time.Time{}

const HASH_BASE = 36

var WORKERS = 4

var HASH uint32 // hash to be compared against

var isSyllabes *regexp.Regexp
var tooMuchVowels *regexp.Regexp
var tooMuchConsonants *regexp.Regexp

func init() {
	WORKERS = runtime.NumCPU()/2 - 1
	if WORKERS < 2 {
		WORKERS = 2
	}

	if w := os.Getenv("WORKERS"); w != "" {
		if ww, err := strconv.ParseUint(w, 10, 32); err == nil {
			WORKERS = int(ww)
		}
	}

	isSyllabes = regexp.MustCompile(
		`^(((ch|[xwghkrdtnbflmpsvzžščřcjďťň]){0,3})[aáeéěiíyýoóuúůvlr][xwghkrdtnbflmpsvzžščřcjďťň]{0,3} ?)+$`,
	)
	tooMuchConsonants = regexp.MustCompile(`[xwghkdtnbfmpszžščřcjďťň]{5}`)
	tooMuchVowels = regexp.MustCompile(`[aáeéěiíyýoóuúů]{4}`)
}

func main() {
	if len(os.Args) != 3 && len(os.Args) != 4 {
		fmt.Println("invalid number of arguments")
		return
	}

	var wordLen = 5
	var templ = []rune{}

	hashString := os.Args[1]

	h, err := strconv.ParseUint(hashString, HASH_BASE, 32)
	if err != nil {
		fmt.Println("invalid murmur hashs - must be base %i", HASH_BASE)
		return
	}
	HASH = uint32(h)
	knownChars := 0

	// set templ and wordlen from second arg
	{
		l, err := strconv.ParseUint(os.Args[2], 10, 32)
		if err != nil { // template as arg
			wordLen = len([]rune(os.Args[2]))
			templ = make([]rune, wordLen)
			for i, c := range []rune(os.Args[2]) { // copy template and count known chars
				if c != '-' {
					templ[i] = c
					knownChars++
				}
			}
		} else { // size as arg
			wordLen = int(l)
			templ = make([]rune, wordLen)
		}
	}

	var decanonHint []int = nil
	if len(os.Args) > 3 { // decanon hint
		decanonHint = make([]int, len(os.Args[3]))
		for i, c := range os.Args[3] {
			l, _ := strconv.ParseUint(string(c), HASH_BASE, 32)
			decanonHint[i] = int(l)
		}
	}

	// test words from wordlist first
	// for _, word := range wordlist {
	// 	var buf = make([]byte, 3*2*len(word)) // each char is up to 2 bytes encoded in 3 chars
	// 	if murmur3.Sum32(escape([]rune(word), buf)) == HASH {
	// 		fmt.Printf("\nWordlist found: %s \n ", string(word))
	// 		return
	// 	}
	// }

	fmt.Printf("chars/wordlen: %v/%v \n\n\n", wordLen-knownChars, wordLen)

	if wordLen-knownChars > 10 {
		fmt.Printf("Too long - I give up")
		return
	}

	timeStart = time.Now()

	done := make(chan bool)
	allDone := make(chan bool)
	startChars := make(chan rune, len(CHARS))
	progresses := make([](chan Progress), WORKERS)
	var max = int(math.Pow(float64(len(CHARS)), float64(wordLen-knownChars))) // max iteration

	file, _ := os.Create("./cpu.pprof")
	pprof.StartCPUProfile(file)
	defer pprof.StopCPUProfile()
	defer file.Close()

	go func() { // output printer
		ticker := time.NewTicker(time.Second / 30)
		words := make([]string, WORKERS)
		counts := make([]int, WORKERS)

		done := false // used to stop the printer

		defer ticker.Stop()
		for {
			select {
			case <-allDone:
				done = true // set to stop in next ticker
			case <-ticker.C:
				counter := 0
				for i := 0; i < WORKERS; i++ {
					if prog, ok := <-progresses[i]; ok {
						word := string(prog.word)
						if utf8.RuneCountInString(word) == wordLen { // to prevent glitches caused by thread unsafe reading of words from workers
							words[i] = word
						}
						counts[i] = prog.counter
					}
					counter += counts[i]
				}
				unit := ""
				speed := float64(counter) / (time.Since(timeStart).Seconds())
				if speed > 1000 {
					speed /= 1000
					unit = "k"
					if speed > 1000 {
						speed /= 1000
						unit = "M"
					}
				}

				str := fmt.Sprintf(
					"%s%s> %s \n%s\t %s\t (%s)\n",
					CURSUP, CURSUP,
					string(strings.Join(words, " ")),                         // current combination
					fmt.Sprintf("%.3f%%", 100*float64(counter)/float64(max)), // percentages done
					eta(float64(counter)/float64(max)),                       // remaining time
					fmt.Sprintf("%.1f%s/s", speed, unit),
				)
				fmt.Printf("% -8s", str) // padding
				if done {                // finish
					allDone <- true
					return
				}
			}
		}
	}()

	stopChans := make([]chan bool, WORKERS)
	for i := 0; i < WORKERS; i++ {
		stopChans[i] = make(chan bool)
		progress := make(chan Progress)
		progresses[i] = progress
		go cracker(i, startChars, templ, progress, done, decanonHint, stopChans[i])
	}

	for _, ch := range CHARS {
		startChars <- ch // will be picked by first available cracker
	}
	close(startChars)

	if found := <-done; found {
		// stop other workers if someone found
		for i := 0; i < WORKERS; i++ {
			close(stopChans[i])
		}
	}
	allDone <- true // notify printer to finish
	<-allDone       // wait for printer to actually finish
	fmt.Printf("Finished in %s", runTime())
}

func escape(text []rune, t []byte) []byte { // WARNING only works on subset of unicode
	j := 0
	hex := "0123456789abcdef"
	for _, c := range text {
		if c == ' ' { // encode rune to utf8 coding and escape as encodeURI
			t[j] = '%'
			t[j+1] = hex[byte(c)>>4]
			t[j+2] = hex[byte(c)&15]
			j += 3
		} else if c > 0x7F { // 110xxxxx 10xxxxxx
			b2 := byte(0b10_000000 | (0b00_111111 & c))
			b1 := byte(0b110_00000 | (0b000_11111 & (c >> 6)))
			t[j] = '%'
			t[j+1] = hex[b1>>4]
			t[j+2] = hex[b1&15]
			j += 3
			t[j] = '%'
			t[j+1] = hex[b2>>4]
			t[j+2] = hex[b2&15]
			j += 3
		} else {
			t[j] = byte(c)
			j++
		}
	}
	return t[:j]
}

func eta(progress float64) string {
	sofar := time.Since(timeStart)
	d := time.Duration(float64(sofar)/progress - float64(sofar))
	return fmt.Sprintf("%vh %vm %vs",
		int(d.Seconds()/3600),
		int(d.Seconds()/60)%60,
		int(d.Seconds())%60,
	)
}

func runTime() string {
	d := time.Since(timeStart)
	return d.String()
}

func cracker(
	id int,
	chars chan rune,
	commonTempl []rune,
	progress chan Progress,
	done chan<- bool,
	decanonHint []int,
	stop chan bool,
) {
	found := false
	templ := make([]rune, len(commonTempl))
	copy(templ, commonTempl)
	wordLen := len(templ)

	var radix = len(CHARS)
	var indexes = make([]int, wordLen)
	var word = make([]rune, wordLen)
	var counter = 0

	defer func() {
		progress <- Progress{word, counter}
		close(progress)
		for range chars { // empty chars
		}
		done <- found
	}()

	if wordLen == 0 {
		return
	}

	charPos := 0
	for pos, ch := range templ {
		if ch == 0 {
			charPos = pos
		}
	}

	for ch := range chars {
		templ[charPos] = ch

		knownChars := 0

		// init know indexes by template
		for pos, ch := range templ {
			if ch != 0 {
				knownChars++
				indexes[pos] = -1
				for i, c := range CHARS {
					if c == ch {
						indexes[pos] = i
					}
				}
			} else {
				indexes[pos] = 0
			}
		}

		// fmt.Println(id, "XXX", ch, templ, indexes, word, wordLen, knownChars)

		var max = int(math.Pow(float64(radix), float64(wordLen-knownChars))) // max iteration

		var buf = make([]byte, 3*2*len(word)) // each char is up to 2 bytes encoded in 3 chars
		const modPrint = 1 << 16

		for i := 0; i < max; i++ { // each word
			for pos, ci := range indexes { // asign chars to word by indexes
				if ci == -1 {
					word[pos] = ' '
				} else {
					word[pos] = CHARS[ci]
				}
			}
			if i%modPrint == 0 {
				select {
				case <-stop:
					return
				default:
					select {
					case progress <- Progress{word, counter}:
					default:
						// nothing
					}
				}
			}
			counter++
			if murmur3.Sum32WithSeed(escape(word, buf), 0) == HASH { // check if word is matching
				note := ""
				isDecanon := decanonHint != nil && murmur3.Sum32WithSeed(escape(decanon(word, decanonHint), buf), 0) == HASH
				if isDecanon {
					note = "✔"
				}
				if decanonHint == nil {
					note = "×"
					if isPossibleWord(word) {
						note = "?"
					}
				}
				if note != "" {
					fmt.Printf("%s%sWord found: %s %s %s\n\n\n",
						CURSUP, CURSUP, strings.Replace(string(word), "χ", "ch", 1), note, spaces((wordLen+1)*(WORKERS-1)-11),
					)
					if isDecanon { // found for sure!
						found = true
						return
					}
				}
			}
			// increment indexes
			for pos, mod := 0, 1; pos < wordLen; pos++ {
				if templ[pos] != 0 { // skip index given by template
					continue
				}
				if i%mod == mod-1 {
					indexes[pos]++

					if indexes[pos] == radix {
						indexes[pos] = 0
					}
				}
				mod *= radix
			}
		}
	} // chars depleted

}

func spaces(n int) string {
	if n < 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

func isPossibleWord(word []rune) bool {

	w := []byte(string(word))
	return len(word) <= 4 || isSyllabes.Match(w) &&
		!tooMuchConsonants.Match(w) &&
		!tooMuchVowels.Match(w)
}
