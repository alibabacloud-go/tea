package dara

import (
	"testing"
)

func TestNewURL(t *testing.T) {
	tests := []struct {
		urlString string
		wantErr   bool
	}{
		{"http://example.com", false},
		{"ftp://user:pass@host:21/path", false},
		{"://example.com", true}, // Invalid URL
	}

	for _, tt := range tests {
		_, err := NewURL(tt.urlString)
		if (err != nil) != tt.wantErr {
			t.Errorf("NewURL(%q) error = %v, wantErr %v", tt.urlString, err, tt.wantErr)
		}
	}
}

func TestDaraURL_Path(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"http://example.com/path?query=1", "/path?query=1"},
		{"https://example.com/", "/"},
	}

	for _, tt := range tests {
		tu, _ := NewURL(tt.urlString)
		if got := tu.Path(); got != tt.want {
			t.Errorf("DaraURL.Path() = %v, want %v", got, tt.want)
		}
	}
}

func TestDaraURL_Pathname(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"http://example.com/path?query=1", "/path"},
		{"https://example.com/another/path/", "/another/path/"},
	}

	for _, tt := range tests {
		tu, _ := NewURL(tt.urlString)
		if got := tu.Pathname(); got != tt.want {
			t.Errorf("DaraURL.Pathname() = %v, want %v", got, tt.want)
		}
	}
}

func TestDaraURL_Protocol(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"http://example.com", "http"},
		{"ftp://example.com", "ftp"},
	}

	for _, tt := range tests {
		tu, _ := NewURL(tt.urlString)
		if got := tu.Protocol(); got != tt.want {
			t.Errorf("DaraURL.Protocol() = %v, want %v", got, tt.want)
		}
	}
}

func TestDaraURL_Hostname(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"http://example.com", "example.com"},
		{"https://user@subdomain.example.com:443", "subdomain.example.com"},
	}

	for _, tt := range tests {
		tu, _ := NewURL(tt.urlString)
		if got := tu.Hostname(); got != tt.want {
			t.Errorf("DaraURL.Hostname() = %v, want %v", got, tt.want)
		}
	}
}

func TestDaraURL_Host(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"http://example.com", "example.com"},
		{"http://example.com:8080", "example.com:8080"},
	}

	for _, tt := range tests {
		tu, _ := NewURL(tt.urlString)
		if got := tu.Host(); got != tt.want {
			t.Errorf("DaraURL.Host() = %v, want %v", got, tt.want)
		}
	}
}

func TestDaraURL_Port(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"http://example.com", "80"},
		{"https://example.com", "443"},
		{"ftp://example.com:21", "21"},
		{"gopher://example.com", "70"},
		{"ws://example.com", "80"},
		{"wss://example.com", "443"},
		{"http://example.com:8080", "8080"},
	}

	for _, tt := range tests {
		tu, _ := NewURL(tt.urlString)
		if got := tu.Port(); got != tt.want {
			t.Errorf("DaraURL.Port() = %v, want %v", got, tt.want)
		}
	}
}

func TestDaraURL_Hash(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"http://example.com#section", "section"},
		{"http://example.com", ""},
	}

	for _, tt := range tests {
		tu, _ := NewURL(tt.urlString)
		if got := tu.Hash(); got != tt.want {
			t.Errorf("DaraURL.Hash() = %v, want %v", got, tt.want)
		}
	}
}

func TestDaraURL_Search(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"http://example.com?query=1", "query=1"},
		{"http://example.com", ""},
	}

	for _, tt := range tests {
		tu, _ := NewURL(tt.urlString)
		if got := tu.Search(); got != tt.want {
			t.Errorf("DaraURL.Search() = %v, want %v", got, tt.want)
		}
	}
}

func TestDaraURL_Href(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"http://example.com", "http://example.com"},
		{"https://user:pass@host:443/path?query=1#section", "https://user:pass@host:443/path?query=1#section"},
	}

	for _, tt := range tests {
		tu, _ := NewURL(tt.urlString)
		if got := tu.Href(); got != tt.want {
			t.Errorf("DaraURL.Href() = %v, want %v", got, tt.want)
		}
	}
}

func TestDaraURL_Auth(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"http://user:pass@example.com", "user:pass"},
		{"http://user@example.com", "user:"},
		{"http://example.com", ""},
	}

	for _, tt := range tests {
		tu, _ := NewURL(tt.urlString)
		if got := tu.Auth(); got != tt.want {
			t.Errorf("DaraURL.Auth() = %v, want %v", got, tt.want)
		}
	}
}

func TestEncodeURL(t *testing.T) {
	tests := []struct {
		urlString string
		want      string
	}{
		{"hello world", "hello%20world"},
		{"test*abcd++", "test%2Aabcd%2B%2B"},
		{"", ""},
	}

	for _, tt := range tests {
		got := EncodeURL(tt.urlString)
		if got != tt.want {
			t.Errorf("EncodeURL(%q) = %v, want %v", tt.urlString, got, tt.want)
		}
	}
}

func TestPercentEncode(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"hello world", "hello%20world"},
		{"hello*world", "hello%2Aworld"},
		{"hello~world", "hello~world"},
		{"", ""},
	}

	for _, tt := range tests {
		got := PercentEncode(tt.raw)
		if got != tt.want {
			t.Errorf("PercentEncode(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestPathEncode(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"hello/world", "hello/world"},
		{"hello world", "hello%20world"},
		{"", ""},
		{"/", "/"},
	}

	for _, tt := range tests {
		got := PathEncode(tt.path)
		if got != tt.want {
			t.Errorf("PathEncode(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
