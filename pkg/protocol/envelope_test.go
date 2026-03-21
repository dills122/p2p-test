package protocol

import "testing"

func TestEnvelopeValidate(t *testing.T) {
	valid := NewEnvelope(MessageTypePing, "node-1", []byte("hello"), DefaultTTL)

	tests := []struct {
		name    string
		env     Envelope
		wantErr bool
	}{
		{name: "valid", env: valid, wantErr: false},
		{name: "invalid version", env: func() Envelope {
			e := valid
			e.Version = "2"
			return e
		}(), wantErr: true},
		{name: "zero ttl", env: func() Envelope {
			e := valid
			e.TTL = 0
			return e
		}(), wantErr: true},
		{name: "missing id", env: func() Envelope {
			e := valid
			e.ID = ""
			return e
		}(), wantErr: true},
		{name: "missing type", env: func() Envelope {
			e := valid
			e.Type = ""
			return e
		}(), wantErr: true},
		{name: "missing origin", env: func() Envelope {
			e := valid
			e.Origin = ""
			return e
		}(), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.env.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnvelopeDecrementTTL(t *testing.T) {
	env := NewEnvelope(MessageTypePing, "node-1", []byte("hello"), 2)

	if ok := env.DecrementTTL(); !ok {
		t.Fatalf("expected ttl to remain valid after first decrement")
	}
	if env.TTL != 1 || env.HopCount != 1 {
		t.Fatalf("unexpected state after first decrement: ttl=%d hop=%d", env.TTL, env.HopCount)
	}

	if ok := env.DecrementTTL(); ok {
		t.Fatalf("expected ttl to expire after second decrement")
	}
	if env.TTL != 0 || env.HopCount != 2 {
		t.Fatalf("unexpected state after second decrement: ttl=%d hop=%d", env.TTL, env.HopCount)
	}
}
