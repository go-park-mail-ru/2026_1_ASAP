package audio

import (
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

func TestAnalyzeVoice_WithFFmpeg(t *testing.T) {
	if !FFmpegAvailable() {
		t.Skip("ffmpeg/ffprobe not available")
	}

	// Generate 0.5s silence via ffmpeg.
	out, err := execSilentWebM(t)
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
