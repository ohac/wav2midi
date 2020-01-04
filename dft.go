package main

import (
	"errors"
	"flag"
	"fmt"
	"gitlab.com/gomidi/midi/midimessage/channel"
	"gitlab.com/gomidi/midi/midimessage/meta"
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
var imin = 7           // E2
var imax = 7 + 12*7
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

func readwav(fp io.Reader, wavi []byte, wav []float64, gain float64) error {
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
		wav[i] *= gain
	}
	return nil
}

func guitar(noteon []bool, vels []uint8) {
	var max uint8
	maxi := 0

	// string 6: 0-4fret
	for i := 7; i < 7+5; i++ {
		v := vels[i]
		if v > max {
			max = v
			maxi = i
		}
	}
	for i := 7; i < 7+5; i++ {
		if i != maxi {
			noteon[i] = false
			vels[i] = 0
		}
	}

	// string 1: 18-23fret
	max = 0
	maxi = 0
	for i := 7 + 24 + 18; i < 7+24+24; i++ {
		v := vels[i]
		if v > max {
			max = v
			maxi = i
		}
	}
	for i := 7 + 24 + 18; i < 7+24+24; i++ {
		if i != maxi {
			noteon[i] = false
			vels[i] = 0
		}
	}

	for i := range vels[1:] {
		if vels[i] < vels[i+1] {
			noteon[i] = false
			vels[i] = 0
		} else if vels[i] > vels[i+1] {
			noteon[i+1] = false
			vels[i+1] = 0
		}
	}
}

func sub(wav []float64, t int, delta uint32) (uint32, []bool, []uint8) {
	wav2 := make([]float64, smpls)
	for i := 0; i < smpls; i++ {
		wav2[i] = wav[i]
		wav2[i] *= 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(smpls))
	}
	spct := dft(wav2)
	noteon := make([]bool, 128)
	vels := make([]uint8, 128)
	for i := 0; i < imax-imin-(12*2+4); i++ {
		v := spct[i]
		x2 := spct[i+12]
		x3 := spct[i+12+7]
		x4 := spct[i+12+12]
		x5 := spct[i+12+12+4]
		db := 20 * math.Log10(v)
		db2 := 20 * math.Log10(x2)
		db3 := 20 * math.Log10(x3)
		db4 := 20 * math.Log10(x4)
		db5 := 20 * math.Log10(x5)
		if db > threshold {
			db2 = db - db2
			db3 = db - db3
			db4 = db - db4
			db5 = db - db5
			note := 40 + i
			judge := false
			pr := false
			if i <= 9 { // makigen (string 5 and 6) 40-49
				judge = db2 < 32.0 && db3 < 42.0 && db4 < 50.0 && db5 < 60.0
			} else if i < 14 { // makigen (string 4) 50-53
				judge = db2 < 24.0 && db3 < 52.0 && db4 < 44.0 && db5 < 75.0
			} else if i < 24 { // 54-63
				judge = db2 < 28.0 && db3 < 69.0 && db4 < 64.0 && db5 < 93.0
			} else if i < 36 { // 64-75
				judge = db2 < 58.0 && db3 < 73.0 && db4 < 89.0 && db5 < 112.0
			} else { // 76-
				judge = db2 < 58.0 && db3 < 82.0 && db4 < 87.0 && db5 < 102.0
			}
			if pr {
				fmt.Printf(
					"%2d %2d %2d %4s %7.5f %6.2f dB %7.1f %7.1f %7.1f %7.1f %v\n",
					t, i, note, note2str(note), v, db, db2, db3, db4, db5, judge)
			}
			j2 := nojudge
			if j2 || judge {
				vel := db*velgain + veloffset
				if vel > 0 {
					if vel > 127 {
						vel = 127
					}
					noteon[note] = true
					vels[note] = uint8(vel)
				}
			}
		}
	}
	guitar(noteon, vels)
	return delta, noteon, vels
}

var (
	velgain   float64
	veloffset float64
	threshold float64
	verbose   bool
	nojudge   bool
)

func main() {
	fn := flag.String("f", "", "filename (.s16)")
	smfp := flag.String("m", "", "filename (.mid)")
	gain := flag.Float64("g", 1.0, "gain")
	smplfreqp := flag.Int("s", 44100, "sampling freq")
	velg := flag.Float64("v", 3.0, "velocity gain")
	velo := flag.Float64("o", 184, "velocity offset")
	thr := flag.Float64("t", -53, "threshold (dB)")
	verb := flag.Bool("V", false, "verbose")
	nojudgep := flag.Bool("n", false, "no judge (for debug)")
	flag.Parse()
	velgain = *velg
	veloffset = *velo
	verbose = *verb
	threshold = *thr
	nojudge = *nojudgep
	smplfreq = float64(*smplfreqp)
	smf := *smfp
	if smf == "" {
		smf = *fn + ".mid"
	}
	fp, err := os.Open(*fn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fp.Close()
	smffp, err := os.Create(smf)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer smffp.Close()
	wr := smfwriter.New(smffp)
	wavi := make([]byte, smpls*2)
	wav := make([]float64, smpls)
	shift := 1024 * 4
	lastnoteon2 := make([]bool, 128)
	lastnoteon := make([]bool, 128)
	var noteon []bool
	var vels, lastvels []uint8
	delta := uint32(0)
	delta2 := uint32(float64(shift) / smplfreq * 480 * 4)
	for i := 0; ; i++ {
		var err error
		if i == 0 {
			err = readwav(fp, wavi, wav, *gain)
		} else {
			copy(wav[:smpls-shift], wav[shift:])
			err = readwav(fp, wavi[:shift*2], wav[smpls-shift:], *gain)
		}
		if err != nil {
			break
		}
		delta, noteon, vels = sub(wav, i, delta+delta2)
		nstr := 0
		for j, v := range lastnoteon {
			if v && nstr < 6 {
				// check 8th, 8+5th, 8+8th, 8+8+3rd
				harm := false
				vel := int(vels[j]) + 10
				if j >= 12 {
					if j-12 <= 45 {
						harm = noteon[j-12] && vels[j-12] > 0 &&
							int(vels[j-12])+30 > int(vels[j])
					} else if j-12 <= 55 {
						harm = noteon[j-12] && vels[j-12] > 0 &&
							int(vels[j-12])+20 > int(vels[j])
					} else if j-12 <= 65 {
						harm = noteon[j-12] && int(vels[j-12])+10 > int(vels[j])
					} else if int(vels[j-12]) > vel {
						harm = noteon[j-12]
					}
				}
				if !harm && j >= 12+7 && int(vels[j-(12+7)]) > vel {
					harm = noteon[j-(12+7)]
				}
				if !harm && j >= 12+12 && int(vels[j-(12+12)]) > vel {
					harm = noteon[j-(12+12)]
				}
				if !harm && j >= 12+12+4 && int(vels[j-(12+12+4)]) > vel {
					harm = noteon[j-(12+12+4)]
				}
				if !harm {
					nstr++
					if lastnoteon2[j] && noteon[j] &&
						int(lastvels[j])+10 < int(vels[j]) {
						if delta > 0 {
							wr.SetDelta(delta)
							delta = 0
						}
						wr.Write(channel.Channel0.NoteOff(uint8(j)))
						wr.Write(channel.Channel0.NoteOn(uint8(j), lastvels[j]))
					} else if lastnoteon2[j] != v && v == noteon[j] {
						if delta > 0 {
							wr.SetDelta(delta)
							delta = 0
						}
						wr.Write(channel.Channel0.NoteOn(uint8(j), lastvels[j]))
					}
				}
			} else {
				if nstr >= 6 || lastnoteon2[j] != v {
					if delta > 0 {
						wr.SetDelta(delta)
						delta = 0
					}
					wr.Write(channel.Channel0.NoteOff(uint8(j)))
				}
			}
		}
		lastnoteon2 = lastnoteon
		lastnoteon = noteon
		lastvels = vels
	}
	for i, v := range lastnoteon2 {
		if v {
			if delta > 0 {
				wr.SetDelta(delta)
				delta = 0
			}
			wr.Write(channel.Channel0.NoteOff(uint8(i)))
		}
	}
	wr.Write(meta.EndOfTrack)
}
