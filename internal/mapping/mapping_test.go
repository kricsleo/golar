package mapping

import (
	"strings"
	"testing"
)

func TestBinarySearch(t *testing.T) {
	t.Run("value between elements", func(t *testing.T) {
		assertBinarySearch(t, []int{1, 3, 5, 7, 9}, 4, 1, 2, nil)
	})
	t.Run("value less than first element", func(t *testing.T) {
		assertBinarySearch(t, []int{1, 3, 5, 7, 9}, 0, 0, 0, nil)
	})
	t.Run("value greater than last element", func(t *testing.T) {
		assertBinarySearch(t, []int{1, 3, 5, 7, 9}, 10, 4, 4, nil)
	})
	t.Run("empty array", func(t *testing.T) {
		assertBinarySearch(t, []int{}, 1, 0, -1, nil)
	})
	t.Run("value at start of array", func(t *testing.T) {
		match := 0
		assertBinarySearch(t, []int{1, 3, 5, 7, 9}, 1, 0, 0, &match)
	})
	t.Run("value at end of array", func(t *testing.T) {
		match := 4
		assertBinarySearch(t, []int{1, 3, 5, 7, 9}, 9, 4, 4, &match)
	})
	t.Run("single element array, value matches", func(t *testing.T) {
		match := 0
		assertBinarySearch(t, []int{1}, 1, 0, 0, &match)
	})
	t.Run("single element array, value does not match", func(t *testing.T) {
		assertBinarySearch(t, []int{1}, 2, 0, 0, nil)
	})
	t.Run("two elements array, value matches first", func(t *testing.T) {
		match := 0
		assertBinarySearch(t, []int{1, 2}, 1, 0, 0, &match)
	})
	t.Run("two elements array, value matches second", func(t *testing.T) {
		match := 1
		assertBinarySearch(t, []int{1, 2}, 2, 1, 1, &match)
	})
	t.Run("two elements array, value does not match", func(t *testing.T) {
		assertBinarySearch(t, []int{1, 2}, 3, 1, 1, nil)
	})
}

func TestTranslateOffset(t *testing.T) {
	t.Run("start within fromRange, offset within toRange", func(t *testing.T) {
		assertTranslateOffset(t, 5, 1, 11, 9, nil, 15, true)
	})
	t.Run("start outside fromRange", func(t *testing.T) {
		assertTranslateOffset(t, 0, 1, 11, 9, nil, 0, false)
	})
	t.Run("calculated offset outside toRange", func(t *testing.T) {
		assertTranslateOffset(t, 11, 1, 11, 9, nil, 0, false)
	})
	t.Run("start at beginning of fromRange", func(t *testing.T) {
		assertTranslateOffset(t, 1, 1, 11, 9, nil, 11, true)
	})
	t.Run("start at end of fromRange", func(t *testing.T) {
		assertTranslateOffset(t, 10, 1, 11, 9, nil, 20, true)
	})
	t.Run("start at the end of fromRange with shorter toLength", func(t *testing.T) {
		toLength := 7
		assertTranslateOffset(t, 10, 1, 11, 9, &toLength, 18, true)
	})
}

func TestMapperAngularTemplate(t *testing.T) {
	mapper := NewMapper([]Mapping{
		{
			SourceOffset: idx(`{{|data?.icon?.toString()}}`),
			ServiceOffset: idx(
				`(null as any ? ((null as any ? ((null as any ? (this.|data)!.icon : undefined)!.toString : undefined))!() : undefined)`,
			),
			Length: len(`data`),
		},
		{
			SourceOffset: idx(`{{data?.|icon?.toString()}}`),
			ServiceOffset: idx(
				`(null as any ? ((null as any ? ((null as any ? (this.data)!.|icon : undefined)!.toString : undefined))!() : undefined)`,
			),
			Length: len(`icon`),
		},
		{
			SourceOffset: idx(`{{data?.icon?.|toString()}}`),
			ServiceOffset: idx(
				`(null as any ? ((null as any ? ((null as any ? (this.data)!.icon : undefined)!.|toString : undefined))!() : undefined)`,
			),
			Length: len(`toString`),
		},
		{
			SourceOffset: idx(`{{data?.icon?.toString|()}}`),
			ServiceOffset: idx(
				`(null as any ? ((null as any ? ((null as any ? (this.data)!.icon : undefined)!.toString : undefined))!|() : undefined)`,
			),
			Length: len(`()`),
		},
	}, nil)

	ranges := mapper.ToServiceRange(
		idx(`{{|data?.icon?.toString()}}`),
		idx(`{{data|?.icon?.toString()}}`),
		false,
	)
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	assertRangeOffsets(
		t,
		ranges[0],
		idx(`(null as any ? ((null as any ? ((null as any ? (this.|data)!.icon : undefined)!.toString : undefined))!() : undefined)`),
		idx(`(null as any ? ((null as any ? ((null as any ? (this.data|)!.icon : undefined)!.toString : undefined))!() : undefined)`),
	)

	ranges = mapper.ToServiceRange(
		idx(`{{|data?.icon?.toString()}}`),
		idx(`{{data?.ic|on?.toString()}}`),
		false,
	)
	if len(ranges) != 0 {
		t.Fatalf("expected no ranges, got %d", len(ranges))
	}
}

func TestMapperFallbackToAnyMatch(t *testing.T) {
	mapper := NewMapper([]Mapping{
		{
			SourceOffset: idx(`{{|data?.icon?.toString()}}`),
			ServiceOffset: idx(
				`(null as any ? ((null as any ? (|(null as any ? (this.data)!.icon : undefined)!.toString : undefined))!() : undefined)`,
			),
			Length: 0,
		},
		{
			SourceOffset: idx(`{{data?.icon|?.toString()}}`),
			ServiceOffset: idx(
				`(null as any ? ((null as any ? ((null as any ? (this.data)!.icon : undefined)|!.toString : undefined))!() : undefined)`,
			),
			Length: 0,
		},
	}, nil)

	ranges := mapper.ToServiceRange(
		idx(`{{|data?.icon?.toString()}}`),
		idx(`{{data?.icon|?.toString()}}`),
		false,
	)
	if len(ranges) != 0 {
		t.Fatalf("expected no ranges, got %d", len(ranges))
	}

	ranges = mapper.ToServiceRange(
		idx(`{{|data?.icon?.toString()}}`),
		idx(`{{data?.icon|?.toString()}}`),
		true,
	)
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	assertRangeOffsets(
		t,
		ranges[0],
		idx(`(null as any ? ((null as any ? (|(null as any ? (this.data)!.icon : undefined)!.toString : undefined))!() : undefined)`),
		idx(`(null as any ? ((null as any ? ((null as any ? (this.data)!.icon : undefined)|!.toString : undefined))!() : undefined)`),
	)
}

func TestMapperPrefersExactRangeMatch(t *testing.T) {
	mapper := NewMapper(nil, []RangeMapping{
		{
			SourceOffset:  0,
			SourceLength:  5,
			ServiceOffset: 100,
			ServiceLength: 5,
		},
		{
			SourceOffset:  0,
			SourceLength:  10,
			ServiceOffset: 200,
			ServiceLength: 10,
		},
	})

	ranges := mapper.ToServiceRange(0, 5, false)
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	assertRangeOffsets(t, ranges[0], 100, 105)
}

func TestMapperPrefersExactServiceRangeMatch(t *testing.T) {
	mapper := NewMapper(nil, []RangeMapping{
		{
			SourceOffset:  0,
			SourceLength:  10,
			ServiceOffset: 100,
			ServiceLength: 10,
		},
		{
			SourceOffset:  5,
			SourceLength:  5,
			ServiceOffset: 100,
			ServiceLength: 5,
		},
	})

	ranges := mapper.ToSourceRange(100, 105, false)
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	assertRangeOffsets(t, ranges[0], 5, 10)
}

func TestMapperPrefersExactRangeMappingOverMapping(t *testing.T) {
	mapper := &Mapper{
		Mappings: []Mapping{
			{
				SourceOffset:  0,
				ServiceOffset: 100,
				Length:        10,
			},
		},
		RangeMappings: []RangeMapping{
			{
				SourceOffset:  0,
				SourceLength:  5,
				ServiceOffset: 200,
				ServiceLength: 5,
			},
		},
	}

	ranges := mapper.ToServiceRange(0, 5, false)
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	assertRangeOffsets(t, ranges[0], 200, 205)
}

func TestMapperFallbackOverlapRange(t *testing.T) {
	mapper := NewMapper([]Mapping{
		{
			SourceOffset:  10,
			ServiceOffset: 100,
			Length:        10,
		},
	}, nil)

	ranges := mapper.ToSourceRange(90, 120, true)
	if len(ranges) != 1 {
		t.Fatalf("expected 1 range, got %d", len(ranges))
	}
	assertRangeOffsets(t, ranges[0], 10, 20)
}

func TestMapperNoOverlapRange(t *testing.T) {
	mapper := NewMapper([]Mapping{
		{
			SourceOffset:  10,
			ServiceOffset: 100,
			Length:        10,
		},
		{
			SourceOffset:  20,
			ServiceOffset: 120,
			Length:        10,
		},
	}, nil)

	ranges := mapper.ToSourceRange(112, 115, true)
	if len(ranges) != 0 {
		t.Fatalf("expected no ranges, got %d", len(ranges))
	}
}

func TestMapperEmptyMappings(t *testing.T) {
	mapper := NewMapper(nil, nil)

	if got := mapper.ToSourceLocation(5); got != nil {
		t.Fatalf("expected nil locations, got %v", got)
	}

	ranges := mapper.ToSourceRange(1, 2, true)
	if len(ranges) != 0 {
		t.Fatalf("expected no ranges, got %d", len(ranges))
	}
}

func TestMapperDedupesLocations(t *testing.T) {
	mapper := NewMapper([]Mapping{
		{
			SourceOffset:  10,
			ServiceOffset: 100,
			Length:        10,
		},
	}, nil)

	locations := mapper.ToServiceLocation(15)
	if len(locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(locations))
	}
	if locations[0].Offset != 105 {
		t.Fatalf("expected offset 105, got %d", locations[0].Offset)
	}
}

func TestMapperFallbackSkipsReversedRange(t *testing.T) {
	mapper := NewMapper([]Mapping{
		{
			SourceOffset:  10,
			ServiceOffset: 200,
			Length:        5,
		},
		{
			SourceOffset:  30,
			ServiceOffset: 100,
			Length:        5,
		},
	}, nil)

	ranges := mapper.ToServiceRange(12, 31, true)
	assertRangeSet(t, ranges, [][2]int{
		{202, 205},
		{100, 101},
	})
}

func assertBinarySearch(t *testing.T, values []int, search int, wantLow int, wantHigh int, wantMatch *int) {
	t.Helper()

	low, high, match, ok := binarySearch(values, search)
	if low != wantLow || high != wantHigh {
		t.Fatalf("expected low=%d high=%d, got low=%d high=%d", wantLow, wantHigh, low, high)
	}

	if wantMatch == nil {
		if ok {
			t.Fatalf("expected no match, got match index %d", match)
		}
		return
	}

	if !ok || match != *wantMatch {
		t.Fatalf("expected match index %d, got ok=%v match=%d", *wantMatch, ok, match)
	}
}

func assertTranslateOffset(
	t *testing.T,
	start int,
	fromOffset int,
	toOffset int,
	fromLength int,
	toLength *int,
	want int,
	wantOk bool,
) {
	t.Helper()

	var (
		got int
		ok  bool
	)
	if toLength == nil {
		got, ok = translateOffset(start, fromOffset, toOffset, fromLength)
	} else {
		got, ok = translateOffset(start, fromOffset, toOffset, fromLength, *toLength)
	}

	if ok != wantOk || (ok && got != want) {
		t.Fatalf("expected ok=%v offset=%d, got ok=%v offset=%d", wantOk, want, ok, got)
	}
}

func assertRangeOffsets(t *testing.T, got MappedRange, wantStart int, wantEnd int) {
	t.Helper()

	if got.MappedStart != wantStart || got.MappedEnd != wantEnd {
		t.Fatalf("expected range [%d, %d], got [%d, %d]", wantStart, wantEnd, got.MappedStart, got.MappedEnd)
	}
}

func assertRangeSet(t *testing.T, got []MappedRange, want [][2]int) {
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

func idx(value string) int {
	return strings.Index(value, "|")
}
