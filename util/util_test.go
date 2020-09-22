package util

import "testing"

func TestSameSlice(t *testing.T) {
	testCases := []struct {
		name   string
		sliceA []string
		sliceB []string
		want   bool
	}{
		{"项目与顺序相同",
			[]string{"aaa", "bbb"}, []string{"aaa", "bbb"}, true},
		{"项目相同，但顺序不同",
			[]string{"aaa", "bbb"}, []string{"bbb", "aaa"}, true},
		{"有项目不同",
			[]string{"aaa", "bbb"}, []string{"aaa", "ccc"}, false},
		{"数量不同",
			[]string{"aaa", "bbb"}, []string{"aaa", "bbb", "ccc"}, false},
		{"都是空",
			[]string{}, []string{}, true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SameSlice(tc.sliceA, tc.sliceB); got != tc.want {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}
