package mapping

import "sort"

// TODO: revise and simplify

type Mapping struct {
	SourceOffset  int
	ServiceOffset int
	Length        int
}

type RangeMapping struct {
	SourceOffset  int
	SourceLength  int
	ServiceOffset int
	ServiceLength int
}

type MappedLocation struct {
	Offset  int
	Mapping Mapping
}

type MappedRange struct {
	MappedStart  int
	MappedEnd    int
	StartMapping RangeMapping
	EndMapping   RangeMapping
}

type Mapper struct {
	Mappings                []Mapping
	RangeMappings           []RangeMapping
	sourceOffsetsMemo       *mappingMemo
	serviceOffsetsMemo      *mappingMemo
	sourceRangeOffsetsMemo  *mappingMemo
	serviceRangeOffsetsMemo *mappingMemo
}

func NewMapper(mappings []Mapping, rangeMappings []RangeMapping) *Mapper {
	return &Mapper{
		Mappings:      mappings,
		RangeMappings: rangeMappings,
	}
}

func (m *Mapper) ToSourceRange(serviceStart, serviceEnd int, fallbackToAnyMatch bool) []MappedRange {
	return m.findMatchingStartEnd(serviceStart, serviceEnd, fallbackToAnyMatch, rangeService)
}

func (m *Mapper) ToServiceRange(sourceStart, sourceEnd int, fallbackToAnyMatch bool) []MappedRange {
	return m.findMatchingStartEnd(sourceStart, sourceEnd, fallbackToAnyMatch, rangeSource)
}

func (m *Mapper) ToSourceLocation(serviceOffset int) []MappedLocation {
	return m.findMatchingOffsets(serviceOffset, rangeService)
}

func (m *Mapper) ToServiceLocation(sourceOffset int) []MappedLocation {
	return m.findMatchingOffsets(sourceOffset, rangeSource)
}

type rangeKey int

const (
	rangeSource rangeKey = iota
	rangeService
)

type mappingMemo struct {
	offsets  []int
	mappings [][]int
}

func (m *Mapper) findMatchingOffsets(offset int, fromRange rangeKey) []MappedLocation {
	memo := m.getMemoBasedOnRange(fromRange)
	if len(memo.offsets) == 0 {
		return nil
	}

	start, end, _, _ := binarySearch(memo.offsets, offset)
	toRange := otherRange(fromRange)
	seen := make(map[int]struct{})
	var results []MappedLocation

	for i := start; i <= end; i++ {
		for _, mappingIndex := range memo.mappings[i] {
			if _, ok := seen[mappingIndex]; ok {
				continue
			}
			seen[mappingIndex] = struct{}{}

			mapping := m.Mappings[mappingIndex]
			fromOffset, fromLength := mappingOffsetLength(mapping, fromRange)
			toOffset, toLength := mappingOffsetLength(mapping, toRange)
			mapped, ok := translateOffset(offset, fromOffset, toOffset, fromLength, toLength)
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

func (m *Mapper) findMatchingStartEnd(
	start int,
	end int,
	fallbackToAnyMatch bool,
	fromRange rangeKey,
) []MappedRange {
	if exactMatches := m.findExactRangeMatches(start, end, fromRange); len(exactMatches) > 0 {
		return exactMatches
	}

	toRange := otherRange(fromRange)
	var mappedStarts []mappedRangeLocation
	var results []MappedRange
	hadMatch := false

	for _, mappedStart := range m.findMatchingRangeOffsets(start, fromRange) {
		mappedStarts = append(mappedStarts, mappedStart)
		mapping := mappedStart.Mapping
		fromOffset, fromLength := rangeOffsetLength(mapping, fromRange)
		toOffset, toLength := rangeOffsetLength(mapping, toRange)
		mappedEnd, ok := translateOffset(end, fromOffset, toOffset, fromLength, toLength)
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
		if len(mappedStarts) > 0 {
			endMatches := m.findMatchingRangeOffsets(end, fromRange)
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
	}

	if fallbackToAnyMatch && len(results) == 0 {
		results = append(results, m.findOverlappingRanges(start, end, fromRange)...)
	}

	return results
}

func (m *Mapper) getMemoBasedOnRange(fromRange rangeKey) *mappingMemo {
	if fromRange == rangeSource {
		if m.sourceOffsetsMemo == nil {
			memo := m.createMemo(rangeSource)
			m.sourceOffsetsMemo = &memo
		}
		return m.sourceOffsetsMemo
	}
	if m.serviceOffsetsMemo == nil {
		memo := m.createMemo(rangeService)
		m.serviceOffsetsMemo = &memo
	}
	return m.serviceOffsetsMemo
}

func (m *Mapper) getRangeMemoBasedOnRange(fromRange rangeKey) *mappingMemo {
	if fromRange == rangeSource {
		if m.sourceRangeOffsetsMemo == nil {
			memo := m.createRangeMemo(rangeSource)
			m.sourceRangeOffsetsMemo = &memo
		}
		return m.sourceRangeOffsetsMemo
	}
	if m.serviceRangeOffsetsMemo == nil {
		memo := m.createRangeMemo(rangeService)
		m.serviceRangeOffsetsMemo = &memo
	}
	return m.serviceRangeOffsetsMemo
}

func (m *Mapper) createMemo(key rangeKey) mappingMemo {
	offsetsSet := make(map[int]struct{})
	for _, mapping := range m.Mappings {
		offset, length := mappingOffsetLength(mapping, key)
		offsetsSet[offset] = struct{}{}
		offsetsSet[offset+length] = struct{}{}
	}

	offsets := make([]int, 0, len(offsetsSet))
	for offset := range offsetsSet {
		offsets = append(offsets, offset)
	}
	sort.Ints(offsets)

	mappings := make([][]int, len(offsets))

	for mappingIndex, mapping := range m.Mappings {
		startOffset, length := mappingOffsetLength(mapping, key)
		endOffset := startOffset + length

		startIndex, _, startMatch, startOk := binarySearch(offsets, startOffset)
		endIndex, _, endMatch, endOk := binarySearch(offsets, endOffset)
		if startOk {
			startIndex = startMatch
		}
		if endOk {
			endIndex = endMatch
		}
		if !startOk || !endOk {
			continue
		}

		for i := startIndex; i <= endIndex; i++ {
			mappings[i] = append(mappings[i], mappingIndex)
		}
	}

	return mappingMemo{offsets: offsets, mappings: mappings}
}

func (m *Mapper) createRangeMemo(key rangeKey) mappingMemo {
	offsetsSet := make(map[int]struct{})
	for _, mapping := range m.RangeMappings {
		offset, length := rangeOffsetLength(mapping, key)
		offsetsSet[offset] = struct{}{}
		offsetsSet[offset+length] = struct{}{}
	}

	offsets := make([]int, 0, len(offsetsSet))
	for offset := range offsetsSet {
		offsets = append(offsets, offset)
	}
	sort.Ints(offsets)

	mappings := make([][]int, len(offsets))

	for mappingIndex, mapping := range m.RangeMappings {
		startOffset, length := rangeOffsetLength(mapping, key)
		endOffset := startOffset + length

		startIndex, _, startMatch, startOk := binarySearch(offsets, startOffset)
		endIndex, _, endMatch, endOk := binarySearch(offsets, endOffset)
		if startOk {
			startIndex = startMatch
		}
		if endOk {
			endIndex = endMatch
		}
		if !startOk || !endOk {
			continue
		}

		for i := startIndex; i <= endIndex; i++ {
			mappings[i] = append(mappings[i], mappingIndex)
		}
	}

	return mappingMemo{offsets: offsets, mappings: mappings}
}

func (m *Mapper) findOverlappingRanges(start int, end int, fromRange rangeKey) []MappedRange {
	results := m.findOverlappingRangeMappings(start, end, fromRange)
	results = append(results, m.findOverlappingMappings(start, end, fromRange)...)
	return results
}

func otherRange(key rangeKey) rangeKey {
	if key == rangeSource {
		return rangeService
	}
	return rangeSource
}

func mappingOffsetLength(mapping Mapping, key rangeKey) (int, int) {
	if key == rangeSource {
		return mapping.SourceOffset, mapping.Length
	}
	return mapping.ServiceOffset, mapping.Length
}

func rangeOffsetLength(mapping RangeMapping, key rangeKey) (int, int) {
	if key == rangeSource {
		return mapping.SourceOffset, mapping.SourceLength
	}
	return mapping.ServiceOffset, mapping.ServiceLength
}

func rangeMappingFromMapping(mapping Mapping) RangeMapping {
	return RangeMapping{
		SourceOffset:  mapping.SourceOffset,
		SourceLength:  mapping.Length,
		ServiceOffset: mapping.ServiceOffset,
		ServiceLength: mapping.Length,
	}
}

type mappedRangeLocation struct {
	Offset  int
	Mapping RangeMapping
}

func (m *Mapper) findExactRangeMatches(start int, end int, fromRange rangeKey) []MappedRange {
	if results := m.findExactRangeMatchesFromRangeMappings(start, end, fromRange); len(results) > 0 {
		return results
	}
	return m.findExactRangeMatchesFromMappings(start, end, fromRange)
}

func (m *Mapper) findMatchingRangeOffsets(offset int, fromRange rangeKey) []mappedRangeLocation {
	results := m.findMatchingRangeOffsetsFromRangeMappings(offset, fromRange)
	return append(results, m.findMatchingRangeOffsetsFromMappings(offset, fromRange)...)
}

func (m *Mapper) findExactRangeMatchesFromRangeMappings(start int, end int, fromRange rangeKey) []MappedRange {
	if len(m.RangeMappings) == 0 {
		return nil
	}

	toRange := otherRange(fromRange)
	var results []MappedRange

	for _, mapping := range m.RangeMappings {
		fromOffset, fromLength := rangeOffsetLength(mapping, fromRange)
		if start != fromOffset || end != fromOffset+fromLength {
			continue
		}

		toOffset, toLength := rangeOffsetLength(mapping, toRange)
		results = append(results, MappedRange{
			MappedStart:  toOffset,
			MappedEnd:    toOffset + toLength,
			StartMapping: mapping,
			EndMapping:   mapping,
		})
	}

	return results
}

func (m *Mapper) findExactRangeMatchesFromMappings(start int, end int, fromRange rangeKey) []MappedRange {
	if len(m.Mappings) == 0 {
		return nil
	}

	toRange := otherRange(fromRange)
	var results []MappedRange

	for _, mapping := range m.Mappings {
		fromOffset, fromLength := mappingOffsetLength(mapping, fromRange)
		if start != fromOffset || end != fromOffset+fromLength {
			continue
		}

		toOffset, toLength := mappingOffsetLength(mapping, toRange)
		rangeMapping := rangeMappingFromMapping(mapping)
		results = append(results, MappedRange{
			MappedStart:  toOffset,
			MappedEnd:    toOffset + toLength,
			StartMapping: rangeMapping,
			EndMapping:   rangeMapping,
		})
	}

	return results
}

func (m *Mapper) findMatchingRangeOffsetsFromRangeMappings(
	offset int,
	fromRange rangeKey,
) []mappedRangeLocation {
	memo := m.getRangeMemoBasedOnRange(fromRange)
	if len(memo.offsets) == 0 {
		return nil
	}

	start, end, _, _ := binarySearch(memo.offsets, offset)
	toRange := otherRange(fromRange)
	seen := make(map[int]struct{})
	var results []mappedRangeLocation

	for i := start; i <= end; i++ {
		for _, mappingIndex := range memo.mappings[i] {
			if _, ok := seen[mappingIndex]; ok {
				continue
			}
			seen[mappingIndex] = struct{}{}

			mapping := m.RangeMappings[mappingIndex]
			fromOffset, fromLength := rangeOffsetLength(mapping, fromRange)
			toOffset, toLength := rangeOffsetLength(mapping, toRange)
			mapped, ok := translateOffset(offset, fromOffset, toOffset, fromLength, toLength)
			if ok {
				results = append(results, mappedRangeLocation{
					Offset:  mapped,
					Mapping: mapping,
				})
			}
		}
	}

	return results
}

func (m *Mapper) findMatchingRangeOffsetsFromMappings(
	offset int,
	fromRange rangeKey,
) []mappedRangeLocation {
	memo := m.getMemoBasedOnRange(fromRange)
	if len(memo.offsets) == 0 {
		return nil
	}

	start, end, _, _ := binarySearch(memo.offsets, offset)
	toRange := otherRange(fromRange)
	seen := make(map[int]struct{})
	var results []mappedRangeLocation

	for i := start; i <= end; i++ {
		for _, mappingIndex := range memo.mappings[i] {
			if _, ok := seen[mappingIndex]; ok {
				continue
			}
			seen[mappingIndex] = struct{}{}

			mapping := m.Mappings[mappingIndex]
			fromOffset, fromLength := mappingOffsetLength(mapping, fromRange)
			toOffset, toLength := mappingOffsetLength(mapping, toRange)
			mapped, ok := translateOffset(offset, fromOffset, toOffset, fromLength, toLength)
			if ok {
				results = append(results, mappedRangeLocation{
					Offset:  mapped,
					Mapping: rangeMappingFromMapping(mapping),
				})
			}
		}
	}

	return results
}

func (m *Mapper) findOverlappingRangeMappings(start int, end int, fromRange rangeKey) []MappedRange {
	memo := m.getRangeMemoBasedOnRange(fromRange)
	if len(memo.offsets) == 0 {
		return nil
	}

	startLow, startHigh, _, _ := binarySearch(memo.offsets, start)
	endLow, endHigh, _, _ := binarySearch(memo.offsets, end)
	startIndex := min(startLow, startHigh)
	endIndex := max(endLow, endHigh)
	toRange := otherRange(fromRange)
	seen := make(map[int]struct{})
	var results []MappedRange

	for i := startIndex; i <= endIndex; i++ {
		for _, mappingIndex := range memo.mappings[i] {
			if _, ok := seen[mappingIndex]; ok {
				continue
			}
			seen[mappingIndex] = struct{}{}

			mapping := m.RangeMappings[mappingIndex]
			fromStart, fromLength := rangeOffsetLength(mapping, fromRange)
			fromEnd := fromStart + fromLength
			if end < fromStart || start > fromEnd {
				continue
			}

			overlapStart := max(start, fromStart)
			overlapEnd := min(end, fromEnd)

			toOffset, toLength := rangeOffsetLength(mapping, toRange)
			mappedStart, okStart := translateOffset(overlapStart, fromStart, toOffset, fromLength, toLength)
			mappedEnd, okEnd := translateOffset(overlapEnd, fromStart, toOffset, fromLength, toLength)
			if !okStart || !okEnd {
				continue
			}

			results = append(results, MappedRange{
				MappedStart:  mappedStart,
				MappedEnd:    mappedEnd,
				StartMapping: mapping,
				EndMapping:   mapping,
			})
		}
	}

	return results
}

func (m *Mapper) findOverlappingMappings(start int, end int, fromRange rangeKey) []MappedRange {
	memo := m.getMemoBasedOnRange(fromRange)
	if len(memo.offsets) == 0 {
		return nil
	}

	startLow, startHigh, _, _ := binarySearch(memo.offsets, start)
	endLow, endHigh, _, _ := binarySearch(memo.offsets, end)
	startIndex := min(startLow, startHigh)
	endIndex := max(endLow, endHigh)
	toRange := otherRange(fromRange)
	seen := make(map[int]struct{})
	var results []MappedRange

	for i := startIndex; i <= endIndex; i++ {
		for _, mappingIndex := range memo.mappings[i] {
			if _, ok := seen[mappingIndex]; ok {
				continue
			}
			seen[mappingIndex] = struct{}{}

			mapping := m.Mappings[mappingIndex]
			fromStart, fromLength := mappingOffsetLength(mapping, fromRange)
			fromEnd := fromStart + fromLength
			if end < fromStart || start > fromEnd {
				continue
			}

			overlapStart := max(start, fromStart)
			overlapEnd := min(end, fromEnd)

			toOffset, toLength := mappingOffsetLength(mapping, toRange)
			mappedStart, okStart := translateOffset(overlapStart, fromStart, toOffset, fromLength, toLength)
			mappedEnd, okEnd := translateOffset(overlapEnd, fromStart, toOffset, fromLength, toLength)
			if !okStart || !okEnd {
				continue
			}

			rangeMapping := rangeMappingFromMapping(mapping)
			results = append(results, MappedRange{
				MappedStart:  mappedStart,
				MappedEnd:    mappedEnd,
				StartMapping: rangeMapping,
				EndMapping:   rangeMapping,
			})
		}
	}

	return results
}

func binarySearch(values []int, searchValue int) (low int, high int, match int, hasMatch bool) {
	if len(values) == 0 {
		return 0, -1, 0, false
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
			hasMatch = true
			break
		}
	}

	finalLow := max(min(min(low, high), len(values)-1), 0)
	finalHigh := min(max(max(low, high), 0), len(values)-1)

	return finalLow, finalHigh, match, hasMatch
}

func translateOffset(
	start int,
	fromOffset int,
	toOffset int,
	fromLength int,
	toLengthOptional ...int,
) (int, bool) {
	if start < fromOffset || start > fromOffset+fromLength {
		return 0, false
	}

	toLength := fromLength
	if len(toLengthOptional) > 0 {
		toLength = toLengthOptional[0]
	}

	rangeOffset := min(start-fromOffset, toLength)

	return toOffset + rangeOffset, true
}
