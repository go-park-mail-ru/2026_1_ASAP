package message

import (
	"encoding/json"
	"testing"
)

func TestMessageAttachmentDTO_MarshalJSON_PhotoBlur(t *testing.T) {
	t.Parallel()
	photo := MessageAttachmentDTO{Type: "photo", IsBlur: true}
	b, err := json.Marshal(photo)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	blur, ok := m["is_blur"].(bool)
	if !ok || !blur {
		t.Fatalf("expected is_blur true, got %v", m["is_blur"])
	}
}

func TestMessageAttachmentDTO_MarshalJSON_VoiceNoBlur(t *testing.T) {
	t.Parallel()
	voice := MessageAttachmentDTO{Type: "voice", IsBlur: true}
	b, err := json.Marshal(voice)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["is_blur"]; ok {
		t.Fatalf("voice attachment must not include is_blur, got %v", m)
	}
}
