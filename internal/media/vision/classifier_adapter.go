package vision

import "context"

type ClassifierAdapter struct {
	Detector *Detector
}

func (a *ClassifierAdapter) DetectBytes(ctx context.Context, data []byte) (bool, error) {
	if a == nil || a.Detector == nil {
		return false, nil
	}
	r, err := a.Detector.DetectBytes(ctx, data)
	return r.IsCapybara, err
}
