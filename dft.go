package main

import (
	"errors"
	"flag"
	"fmt"
	"gitlab.com/gomidi/midi/midimessage/channel"
	"gitlab.com/gomidi/midi/midimessage/meta"
	"gitlab.com/gomidi/midi/smf"
	"gitlab.com/gomidi/midi/smf/smfwriter"
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
var smplfreq = 44100.0
var smpls = 1024 * 8

func dft(wav []float64) []float64 {
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

func readwav(fp io.Reader, wavi []byte, wav []float64) error {
	n, err := io.ReadFull(fp, wavi)
	if err != nil {
		return err
	}
	if n != len(wavi) {
		return errors.New("")
	}
	for i := range wav {
		wav[i] = float64(int16(wavi[i*2]) | (int16(wavi[i*2+1]) << 8))
		wav[i] /= 32768.0
	}
	return nil
}

func eq(i int) float64 {
	switch {
	case i == 0:
		return 75.0
	case i == 1:
		return 50.0
	case i == 2:
		return 30.0
	case i == 3:
		return 18.0
	case i == 4:
		return 10.0
	case i == 5:
		return 15
	case i == 6:
		return 10
	case i == 7:
		return 10
	case i == 8:
		return 16.0
	case i == 9:
		return 15.0
	case i == 10:
		return 18.0
	case i == 11:
		return 4.0
	case i == 12:
		return 8.0
	case i == 13:
		return 6
	case i == 14:
		return 2
	case i == 15:
		return 8
	case i == 16:
		return 6.0
	case i == 17:
		return 10
	case i == 18:
		return 4
	case i == 19:
		return 3
	case i == 20:
		return 4
	case i == 21:
		return 0.9
	case i == 24:
		return 0.9
	case i == 28:
		return 1.5
	case i == 31:
		return 4
	case i == 32:
		return 20
	case i == 35:
		return 6.0
	default:
		return 1.0
	}
}

func reduceharm(spct []float64, i int) {
	for _, j := range []int{12, 12 + 7, 12 + 12, 12 + 12 + 7} {
		k := i + j
		if k < len(spct) {
			spct[k] -= spct[i] * 3.5
			if spct[k] < 0 {
				spct[k] = 0
			}
		}
	}
}

func reducenear(spct []float64, i int) {
	gain := []float64{0.03, 0.05, 0.1, 0.2, 0.3, 0.3, 0.2, 0.1, 0.05, 0.03}
	for x, j := range []int{-5, -4, -3, -2, -1, 1, 2, 3, 4, 5} {
		k := i + j
		if k >= 0 && k < len(spct) && spct[k] < spct[i] {
			spct[k] -= spct[i] * gain[x]
			if spct[k] < 0 {
				spct[k] = 0
			}
		}
	}
}

func sub(wav []float64, t int, wr smf.Writer) error {
	wav2 := make([]float64, smpls)
	for i := 0; i < smpls; i++ {
		wav2[i] = wav[i]
		wav2[i] *= 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(smpls))
	}
	spct := dft(wav2)
	for i, v := range spct {
		v *= eq(i)
		spct[i] = v
	}
	for i := range spct {
		reduceharm(spct, i)
		reducenear(spct, i)
	}
	noteon := make([]bool, 128)
	for i, v := range spct {
		db := 20 * math.Log10(v)
		if db > -40 {
			note := 40 + i
			fmt.Printf("%2d %2d %2d %4s %8.6f %6.2f dB ", t, i,
				note, note2str(note), v, db)
			for j := 0; j < (60+int(db))/2; j++ {
				fmt.Print("*")
			}
			fmt.Printf("\n")
			vel := db*2 + 192
			if vel < 1 {
				vel = 1
			}
			if vel > 127 {
				vel = 127
			}
			wr.Write(channel.Channel0.NoteOn(uint8(note), uint8(vel)))
			noteon[note] = true
		}
	}
	wr.SetDelta(480)
	for i, v := range noteon {
		if v {
			wr.Write(channel.Channel0.NoteOff(uint8(i)))
		}
	}
	return nil
}

func main() {
	fn := flag.String("f", "", "filename (.s16)")
	smf := flag.String("m", "output.mid", "filename (.mid)")
	flag.Parse()
	fp, err := os.Open(*fn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fp.Close()
	smffp, err := os.Create(*smf)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer smffp.Close()
	wr := smfwriter.New(smffp)
	wavi := make([]byte, smpls*2)
	wav := make([]float64, smpls)
	shift := 1024
	for i := 0; ; i++ {
		var err error
		if i == 0 {
			err = readwav(fp, wavi, wav)
		} else {
			copy(wav[:smpls-shift], wav[shift:])
			err = readwav(fp, wavi[:shift*2], wav[smpls-shift:])
		}
		if err != nil {
			break
		}
		sub(wav, i, wr)
	}
	wr.Write(meta.EndOfTrack)
}
