package audio

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeWaveformForTest_Length(t *testing.T) {
	t.Parallel()

	samples := make([]int16, 8000)
	for i := range samples {
		samples[i] = int16(i % 300)
	}
	pcm := BuildPCMFixture(samples)
	wf := NormalizeWaveformForTest(pcm)
	require.Len(t, wf, WaveformBars)
}

func TestAnalyzeVoice_Empty(t *testing.T) {
	t.Parallel()

	_, _, err := AnalyzeVoice(nil, "audio/webm")
	require.ErrorIs(t, err, ErrInvalidVoiceAudio)
}

func TestParseProbeDurationOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		out     string
		want    float64
		wantErr bool
	}{
		{name: "duration", out: "N/A\n1.25\n", want: 1.25},
		{name: "skips invalid", out: "bad\n0\n2.5", want: 2.5},
		{name: "empty", out: "N/A\n\n", want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseProbeDurationOutput(tt.out)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseProbeFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    float64
		wantErr error
	}{
		{name: "valid", raw: " 3.5 ", want: 3.5},
		{name: "empty", raw: "", wantErr: ErrInvalidVoiceAudio},
		{name: "na", raw: "N/A", wantErr: ErrInvalidVoiceAudio},
		{name: "bad", raw: "abc", wantErr: ErrInvalidVoiceAudio},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseProbeFloat(tt.raw)
			if tt.wantErr != nil {
				require.True(t, errors.Is(err, tt.wantErr))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestExtensionForContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		contentType string
		want        string
	}{
		{contentType: "audio/webm", want: ".webm"},
		{contentType: "video/webm", want: ".webm"},
		{contentType: "audio/ogg", want: ".ogg"},
		{contentType: "audio/mp4", want: ".m4a"},
		{contentType: "audio/x-m4a", want: ".m4a"},
		{contentType: "audio/mpeg", want: ".mp3"},
		{contentType: "text/plain", want: ".bin"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.contentType, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, extensionForContentType(tt.contentType))
		})
	}
}

func TestZerosWaveform(t *testing.T) {
	t.Parallel()

	wf := zerosWaveform()
	require.Len(t, wf, WaveformBars)
	require.Equal(t, make([]uint8, WaveformBars), wf)
}

func TestAnalyzeVoice_WithFFmpeg(t *testing.T) {
	if !FFmpegAvailable() {
		t.Skip("ffmpeg/ffprobe not available")
	}

	out, err := execSilentWebM(t)
	require.NoError(t, err)

	durationMs, waveform, err := AnalyzeVoice(out, "audio/webm")
	require.NoError(t, err)
	require.Greater(t, durationMs, 0)
	require.LessOrEqual(t, durationMs, MaxVoiceDurationMs)
	require.Len(t, waveform, WaveformBars)
}

func TestAnalyzeVoice_LiveWebMWithoutHeaderDuration(t *testing.T) {
	if !FFmpegAvailable() {
		t.Skip("ffmpeg/ffprobe not available")
	}

	out, err := execLiveWebM(t)
	require.NoError(t, err)

	durationMs, waveform, err := AnalyzeVoice(out, "audio/webm")
	require.NoError(t, err)
	require.Greater(t, durationMs, 0)
	require.LessOrEqual(t, durationMs, MaxVoiceDurationMs)
	require.Len(t, waveform, WaveformBars)
}

func execSilentWebM(t *testing.T) ([]byte, error) {
	t.Helper()
	if !FFmpegAvailable() {
		t.Skip("ffmpeg not available")
	}

	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "anullsrc=r=8000:cl=mono", "-t", "0.5", "-f", "webm", "pipe:1")
	return cmd.Output()
}

func execLiveWebM(t *testing.T) ([]byte, error) {
	t.Helper()
	if !FFmpegAvailable() {
		t.Skip("ffmpeg not available")
	}

	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "anullsrc=r=8000:cl=mono", "-t", "0.5", "-f", "webm", "-live", "1", "pipe:1")
	return cmd.Output()
}
