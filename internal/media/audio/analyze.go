package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	MaxVoiceDurationMs = 30000
	WaveformBars       = 100
	MaxVoiceBytes      = 5 * 1024 * 1024
)

var (
	ErrVoiceTooLong      = errors.New("voice message too long")
	ErrInvalidVoiceAudio = errors.New("invalid voice audio")
)

func AnalyzeVoice(content []byte, contentType string) (durationMs int, waveform []uint8, err error) {
	if len(content) == 0 {
		return 0, nil, ErrInvalidVoiceAudio
	}

	tmp, err := os.CreateTemp("", "voice-*"+extensionForContentType(contentType))
	if err != nil {
		return 0, nil, fmt.Errorf("create temp file: %w", err)
	}
	path := tmp.Name()
	defer func() { _ = os.Remove(path) }()

	if _, err = tmp.Write(content); err != nil {
		_ = tmp.Close()
		return 0, nil, fmt.Errorf("write temp file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return 0, nil, fmt.Errorf("close temp file: %w", err)
	}

	durationSec, err := probeDuration(path)
	if err != nil {
		return 0, nil, err
	}
	if durationSec <= 0 {
		return 0, nil, ErrInvalidVoiceAudio
	}

	durationMs = int(math.Round(durationSec * 1000))
	if durationMs > MaxVoiceDurationMs {
		return 0, nil, ErrVoiceTooLong
	}

	waveform, err = buildWaveform(path)
	if err != nil {
		return 0, nil, err
	}
	if len(waveform) != WaveformBars {
		return 0, nil, fmt.Errorf("unexpected waveform length: %d", len(waveform))
	}

	return durationMs, waveform, nil
}

func probeDuration(path string) (float64, error) {
	duration, err := probeFormatDuration(path)
	if err != nil {
		return 0, err
	}
	if duration > 0 {
		return duration, nil
	}

	duration, err = probeStreamDuration(path)
	if err != nil {
		return 0, err
	}
	if duration > 0 {
		return duration, nil
	}

	return probePacketDuration(path)
}

func probeFormatDuration(path string) (float64, error) {
	out, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidVoiceAudio, err)
	}

	return parseProbeDurationOutput(string(out))
}

func probeStreamDuration(path string) (float64, error) {
	out, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-count_packets",
		"-select_streams", "a:0",
		"-show_entries", "stream=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidVoiceAudio, err)
	}

	return parseProbeDurationOutput(string(out))
}

func probePacketDuration(path string) (float64, error) {
	out, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "packet=pts_time,duration_time",
		"-of", "csv=p=0",
		path,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidVoiceAudio, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return 0, ErrInvalidVoiceAudio
	}

	lastLine := strings.Trim(lines[len(lines)-1], ", \t")
	parts := strings.Split(lastLine, ",")
	if len(parts) == 0 {
		return 0, ErrInvalidVoiceAudio
	}

	pts, err := parseProbeFloat(parts[0])
	if err != nil {
		return 0, err
	}

	end := pts
	if len(parts) > 1 {
		if packetDuration, err := parseProbeFloat(parts[1]); err == nil {
			end += packetDuration
		}
	}
	if end <= 0 {
		return 0, ErrInvalidVoiceAudio
	}
	return end, nil
}

func parseProbeDurationOutput(out string) (float64, error) {
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		raw := strings.TrimSpace(line)
		if raw == "" || raw == "N/A" {
			continue
		}
		duration, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		if duration > 0 {
			return duration, nil
		}
	}
	return 0, nil
}

func parseProbeFloat(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "N/A" {
		return 0, ErrInvalidVoiceAudio
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: parse duration: %v", ErrInvalidVoiceAudio, err)
	}
	return value, nil
}

func buildWaveform(path string) ([]uint8, error) {
	cmd := exec.Command(
		"ffmpeg",
		"-i", path,
		"-ac", "1",
		"-ar", "8000",
		"-f", "s16le",
		"pipe:1",
	)
	cmd.Stderr = nil

	pcm, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: decode audio: %v", ErrInvalidVoiceAudio, err)
	}
	if len(pcm) < 2 {
		return nil, ErrInvalidVoiceAudio
	}

	sampleCount := len(pcm) / 2
	if sampleCount == 0 {
		return zerosWaveform(), nil
	}

	segmentSize := sampleCount / WaveformBars
	if segmentSize == 0 {
		segmentSize = 1
	}

	out := make([]uint8, WaveformBars)
	maxPeak := int16(1)

	peaks := make([]int16, WaveformBars)
	for i := 0; i < WaveformBars; i++ {
		start := i * segmentSize
		end := start + segmentSize
		if i == WaveformBars-1 {
			end = sampleCount
		}
		if start >= sampleCount {
			continue
		}
		if end > sampleCount {
			end = sampleCount
		}

		var peak int16
		for j := start; j < end; j++ {
			sample := int16(binary.LittleEndian.Uint16(pcm[j*2 : j*2+2]))
			abs := sample
			if abs < 0 {
				abs = -abs
			}
			if abs > peak {
				peak = abs
			}
		}
		peaks[i] = peak
		if peak > maxPeak {
			maxPeak = peak
		}
	}

	for i, peak := range peaks {
		out[i] = uint8((float64(peak) / float64(maxPeak)) * 255)
	}

	return out, nil
}

func zerosWaveform() []uint8 {
	out := make([]uint8, WaveformBars)
	return out
}

func extensionForContentType(contentType string) string {
	switch strings.ToLower(contentType) {
	case "audio/webm", "video/webm":
		return ".webm"
	case "audio/ogg":
		return ".ogg"
	case "audio/mp4", "audio/x-m4a":
		return ".m4a"
	case "audio/mpeg":
		return ".mp3"
	default:
		return ".bin"
	}
}

func FFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return false
	}
	_, err = exec.LookPath("ffprobe")
	return err == nil
}

// NormalizeWaveformForTest exposes normalization for unit tests without ffmpeg.
func NormalizeWaveformForTest(pcm []byte) []uint8 {
	if len(pcm) < 2 {
		return zerosWaveform()
	}
	sampleCount := len(pcm) / 2
	segmentSize := sampleCount / WaveformBars
	if segmentSize == 0 {
		segmentSize = 1
	}

	out := make([]uint8, WaveformBars)
	maxPeak := int16(1)
	peaks := make([]int16, WaveformBars)

	for i := 0; i < WaveformBars; i++ {
		start := i * segmentSize
		end := start + segmentSize
		if i == WaveformBars-1 {
			end = sampleCount
		}
		if start >= sampleCount {
			continue
		}
		if end > sampleCount {
			end = sampleCount
		}

		var peak int16
		for j := start; j < end; j++ {
			sample := int16(binary.LittleEndian.Uint16(pcm[j*2 : j*2+2]))
			abs := sample
			if abs < 0 {
				abs = -abs
			}
			if abs > peak {
				peak = abs
			}
		}
		peaks[i] = peak
		if peak > maxPeak {
			maxPeak = peak
		}
	}

	for i, peak := range peaks {
		out[i] = uint8((float64(peak) / float64(maxPeak)) * 255)
	}
	return out
}

// BuildPCMFixture creates little-endian s16le mono PCM for tests.
func BuildPCMFixture(samples []int16) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, len(samples)*2))
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}
	return buf.Bytes()
}
