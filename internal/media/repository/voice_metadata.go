package repository

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	voiceMetaAttachmentKind = "attachment-kind"
	voiceMetaDurationMS     = "duration-ms"
	voiceMetaWaveform       = "waveform"
	voiceKindValue          = "voice"
)

type VoiceMetadata struct {
	DurationMs int
	Waveform   []uint8
	MimeType   string
	FileSize   int64
}

func encodeVoiceUserMetadata(durationMs int, waveform []uint8) (map[string]string, error) {
	wfJSON, err := json.Marshal(waveform)
	if err != nil {
		return nil, fmt.Errorf("marshal waveform: %w", err)
	}
	return map[string]string{
		voiceMetaAttachmentKind: voiceKindValue,
		voiceMetaDurationMS:     strconv.Itoa(durationMs),
		voiceMetaWaveform:       string(wfJSON),
	}, nil
}

func parseVoiceUserMetadata(meta map[string]string, contentType string, size int64) (*VoiceMetadata, error) {
	if meta == nil {
		return nil, fmt.Errorf("missing voice metadata")
	}
	if strings.ToLower(meta[voiceMetaAttachmentKind]) != voiceKindValue {
		return nil, fmt.Errorf("not a voice attachment")
	}

	durationMs, err := strconv.Atoi(meta[voiceMetaDurationMS])
	if err != nil || durationMs <= 0 {
		return nil, fmt.Errorf("invalid duration metadata")
	}

	var waveform []uint8
	if err = json.Unmarshal([]byte(meta[voiceMetaWaveform]), &waveform); err != nil {
		return nil, fmt.Errorf("invalid waveform metadata: %w", err)
	}

	return &VoiceMetadata{
		DurationMs: durationMs,
		Waveform:   waveform,
		MimeType:   contentType,
		FileSize:   size,
	}, nil
}

func userMetadataFromStat(meta map[string]string) map[string]string {
	out := make(map[string]string, len(meta))
	for k, v := range meta {
		key := strings.TrimPrefix(strings.ToLower(k), "x-amz-meta-")
		out[key] = v
	}
	return out
}
