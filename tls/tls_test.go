package tls

import (
	"context"
	"net/url"
	"testing"
)

func TestNewConn(t *testing.T) {
	ctx := context.TODO()
	u, _ := url.Parse("tls://8.8.4.4:853")

	tests := []struct {
		name    string
		u       url.URL
		wantNil bool
		wantErr bool
	}{
		{
			name:    "google",
			u:       *u,
			wantNil: false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, elapse, err := NewConn(ctx, tt.u)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("NewConn() got nil, wantNil %v", tt.wantNil)
				return
			}

			t.Logf("NewConn() of %s elapse %s", tt.u.Host, elapse)
			if err = got.Close(); err != nil {
				t.Errorf("NewConn() close error = %v", err)
				return
			}
		})
	}
}
