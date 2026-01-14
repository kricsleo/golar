package mapping

import "github.com/microsoft/typescript-go/shim/core"

type IgnoreDirectiveMapping struct {
	ServiceOffset int
	ServiceLength int
}

type ExpectErrorDirectiveMapping struct {
	SourceOffset  int
	ServiceOffset int
	SourceLength  int
	ServiceLength int
}

type ExpectErrorDirectiveMappingWithUsed struct {
	ExpectErrorDirectiveMapping
	Used bool
}

type DirectiveMap struct {
	IgnoreMappings      []IgnoreDirectiveMapping
	ExpectErrorMappings []ExpectErrorDirectiveMappingWithUsed
	Used                int
}

func NewDirectiveMap(ignore []IgnoreDirectiveMapping, expectError []ExpectErrorDirectiveMapping) DirectiveMap {
	e := make([]ExpectErrorDirectiveMappingWithUsed, len(expectError))
	for i, d := range expectError {
		e[i] = ExpectErrorDirectiveMappingWithUsed{
			d,
			false,
		}
	}

	return DirectiveMap{
		IgnoreMappings:      ignore,
		ExpectErrorMappings: e,
	}
}

func (d *DirectiveMap) IsServiceRangeIgnored(serviceRange core.TextRange) bool {
	result := false
	for _, mapping := range d.IgnoreMappings {
		mappingRange := core.NewTextRange(
			mapping.ServiceOffset,
			mapping.ServiceOffset+mapping.ServiceLength,
		)
		if serviceRange.ContainedBy(mappingRange) {
			result = true
			break
		}
	}

	for i, mapping := range d.ExpectErrorMappings {
		mappingRange := core.NewTextRange(
			mapping.ServiceOffset,
			mapping.ServiceOffset+mapping.ServiceLength,
		)
		if serviceRange.ContainedBy(mappingRange) {
			result = true
			d.ExpectErrorMappings[i].Used = true
			d.Used++
		}
	}

	return result
}

func (d *DirectiveMap) CollectUnused() []ExpectErrorDirectiveMapping {
	if d.Used == len(d.ExpectErrorMappings) {
		return nil
	}
	res := make([]ExpectErrorDirectiveMapping, 0, len(d.ExpectErrorMappings)-d.Used)
	for _, e := range d.ExpectErrorMappings {
		if !e.Used {
			res = append(res, e.ExpectErrorDirectiveMapping)
		}
	}

	return res
}
