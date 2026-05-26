package jsonbody_test

import (
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/jsonbody"
	gatewaypayment "github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/payment"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "valid", body: `{"amount":19900,"subscription_days":30}`},
		{name: "invalid", body: `{bad`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got gatewaypayment.CreatePaymentRequest
			err := jsonbody.Decode(strings.NewReader(tt.body), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if got.Amount != 19900 || got.SubscriptionDays != 30 {
				t.Fatalf("Decode() = %+v", got)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{name: "valid", data: []byte(`{"amount":19900,"subscription_days":30}`)},
		{name: "invalid", data: []byte(`{bad`), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got gatewaypayment.CreatePaymentRequest
			err := jsonbody.Unmarshal(tt.data, &got)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if got.Amount != 19900 || got.SubscriptionDays != 30 {
				t.Fatalf("Unmarshal() = %+v", got)
			}
		})
	}
}
