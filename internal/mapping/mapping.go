package mapping

import (
	"sort"

	"github.com/auvred/golar/internal/collections"
)

type CodeRangeKey int

const (
	SourceOffsets CodeRangeKey = iota
	ServiceOffsets
)

type Mapping struct {
	SourceOffsets  []int
	ServiceOffsets []int
	Lengths        []int
	ServiceLengths []int
}

type MappingMemo struct {
	offsets  []int
	mappings []*collections.Set[*Mapping]
}

type SourceMap struct {
	Mappings               []Mapping
	sourceCodeOffsetsMemo  *MappingMemo
	serviceCodeOffsetsMemo *MappingMemo
}

type MappedLocation struct {
	Offset  int
	Mapping *Mapping
}

type MappedRange struct {
	MappedStart  int
	MappedEnd    int
	StartMapping *Mapping
	EndMapping   *Mapping
}

func NewSourceMap(mappings []Mapping) *SourceMap {
	return &SourceMap{Mappings: mappings}
}

func (m *SourceMap) ToSourceRange(
	serviceStart int,
	serviceEnd int,
	fallbackToAnyMatch bool,
) []MappedRange {
	return m.findMatchingStartEnd(serviceStart, serviceEnd, fallbackToAnyMatch, ServiceOffsets)
}

func (m *SourceMap) ToServiceRange(
	sourceStart int,
	sourceEnd int,
	fallbackToAnyMatch bool,
) []MappedRange {
	return m.findMatchingStartEnd(sourceStart, sourceEnd, fallbackToAnyMatch, SourceOffsets)
}

func (m *SourceMap) ToSourceLocation(serviceOffset int) []MappedLocation {
	return m.findMatchingOffsets(serviceOffset, ServiceOffsets)
}

func (m *SourceMap) ToServiceLocation(sourceOffset int) []MappedLocation {
	return m.findMatchingOffsets(sourceOffset, SourceOffsets)
}

func (m *SourceMap) findMatchingOffsets(
	offset int,
	fromRange CodeRangeKey,
) []MappedLocation {
	memo := m.getMemoBasedOnRange(fromRange)
	if len(memo.offsets) == 0 {
		return nil
	}

	start, end, _ := BinarySearch(memo.offsets, offset)
	skip := collections.NewSetWithSizeHint[*Mapping](len(memo.mappings))
	toRange := otherRangeKey(fromRange)
	results := make([]MappedLocation, 0)

	for i := start; i <= end; i++ {
		for mapping := range memo.mappings[i].Keys() {
			if !skip.AddIfAbsent(mapping) {
				continue
			}
			mapped, ok := TranslateOffset(
				offset,
				getOffsets(mapping, fromRange),
				getOffsets(mapping, toRange),
				getLengths(mapping, fromRange),
				getLengths(mapping, toRange),
			)
			if ok {
				results = append(results, MappedLocation{
					Offset:  mapped,
					Mapping: mapping,
				})
			}
		}
	}

	return results
}

func (m *SourceMap) findMatchingStartEnd(
	start int,
	end int,
	fallbackToAnyMatch bool,
	fromRange CodeRangeKey,
) []MappedRange {
	toRange := otherRangeKey(fromRange)
	mappedStarts := make([]MappedLocation, 0)
	results := make([]MappedRange, 0)
	hadMatch := false

	for _, mappedStart := range m.findMatchingOffsets(start, fromRange) {
		mappedStarts = append(mappedStarts, mappedStart)
		mapping := mappedStart.Mapping
		mappedEnd, ok := TranslateOffset(
			end,
			getOffsets(mapping, fromRange),
			getOffsets(mapping, toRange),
			getLengths(mapping, fromRange),
			getLengths(mapping, toRange),
		)
		if ok {
			hadMatch = true
			results = append(results, MappedRange{
				MappedStart:  mappedStart.Offset,
				MappedEnd:    mappedEnd,
				StartMapping: mapping,
				EndMapping:   mapping,
			})
		}
	}

	if !hadMatch && fallbackToAnyMatch {
		endMatches := m.findMatchingOffsets(end, fromRange)
		for _, mappedStart := range mappedStarts {
			for _, mappedEnd := range endMatches {
				if mappedEnd.Offset < mappedStart.Offset {
					continue
				}
				results = append(results, MappedRange{
					MappedStart:  mappedStart.Offset,
					MappedEnd:    mappedEnd.Offset,
					StartMapping: mappedStart.Mapping,
					EndMapping:   mappedEnd.Mapping,
				})
				break
			}
		}
	}

	return results
}

func (m *SourceMap) getMemoBasedOnRange(fromRange CodeRangeKey) *MappingMemo {
	if fromRange == SourceOffsets {
		if m.sourceCodeOffsetsMemo == nil {
			memo := m.createMemo(SourceOffsets)
			m.sourceCodeOffsetsMemo = &memo
		}
		return m.sourceCodeOffsetsMemo
	}
	if m.serviceCodeOffsetsMemo == nil {
		memo := m.createMemo(ServiceOffsets)
		m.serviceCodeOffsetsMemo = &memo
	}
	return m.serviceCodeOffsetsMemo
}

func (m *SourceMap) createMemo(key CodeRangeKey) MappingMemo {
	offsetsSet := collections.NewSetWithSizeHint[int](0)
	for _, mapping := range m.Mappings {
		offsets := getOffsets(&mapping, key)
		lengths := getLengths(&mapping, key)
		count := min(len(offsets), len(lengths))
		for i := range count {
			offsetsSet.Add(offsets[i])
			offsetsSet.Add(offsets[i] + lengths[i])
		}
	}

	offsets := make([]int, 0, offsetsSet.Len())
	for offset := range offsetsSet.Keys() {
		offsets = append(offsets, offset)
	}
	sort.Ints(offsets)

	mappings := make([]*collections.Set[*Mapping], len(offsets))
	for i := range mappings {
		mappings[i] = collections.NewSetWithSizeHint[*Mapping](0)
	}

	for _, mapping := range m.Mappings {
		offsetsList := getOffsets(&mapping, key)
		lengths := getLengths(&mapping, key)
		count := min(len(offsetsList), len(lengths))
		for i := range count {
			startOffset := offsetsList[i]
			endOffset := startOffset + lengths[i]
			_, _, startMatch := BinarySearch(offsets, startOffset)
			_, _, endMatch := BinarySearch(offsets, endOffset)
			for j := startMatch; j <= endMatch; j++ {
				mappings[j].Add(&mapping)
			}
		}
	}

	return MappingMemo{offsets: offsets, mappings: mappings}
}

func otherRangeKey(key CodeRangeKey) CodeRangeKey {
	if key == SourceOffsets {
		return ServiceOffsets
	}
	return SourceOffsets
}

func getOffsets(mapping *Mapping, key CodeRangeKey) []int {
	if key == SourceOffsets {
		return mapping.SourceOffsets
	}
	return mapping.ServiceOffsets
}

func getLengths(mapping *Mapping, key CodeRangeKey) []int {
	if key == SourceOffsets {
		return mapping.Lengths
	}
	if len(mapping.ServiceLengths) > 0 {
		return mapping.ServiceLengths
	}
	return mapping.Lengths
}

func TranslateOffset(
	start int,
	fromOffsets []int,
	toOffsets []int,
	fromLengths []int,
	toLengths []int,
) (int, bool) {
	if len(toLengths) == 0 {
		toLengths = fromLengths
	}

	count := min(len(fromOffsets), len(toOffsets), len(fromLengths), len(toLengths))
	if count == 0 {
		return 0, false
	}

	offsets := fromOffsets
	if len(offsets) > count {
		offsets = offsets[:count]
	}

	low := 0
	high := len(offsets) - 1

	for low <= high {
		mid := (low + high) / 2
		fromOffset := offsets[mid]
		fromLength := fromLengths[mid]
		if start >= fromOffset && start <= fromOffset+fromLength {
			toLength := toLengths[mid]
			toOffset := toOffsets[mid]
			rangeOffset := min(start-fromOffset, toLength)
			return toOffset + rangeOffset, true
		}
		if start < fromOffset {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}

	return 0, false
}

func BinarySearch(values []int, searchValue int) (low int, high int, match int) {
	if len(values) == 0 {
		return 0, -1, 0
	}

	low = 0
	high = len(values) - 1

	for low <= high {
		mid := (low + high) / 2
		midValue := values[mid]
		if midValue < searchValue {
			low = mid + 1
		} else if midValue > searchValue {
			high = mid - 1
		} else {
			low = mid
			high = mid
			match = mid
			break
		}
	}

	finalLow := max(min(low, high, len(values)-1), 0)
	finalHigh := min(max(low, high, 0), len(values)-1)

	return finalLow, finalHigh, match
}
