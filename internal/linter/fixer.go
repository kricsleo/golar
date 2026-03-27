package linter

import (
	"slices"
	"strings"

	"github.com/auvred/golar/internal/linter/rule"
)

type WithFixes interface {
	GetFixes() []rule.Fix
}

func ApplyRuleFixes[M WithFixes](code string, diagnostics []M) (string, []M, bool) {
	unapplied := []M{}
	withFixes := []M{}

	fixed := false

	for _, diagnostic := range diagnostics {
		if len(diagnostic.GetFixes()) > 0 {
			fixes := diagnostic.GetFixes()
			slices.SortFunc(fixes, func(a rule.Fix, b rule.Fix) int {
				start := a.Range.Pos() - b.Range.Pos()
				if start == 0 {
					return a.Range.End() - b.Range.End()
				}
				return start
			})
			withFixes = append(withFixes, diagnostic)
		} else {
			unapplied = append(unapplied, diagnostic)
		}
	}

	slices.SortFunc(withFixes, func(a M, b M) int {
		aFixes, bFixes := a.GetFixes(), b.GetFixes()

		start := aFixes[0].Range.Pos() - bFixes[0].Range.Pos()
		if start == 0 {
			return aFixes[len(aFixes)-1].Range.End() - bFixes[len(bFixes)-1].Range.End()
		}
		return start
	})

	var builder strings.Builder

	lastFixEnd := 0
	for _, diagnostic := range withFixes {
		fixes := diagnostic.GetFixes()
		if lastFixEnd > fixes[0].Range.Pos() {
			unapplied = append(unapplied, diagnostic)
			continue
		}

		for _, fix := range fixes {
			fixed = true

			builder.WriteString(code[lastFixEnd:fix.Range.Pos()])
			builder.WriteString(fix.Text)

			lastFixEnd = fix.Range.End()
		}
	}

	builder.WriteString(code[lastFixEnd:])

	return builder.String(), unapplied, fixed
}
