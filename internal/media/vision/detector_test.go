package vision

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ASAP/config"
)

func TestClassifierAdapter_Disabled(t *testing.T) {
	t.Parallel()
	adapter := &ClassifierAdapter{
		Detector: NewDetector(config.CapybaraDetectorConfig{Enabled: false}, nil),
	}
	got, err := adapter.DetectBytes(context.Background(), []byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("expected false when detector disabled")
	}
}
