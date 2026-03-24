package connector

import "testing"

func TestSanitizeURI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			"mongodb://localhost:27017",
			"mongodb://localhost:27017",
		},
		{
			"mongodb://user:pass@localhost:27017",
			"mongodb://***:***@localhost:27017",
		},
		{
			"mongodb+srv://admin:secret@cluster.example.com/db",
			"mongodb+srv://***:***@cluster.example.com/db",
		},
	}
	for _, tt := range tests {
		got := sanitizeURI(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeURI(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestNew_InvalidURI(t *testing.T) {
	// Empty URI should return an error immediately.
	_, err := New(nil, "", "", "", "")
	if err == nil {
		t.Fatal("expected error for empty URI")
	}
}
