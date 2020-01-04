# wav2midi
Convert guitar audio data to standard MIDI file.

## Status

- Go version: >= 1.12
- OS/architectures: everywhere Go runs (tested on Linux and Windows).

## Installation

```
go get gitlab.com/gomidi/midi@latest
go build
```

## Features

- [x] analysis notes using DFT
- [x] limit 6 notes

## Non-Goals

- [ ] pitch bend
- [ ] half note voicing

## Usage

```
sox guitartrack.flac -c 1 guitartrack.s16
./wav2midi -f guitartrack.s16
# convert to guitartrack.s16.mid
```

## Demo

- https://twitter.com/ohac/status/1212991307532001285

## License

MIT (see LICENSE file) 
