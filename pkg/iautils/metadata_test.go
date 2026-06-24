package iautils

import (
	"fmt"
	"testing"
)

func Test_GetMetadataOnline(t *testing.T) {
	tests := []struct {
		identifier string
		wantErr    bool
	}{
		{"BiliBili-BV1Vt411V7iQ_p1-Q2V4V1VB", false},
	}
	for _, tt := range tests {
		t.Run(tt.identifier, func(t *testing.T) {
			got, gotErr := getMetadataOnline(tt.identifier)
			if (gotErr != nil) != tt.wantErr {
				t.Fatalf("want error %v, got %v", tt.wantErr, gotErr)
			}
			fmt.Printf("%+v\n", got)
		})
	}
}
