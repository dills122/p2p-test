package node

import "testing"

func TestResolveTransportForConfig(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "default empty", value: "", wantError: false},
		{name: "grpc", value: TransportGRPC, wantError: false},
		{name: "libp2p", value: TransportLibP2P, wantError: false},
		{name: "invalid", value: "unknown", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := ResolveTransportForConfig(tt.value)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error for %q", tt.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.value, err)
			}
			if transport == nil {
				t.Fatalf("expected transport instance for %q", tt.value)
			}
		})
	}
}
