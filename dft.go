package main

import (
	"fmt"
	"io"
	"math"
	"os"
)

func dft(wav []float64) []float64 {
	smpls := len(wav)
	spct := make([]float64, smpls)
	for i := 0; i < smpls; i++ {
		wav[i] *= 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(smpls))
	}
	for i := 0; i < 128; i++ {
		re := 0.0
		im := 0.0
		for j := 0; j < smpls; j++ {
			arg := -2 * math.Pi / float64(smpls) * float64(i*j)
			re += math.Cos(arg) * wav[j]
			im += math.Sin(arg) * wav[j]
		}
		spct[i] = math.Sqrt(re*re + im*im)
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
		//fmt.Println(wav[i])
	}
	return wav, nil
}

func main() {
	smplfreq := 44100.0
	smpls := 4096
	fn := "e.s16" // sox a.flac a.s16
	wav, _ := readwav(smpls, fn)
	spct := dft(wav)
	for i, v := range spct {
		if i == smpls/8 {
			break
		}
		if v > 9.0 {
			freq := float64(i) * smplfreq / float64(smpls)
			db := 20 * math.Log10(v)
			// 1: E4 330 ... E6 1320
			// 6: E2  82
			fmt.Printf("%4d %7.1f %9.5f %9.5f\n", i, freq, v, db)
		}
	}
}
