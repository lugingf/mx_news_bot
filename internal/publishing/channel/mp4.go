package channel

import (
	"encoding/binary"
	"errors"
	"math"
)

type mp4Video struct {
	Width    int
	Height   int
	Duration int
}

type mp4Box struct {
	kind string
	body []byte
}

func mp4Boxes(data []byte) []mp4Box {
	var boxes []mp4Box
	for len(data) >= 8 {
		size := uint64(binary.BigEndian.Uint32(data[0:4]))
		kind := string(data[4:8])
		header := uint64(8)
		if size == 1 {
			if len(data) < 16 {
				break
			}
			size = binary.BigEndian.Uint64(data[8:16])
			header = 16
		} else if size == 0 {
			size = uint64(len(data))
		}
		if size < header || size > uint64(len(data)) {
			break
		}
		boxes = append(boxes, mp4Box{kind: kind, body: data[header:size]})
		data = data[size:]
	}

	return boxes
}

func mp4Child(data []byte, kind string) ([]byte, bool) {
	for _, box := range mp4Boxes(data) {
		if box.kind == kind {
			return box.body, true
		}
	}

	return nil, false
}

// readMP4Video reads the picture size and the length of a clip from its moov box.
func readMP4Video(data []byte) (mp4Video, error) {
	moov, ok := mp4Child(data, "moov")
	if !ok {
		return mp4Video{}, errors.New("mp4: no moov box")
	}

	var video mp4Video
	if mvhd, ok := mp4Child(moov, "mvhd"); ok {
		var timescale, duration uint64
		if len(mvhd) >= 32 && mvhd[0] == 1 {
			timescale = uint64(binary.BigEndian.Uint32(mvhd[20:24]))
			duration = binary.BigEndian.Uint64(mvhd[24:32])
		} else if len(mvhd) >= 20 {
			timescale = uint64(binary.BigEndian.Uint32(mvhd[12:16]))
			duration = uint64(binary.BigEndian.Uint32(mvhd[16:20]))
		}
		if timescale > 0 {
			video.Duration = int(math.Round(float64(duration) / float64(timescale)))
		}
	}

	for _, trak := range mp4Boxes(moov) {
		if trak.kind != "trak" {
			continue
		}
		mdia, ok := mp4Child(trak.body, "mdia")
		if !ok {
			continue
		}
		hdlr, ok := mp4Child(mdia, "hdlr")
		if !ok || len(hdlr) < 12 || string(hdlr[8:12]) != "vide" {
			continue
		}
		tkhd, ok := mp4Child(trak.body, "tkhd")
		if !ok {
			continue
		}
		offset := 76
		if tkhd[0] == 1 {
			offset = 88
		}
		if len(tkhd) < offset+8 {
			continue
		}
		video.Width = int(binary.BigEndian.Uint32(tkhd[offset:offset+4]) >> 16)
		video.Height = int(binary.BigEndian.Uint32(tkhd[offset+4:offset+8]) >> 16)

		return video, nil
	}

	return mp4Video{}, errors.New("mp4: no video track")
}
