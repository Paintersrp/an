package arg

import "testing"

func TestHandleContent(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		expect string
	}{
		{
			name:   "title only",
			args:   []string{"title"},
			expect: "",
		},
		{
			name:   "title and tags",
			args:   []string{"title", "tag1 tag2"},
			expect: "",
		},
		{
			name:   "title tags content",
			args:   []string{"title", "tag1 tag2", "content body"},
			expect: "content body",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := HandleContent(tc.args)
			if got != tc.expect {
				t.Fatalf("HandleContent(%v) = %q, want %q", tc.args, got, tc.expect)
			}
		})
	}
}
