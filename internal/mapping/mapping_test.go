package mapping

import "testing"

func TestBinarySearch(t *testing.T) {
	cases := []struct {
		name      string
		values    []int
		search    int
		wantLow   int
		wantHigh  int
		wantMatch *int
	}{
		{
			name:     "value between elements",
			values:   []int{1, 3, 5, 7, 9},
			search:   4,
			wantLow:  1,
			wantHigh: 2,
		},
		{
			name:     "value less than first element",
			values:   []int{1, 3, 5, 7, 9},
			search:   0,
			wantLow:  0,
			wantHigh: 0,
		},
		{
			name:     "value greater than last element",
			values:   []int{1, 3, 5, 7, 9},
			search:   10,
			wantLow:  4,
			wantHigh: 4,
		},
		{
			name:     "empty array",
			values:   []int{},
			search:   1,
			wantLow:  0,
			wantHigh: -1,
		},
		{
			name:      "value at start of array",
			values:    []int{1, 3, 5, 7, 9},
			search:    1,
			wantLow:   0,
			wantHigh:  0,
			wantMatch: intPtr(0),
		},
		{
			name:      "value at end of array",
			values:    []int{1, 3, 5, 7, 9},
			search:    9,
			wantLow:   4,
			wantHigh:  4,
			wantMatch: intPtr(4),
		},
		{
			name:      "single element array, value matches",
			values:    []int{1},
			search:    1,
			wantLow:   0,
			wantHigh:  0,
			wantMatch: intPtr(0),
		},
		{
			name:     "single element array, value does not match",
			values:   []int{1},
			search:   2,
			wantLow:  0,
			wantHigh: 0,
		},
		{
			name:      "two elements array, value matches first",
			values:    []int{1, 2},
			search:    1,
			wantLow:   0,
			wantHigh:  0,
			wantMatch: intPtr(0),
		},
		{
			name:      "two elements array, value matches second",
			values:    []int{1, 2},
			search:    2,
			wantLow:   1,
			wantHigh:  1,
			wantMatch: intPtr(1),
		},
		{
			name:     "two elements array, value does not match",
			values:   []int{1, 2},
			search:   3,
			wantLow:  1,
			wantHigh: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertBinarySearch(t, tc.values, tc.search, tc.wantLow, tc.wantHigh, tc.wantMatch)
		})
	}
}

func TestTranslateOffset(t *testing.T) {
	cases := []struct {
		name        string
		start       int
		fromOffsets []int
		toOffsets   []int
		fromLengths []int
		toLengths   []int
		want        int
		wantOk      bool
	}{
		{
			name:        "start within fromRange, offset within toRange",
			start:       5,
			fromOffsets: []int{1},
			toOffsets:   []int{11},
			fromLengths: []int{9},
			want:        15,
			wantOk:      true,
		},
		{
			name:        "start outside fromRange",
			start:       0,
			fromOffsets: []int{1},
			toOffsets:   []int{11},
			fromLengths: []int{9},
			want:        0,
			wantOk:      false,
		},
		{
			name:        "start at end of fromRange with shorter toLength",
			start:       10,
			fromOffsets: []int{1},
			toOffsets:   []int{11},
			fromLengths: []int{9},
			toLengths:   []int{7},
			want:        18,
			wantOk:      true,
		},
		{
			name:        "hits second segment",
			start:       12,
			fromOffsets: []int{0, 10},
			toOffsets:   []int{100, 200},
			fromLengths: []int{5, 5},
			want:        202,
			wantOk:      true,
		},
		{
			name:        "uses fromLengths when toLengths is empty",
			start:       3,
			fromOffsets: []int{1},
			toOffsets:   []int{11},
			fromLengths: []int{4},
			want:        13,
			wantOk:      true,
		},
		{
			name:        "mismatched lengths ignore extra segments",
			start:       12,
			fromOffsets: []int{0, 10},
			toOffsets:   []int{100},
			fromLengths: []int{5, 5},
			want:        0,
			wantOk:      false,
		},
		{
			name:        "empty inputs",
			start:       5,
			fromOffsets: nil,
			toOffsets:   nil,
			fromLengths: nil,
			toLengths:   nil,
			want:        0,
			wantOk:      false,
		},
		{
			name:        "start equals fromOffset",
			start:       10,
			fromOffsets: []int{10},
			toOffsets:   []int{50},
			fromLengths: []int{2},
			want:        50,
			wantOk:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertTranslateOffset(t, tc.start, tc.fromOffsets, tc.toOffsets, tc.fromLengths, tc.toLengths, tc.want, tc.wantOk)
		})
	}
}

func TestSourceMapLocations(t *testing.T) {
	mapping := Mapping{
		SourceOffsets:    []int{0, 10},
		GeneratedOffsets: []int{100, 110},
		Lengths:          []int{5, 5},
	}
	overlapMapping := Mapping{
		SourceOffsets:    []int{0},
		GeneratedOffsets: []int{100},
		Lengths:          []int{10},
	}

	cases := []struct {
		name      string
		mappings  []Mapping
		toSource  bool
		offset    int
		want      []int
		wantEmpty bool
	}{
		{
			name:      "empty mappings",
			mappings:  nil,
			toSource:  true,
			offset:    10,
			wantEmpty: true,
		},
		{
			name:     "to source location",
			mappings: []Mapping{mapping},
			toSource: true,
			offset:   102,
			want:     []int{2},
		},
		{
			name:     "to generated location",
			mappings: []Mapping{mapping},
			toSource: false,
			offset:   12,
			want:     []int{112},
		},
		{
			name:     "dedupes mapping across memo buckets",
			mappings: []Mapping{overlapMapping},
			toSource: true,
			offset:   105,
			want:     []int{5},
		},
		{
			name:     "no matching location",
			mappings: []Mapping{mapping},
			toSource: false,
			offset:   99,
			want:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sourceMap := NewSourceMap(tc.mappings)
			var locations []MappedLocation
			if tc.toSource {
				locations = sourceMap.ToSourceLocation(tc.offset)
			} else {
				locations = sourceMap.ToGeneratedLocation(tc.offset)
			}

			if tc.wantEmpty {
				if len(locations) != 0 {
					t.Fatalf("expected no locations, got %d", len(locations))
				}
				return
			}
			assertLocationsSet(t, locations, tc.want)
		})
	}
}

func TestSourceMapRanges(t *testing.T) {
	basicMapping := Mapping{
		SourceOffsets:    []int{0},
		GeneratedOffsets: []int{100},
		Lengths:          []int{5},
	}
	multiSegment := Mapping{
		SourceOffsets:    []int{0, 20},
		GeneratedOffsets: []int{100, 200},
		Lengths:          []int{5, 5},
	}
	generatedLengthMapping := Mapping{
		SourceOffsets:    []int{0},
		GeneratedOffsets: []int{100},
		Lengths:          []int{10},
		GeneratedLengths: []int{5},
	}
	mappingA := Mapping{
		SourceOffsets:    []int{0},
		GeneratedOffsets: []int{100},
		Lengths:          []int{5},
	}
	mappingB := Mapping{
		SourceOffsets:    []int{10},
		GeneratedOffsets: []int{200},
		Lengths:          []int{5},
	}
	reversedStart := Mapping{
		SourceOffsets:    []int{0},
		GeneratedOffsets: []int{200},
		Lengths:          []int{5},
	}
	reversedEnd := Mapping{
		SourceOffsets:    []int{10},
		GeneratedOffsets: []int{100},
		Lengths:          []int{5},
	}

	cases := []struct {
		name     string
		mappings []Mapping
		toSource bool
		start    int
		end      int
		fallback bool
		want     [][2]int
	}{
		{
			name:     "direct mapping to source",
			mappings: []Mapping{basicMapping},
			toSource: true,
			start:    100,
			end:      105,
			fallback: false,
			want:     [][2]int{{0, 5}},
		},
		{
			name:     "direct mapping to generated",
			mappings: []Mapping{basicMapping},
			toSource: false,
			start:    0,
			end:      5,
			fallback: false,
			want:     [][2]int{{100, 105}},
		},
		{
			name:     "start in segment, end outside without fallback",
			mappings: []Mapping{basicMapping},
			toSource: false,
			start:    2,
			end:      12,
			fallback: false,
			want:     nil,
		},
		{
			name:     "fallback maps across mappings",
			mappings: []Mapping{mappingA, mappingB},
			toSource: false,
			start:    2,
			end:      12,
			fallback: true,
			want:     [][2]int{{102, 202}},
		},
		{
			name:     "fallback skips reversed range",
			mappings: []Mapping{reversedStart, reversedEnd},
			toSource: false,
			start:    2,
			end:      12,
			fallback: true,
			want:     nil,
		},
		{
			name:     "generated lengths constrain mapped end",
			mappings: []Mapping{generatedLengthMapping},
			toSource: false,
			start:    2,
			end:      8,
			fallback: false,
			want:     [][2]int{{102, 105}},
		},
		{
			name:     "multiple segments in one mapping",
			mappings: []Mapping{multiSegment},
			toSource: false,
			start:    22,
			end:      24,
			fallback: false,
			want:     [][2]int{{202, 204}},
		},
		{
			name:     "no mappings",
			mappings: nil,
			toSource: true,
			start:    0,
			end:      1,
			fallback: true,
			want:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sourceMap := NewSourceMap(tc.mappings)
			var ranges []MappedRange
			if tc.toSource {
				ranges = sourceMap.ToSourceRange(tc.start, tc.end, tc.fallback)
			} else {
				ranges = sourceMap.ToGeneratedRange(tc.start, tc.end, tc.fallback)
			}
			assertRangesSet(t, ranges, tc.want)
		})
	}
}

func assertBinarySearch(t *testing.T, values []int, search int, wantLow int, wantHigh int, wantMatch *int) {
	t.Helper()

	low, high, match := BinarySearch(values, search)
	if low != wantLow || high != wantHigh {
		t.Fatalf("expected low=%d high=%d, got low=%d high=%d", wantLow, wantHigh, low, high)
	}

	if wantMatch == nil {
		return
	}

	if match != *wantMatch {
		t.Fatalf("expected match index %d, got match=%d", *wantMatch, match)
	}
}

func assertTranslateOffset(
	t *testing.T,
	start int,
	fromOffsets []int,
	toOffsets []int,
	fromLengths []int,
	toLengths []int,
	want int,
	wantOk bool,
) {
	t.Helper()

	got, ok := TranslateOffset(start, fromOffsets, toOffsets, fromLengths, toLengths)
	if ok != wantOk || (ok && got != want) {
		t.Fatalf("expected ok=%v offset=%d, got ok=%v offset=%d", wantOk, want, ok, got)
	}
}

func assertLocationsSet(t *testing.T, got []MappedLocation, want []int) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d locations, got %d", len(want), len(got))
	}
	seen := make([]bool, len(want))
	for _, item := range got {
		matched := false
		for i, w := range want {
			if seen[i] {
				continue
			}
			if item.Offset == w {
				seen[i] = true
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("unexpected location offset %d", item.Offset)
		}
	}
}

func assertRangesSet(t *testing.T, got []MappedRange, want [][2]int) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d ranges, got %d", len(want), len(got))
	}
	seen := make([]bool, len(want))
	for _, item := range got {
		matched := false
		for i, w := range want {
			if seen[i] {
				continue
			}
			if item.MappedStart == w[0] && item.MappedEnd == w[1] {
				seen[i] = true
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("unexpected range [%d, %d]", item.MappedStart, item.MappedEnd)
		}
	}
}

func intPtr(value int) *int {
	return &value
}
