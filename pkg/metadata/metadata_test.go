package metadata

import "testing"

func Test_toS3HeaderKey(t *testing.T) {
	var tests = []struct {
		input string
		seq   int
		want  string
		err   bool
	}{
		{"", 1, "", true},
		{"123", 1, "", true},
		{"a", 2, "x-archive-meta02-a", false},
		{"zero_index", 0, "", true},
		{"abc-b_", 3, "x-archive-meta03-abc-b_", false},
		{"a", 123, "x-archive-meta123-a", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := toS3HeaderKey(tt.input, tt.seq)
			if (err != nil) != tt.err {
				t.Fatalf("want error %v, got %v", tt.err, err)
			}
			if got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func Test_toS3HeaderValue(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{"", ""},
		{"hello world", "uri(hello%20world)"},
		{"hell\no-\x12world", "uri(hell%0Ao-%EF%BF%BDworld)"},
		{"hello_world\x0b", "uri(hello_world%EF%BF%BD)"},
		{"hello+world", "hello+world"},
		{"hello%world\xef\xbf\xbe", "uri(hello%25world%EF%BF%BD)"},
		{"hello👋", "uri(hello%F0%9F%91%8B)"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := toS3HeaderValue(tt.s); got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}

func Test_ToS3Headers(t *testing.T) {

}
