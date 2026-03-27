module github.com/auvred/golar

go 1.26

replace github.com/microsoft/typescript-go => ./thirdparty/typescript-go

require (
	github.com/microsoft/typescript-go v0.0.0
	github.com/zeebo/xxh3 v1.1.0
	golang.org/x/sys v0.42.0
	golang.org/x/term v0.41.0
	golang.org/x/tools v0.43.0
	gotest.tools/v3 v3.5.2
)

require (
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/go-json-experiment/json v0.0.0-20260214004413-d219187c3433 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)
