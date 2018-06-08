package response

import (
	"encoding/json"
	"testing"
)

func intPtr(i int) *int {
	return &i
}

func prettyJSON(o interface{}) (string, error) {
	bytes, err := json.MarshalIndent(o, "", "\t")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func TestNewPaginationMeta(t *testing.T) {
	type args struct {
		offset     int
		limit      int
		hasMore    bool
		currentURL string
	}
	tests := []struct {
		name     string
		args     args
		wantJSON string
	}{
		{
			name: "first page",
			args: args{0, 10, true, "http://foo.com/v1/bar"},
			wantJSON: `{
	"limit": 10,
	"current_offset": 0,
	"next_offset": 10,
	"next_url": "http://foo.com/v1/bar?offset=10"
}`,
		},
		{
			name: "second page",
			args: args{10, 10, true, "http://foo.com/v1/bar"},
			wantJSON: `{
	"limit": 10,
	"current_offset": 10,
	"next_offset": 20,
	"prev_offset": 0,
	"next_url": "http://foo.com/v1/bar?offset=20",
	"prev_url": "http://foo.com/v1/bar"
}`,
		},
		{
			name: "last page",
			args: args{40, 10, false, "http://foo.com/v1/bar"},
			wantJSON: `{
	"limit": 10,
	"current_offset": 40,
	"prev_offset": 30,
	"prev_url": "http://foo.com/v1/bar?offset=30"
}`,
		},
		{
			name: "misaligned start ending exactly on boundary",
			args: args{32, 10, false, "http://foo.com/v1/bar"},
			wantJSON: `{
	"limit": 10,
	"current_offset": 32,
	"prev_offset": 22,
	"prev_url": "http://foo.com/v1/bar?offset=22"
}`,
		},
		{
			name: "misaligned start partially through first page",
			args: args{5, 12, true, "http://foo.com/v1/bar"},
			wantJSON: `{
	"limit": 12,
	"current_offset": 5,
	"next_offset": 17,
	"prev_offset": 0,
	"next_url": "http://foo.com/v1/bar?offset=17",
	"prev_url": "http://foo.com/v1/bar"
}`,
		},
		{
			name: "no current URL",
			args: args{10, 10, true, ""},
			wantJSON: `{
	"limit": 10,
	"current_offset": 10,
	"next_offset": 20,
	"prev_offset": 0
}`,
		},
		{
			name: "#58 regression test",
			args: args{1, 3, true, ""},
			wantJSON: `{
	"limit": 3,
	"current_offset": 1,
	"next_offset": 4,
	"prev_offset": 0
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPaginationMeta(tt.args.offset, tt.args.limit, tt.args.hasMore,
				tt.args.currentURL)
			gotJSON, err := prettyJSON(got)
			if err != nil {
				t.Fatalf("failed to marshal PaginationMeta to JSON: %s", err)
			}
			if gotJSON != tt.wantJSON {
				// prettyJSON makes debugging easier due to the annoying pointer-to-ints, but it
				// also implicitly tests JSON marshalling as we can see if it's omitting fields etc.
				t.Fatalf("NewPaginationMeta() =\n%s\n  want:\n%s\n", gotJSON, tt.wantJSON)
			}
		})
	}
}
