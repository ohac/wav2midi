package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
)

func freq2note(freq float64) int {
	// 1: E4 330 ... E6 1320
	// 6: E2  82
	table := []float64{82, 87, 92, 98, 104, 110, 117, 123, 131, 139, 147, 156,
		165, 175, 185, 196, 208, 220, 233, 247, 262, 277, 294, 311, 330, 349, 370,
		392, 415, 440, 466, 494, 523, 554, 587, 622, 659, 699, 740, 784, 831, 880,
		932, 988, 1046, 1108, 1174, 1245, 1320}
	diff := 2.0
	for i, v := range table {
		if freq > v-diff && freq < v+diff {
			return 40 + i
		}
	}
	return -1
}

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

func dft(wav []float64, smplfreq float64) []float64 {
	smpls := len(wav)
	spct := make([]float64, smpls)
	for i := 0; i < smpls; i++ {
		wav[i] *= 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(smpls))
	}
	for i := 0; i < smpls/8; i++ {
		freq := float64(i) * smplfreq / float64(smpls)
		if freq2note(freq) == -1 {
			continue
		}
		re := 0.0
		im := 0.0
		for j := 0; j < smpls; j++ {
			arg := -2 * math.Pi / float64(smpls) * float64(i*j)
			re += math.Cos(arg) * wav[j]
			im += math.Sin(arg) * wav[j]
		}
		spct[i] = math.Sqrt(re*re+im*im) / float64(smpls)
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

func eq(freq float64) float64 {
	if freq < 87 {
		return 15.0
	} else if freq < 95 {
		return 5.0
	} else if freq < 150 {
		return 10.0
	} else if freq < 200 {
		return 1.0
	} else if freq < 300 {
		return 1.0
	} else if freq < 400 {
		return 1.0
	} else if freq < 500 {
		return 10.0
	} else if freq < 600 {
		return 10.0
	} else if freq < 700 {
		return 10.0
	} else if freq < 800 {
		return 10.0
	} else if freq < 900 {
		return 10.0
	} else if freq < 1000 {
		return 10.0
	} else {
		return 20.0
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
		if i == smpls/8 {
			break
		}
		if v > 0.0002 {
			freq := float64(i) * smplfreq / float64(smpls)
			note := freq2note(freq)
			v *= eq(freq)
			db := 20 * math.Log10(v)
			if db > -50 {
				fmt.Printf("%4d %7.1f Hz %2d %4s %8.6f %6.2f dB ", i, freq, note,
					note2str(note), v, db)
				for j := 0; j < (60+int(db))/2; j++ {
					fmt.Print("*")
				}
				fmt.Printf("\n")
			}
		}
	}
}
