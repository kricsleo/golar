`internal/linter/create_programs.go` mirrors project-loading behavior from upstream `thirdparty/typescript-go`.

When updating `thirdparty/typescript-go`, check for any relevant behavior changes in these upstream files:

- `thirdparty/typescript-go/pkg/project/configfileregistrybuilder.go`
- `thirdparty/typescript-go/pkg/project/projectcollectionbuilder.go`
- `thirdparty/typescript-go/pkg/project/projectcollection.go`
- `thirdparty/typescript-go/pkg/project/project.go`
- `thirdparty/typescript-go/pkg/core/projectreference.go`

If behavior changed in any of them, review and optionally adjust:

- `internal/linter/create_programs.go`
- `internal/linter/create_programs_test.go`

TODO: mention other things
