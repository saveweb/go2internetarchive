package utils

import "testing"

func Test_ReadKeysFromFile(t *testing.T) {
	tests := []struct {
		file       string
		wantAccKey string
		wantSecKey string
		wantErr    bool
	}{
		{"test_keys.txt", "access_key", "secret_key", false},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			gotAccKey, gotSecKey, gotErr := ReadKeysFromFile(tt.file)
			if (gotErr != nil) != tt.wantErr {
				t.Fatalf("want error %v, got %v", tt.wantErr, gotErr)
			}
			if gotAccKey != tt.wantAccKey {
				t.Fatalf("want %v, got %v", tt.wantAccKey, gotAccKey)
			}
			if gotSecKey != tt.wantSecKey {
				t.Fatalf("want %v, got %v", tt.wantSecKey, gotSecKey)
			}
		})
	}
}

func Test_SHA1SUM(t *testing.T) {
	tests := []struct {
		file    string
		wantSum string
		wantErr bool
	}{
		{"test_keys.txt", "0908539741466df32fa5fcb1843986ad1cbca24e", false},
		{"ahahahaha", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			gotSum, gotErr := SHA1SUM(tt.file)
			if (gotErr != nil) != tt.wantErr {
				t.Fatalf("want error %v, got %v", tt.wantErr, gotErr)
			}
			if gotSum != tt.wantSum {
				t.Fatalf("want %v, got %v", tt.wantSum, gotSum)
			}
		})
	}
}
