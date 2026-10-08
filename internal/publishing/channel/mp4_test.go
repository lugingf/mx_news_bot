package channel

import (
	"encoding/binary"
	"testing"
)

func testBox(kind string, body ...[]byte) []byte {
	size := 8
	for _, part := range body {
		size += len(part)
	}
	out := make([]byte, 8, size)
	binary.BigEndian.PutUint32(out[0:4], uint32(size))
	copy(out[4:8], kind)
	for _, part := range body {
		out = append(out, part...)
	}

	return out
}

func testTrack(handler string, width, height int) []byte {
	tkhd := make([]byte, 84)
	binary.BigEndian.PutUint32(tkhd[76:80], uint32(width)<<16)
	binary.BigEndian.PutUint32(tkhd[80:84], uint32(height)<<16)
	hdlr := make([]byte, 24)
	copy(hdlr[8:12], handler)

	return testBox("trak", testBox("tkhd", tkhd), testBox("mdia", testBox("hdlr", hdlr)))
}

func testMP4(width, height, seconds int) []byte {
	mvhd := make([]byte, 100)
	binary.BigEndian.PutUint32(mvhd[12:16], 1000)
	binary.BigEndian.PutUint32(mvhd[16:20], uint32(seconds*1000))

	return append(
		testBox("ftyp", []byte("isom")),
		testBox("moov", testBox("mvhd", mvhd), testTrack("soun", 0, 0), testTrack("vide", width, height))...,
	)
}

func TestReadMP4VideoTakesTheVideoTrack(t *testing.T) {
	video, err := readMP4Video(testMP4(1920, 1080, 30))
	if err != nil {
		t.Fatalf("readMP4Video: %v", err)
	}
	if video != (mp4Video{Width: 1920, Height: 1080, Duration: 30}) {
		t.Fatalf("video = %+v", video)
	}
}

func TestReadMP4VideoWithoutMoov(t *testing.T) {
	if _, err := readMP4Video(testBox("ftyp", []byte("isom"))); err == nil {
		t.Fatal("expected an error")
	}
}
