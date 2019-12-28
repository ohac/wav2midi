package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
)

func note2str(note int) string {
	if note < 40 {
		return "??"
	}
	note -= 40
	oct := (note+4)/12 + 2
	note %= 12
	strtbl := []string{"E", "F", "F#", "G", "G#", "A", "A#", "B", "C", "C#",
		"D", "D#"}
	return strtbl[note] + strconv.Itoa(oct)
}

var basehz = 440.0 / 8 // 55hz
var tw = 1.05946309    // 12sqrt(2)
var imin = 7
var imax = 7 + 12*4 + 1

func dft(wav []float64, smplfreq float64) []float64 {
	smpls := len(wav)
	spct := make([]float64, imax-imin)
	for i := imin; i < imax; i++ {
		freq := basehz * math.Pow(tw, float64(i))
		re := 0.0
		im := 0.0
		for j := 0; j < smpls; j++ {
			arg := -2 * math.Pi * freq / smplfreq * float64(j)
			re += math.Cos(arg) * wav[j]
			im += math.Sin(arg) * wav[j]
		}
		spct[i-imin] = math.Sqrt(re*re+im*im) / float64(smpls)
	}
	return spct
}

func readwav(smpls int, fn string) ([]float64, error) {
	wavi := make([]byte, smpls*2)
	wav := make([]float64, smpls)
	fp, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	n, err := io.ReadFull(fp, wavi)
	if err != nil {
		return nil, err
	}
	if n != len(wavi) {
		return nil, nil
	}
	for i := range wav {
		wav[i] = float64(int16(wavi[i*2]) | (int16(wavi[i*2+1]) << 8))
		wav[i] /= 32768.0
	}
	return wav, nil
}

func eq(i int) float64 {
	if i <= 2 {
		return 20.0
	} else if i <= 12 {
		return 1.2
	} else {
		return 1.0
	}
}

func reduceharm(spct []float64, i int) {
	for _, j := range []int{12, 12 + 7, 12 + 12, 12 + 12 + 7} {
		k := i + j
		if k < len(spct) {
			spct[k] -= spct[i] * 1.0
			if spct[k] < 0 {
				spct[k] = 0
			}
		}
	}
}

func main() {
	fn := flag.String("f", "", "filename (.s16)")
	flag.Parse()
	smplfreq := 44100.0
	smpls := 1024 * 32
	wav, _ := readwav(smpls, *fn)
	spct := dft(wav, smplfreq)
	for i, v := range spct {
		if v > 0.0002 {
			v *= eq(i)
			reduceharm(spct, i)
			db := 20 * math.Log10(v)
			if db > -50 {
				note := 40 + i
				fmt.Printf("%2d %4s %8.6f %6.2f dB ", note, note2str(note), v, db)
				for j := 0; j < (60+int(db))/2; j++ {
					fmt.Print("*")
				}
				fmt.Printf("\n")
			}
		}
	}
}
