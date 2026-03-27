package linter

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/auvred/golar/internal/golar"
	"github.com/auvred/golar/internal/utils"
	"github.com/auvred/golar/util"
	"github.com/microsoft/typescript-go/pkg/ast"
	"github.com/microsoft/typescript-go/pkg/collections"
	"github.com/microsoft/typescript-go/pkg/compiler"
	"github.com/microsoft/typescript-go/pkg/core"
	"github.com/microsoft/typescript-go/pkg/execute/tsc"
	"github.com/microsoft/typescript-go/pkg/tsoptions"
	"github.com/microsoft/typescript-go/pkg/tspath"
	"github.com/microsoft/typescript-go/pkg/vfs"
	"github.com/zeebo/xxh3"
)

type CreateProgramsOptions struct {
	FS                 vfs.FS
	CurrentDirectory   string
	DefaultLibraryPath string
}

type ProgramSourceFile struct {
	Program    *compiler.Program
	ProgramId  uint32
	SourceFile *ast.SourceFile
}

var createProgramsDebug = util.NewDebug("create-programs")

func debugProgramName(program *compiler.Program) string {
	if program == nil {
		return "<nil>"
	}
	commandLine := program.CommandLine()
	if commandLine == nil || commandLine.ConfigFile == nil {
		return "<inferred>"
	}
	return commandLine.ConfigName()
}

func CreatePrograms(filenames []string, opts CreateProgramsOptions) (map[string]ProgramSourceFile, []*compiler.Program) {
	result := make(map[string]ProgramSourceFile, len(filenames))
	if len(filenames) == 0 {
		return nil, nil
	}

	currentDirectory := tspath.NormalizePath(opts.CurrentDirectory)

	extendedConfig := &tsc.ExtendedConfigCache{}
	host := compiler.NewCachedFSCompilerHost(currentDirectory, opts.FS, opts.DefaultLibraryPath, extendedConfig, nil)

	state := programSearchState{
		fs:                     opts.FS,
		host:                   host,
		extendedConfig:         extendedConfig,
		currentDirectory:       currentDirectory,
		useCaseSensitive:       opts.FS.UseCaseSensitiveFileNames(),
		nearestConfigByDir:     map[string]string{},
		ancestorConfigByConfig: map[string]string{},
		parsedConfigByPath:     map[tspath.Path]*tsoptions.ParsedCommandLine{},
		programByConfigPath:    map[tspath.Path]*compiler.Program{},
	}

	uniqueFilePaths := collections.NewSetWithSizeHint[tspath.Path](len(filenames))
	originalToPath := make(map[string]tspath.Path, len(filenames))
	state.fileNameByPath = make(map[tspath.Path]string, len(filenames))

	filesByStartConfig := map[string][]tspath.Path{}
	var filesWithoutConfig []tspath.Path

	for _, fileName := range filenames {
		normalizedFileName := tspath.GetNormalizedAbsolutePath(fileName, state.currentDirectory)
		path := state.toPath(normalizedFileName)
		originalToPath[fileName] = path

		if !uniqueFilePaths.AddIfAbsent(path) {
			continue
		}
		state.fileNameByPath[path] = normalizedFileName

		startConfig := state.nearestConfigForFile(normalizedFileName)
		if startConfig == "" {
			filesWithoutConfig = append(filesWithoutConfig, path)
			continue
		}
		filesByStartConfig[startConfig] = append(filesByStartConfig[startConfig], path)
	}

	assignments := make(map[tspath.Path]*compiler.Program, uniqueFilePaths.Len())
	var unresolved []tspath.Path

	startConfigs := make([]string, 0, len(filesByStartConfig))
	for startConfig := range filesByStartConfig {
		startConfigs = append(startConfigs, startConfig)
	}
	sort.Strings(startConfigs)

	for _, startConfig := range startConfigs {
		state.sortPaths(filesByStartConfig[startConfig])
		assigned, unassigned := state.assignConfiguredProgramForFiles(startConfig, filesByStartConfig[startConfig])
		maps.Copy(assignments, assigned)
		unresolved = append(unresolved, unassigned...)
	}

	unresolved = append(unresolved, filesWithoutConfig...)
	if len(unresolved) > 0 {
		state.sortPaths(unresolved)
		inferredProgram := state.inferredProgramForFiles(unresolved)
		for _, path := range unresolved {
			assignments[path] = inferredProgram
		}
	}

	orderedPrograms := make([]*compiler.Program, 0, len(filenames))
	seenPrograms := map[*compiler.Program]uint32{}

	for _, fileName := range filenames {
		path := originalToPath[fileName]
		program := assignments[path]
		if program == nil {
			panic(fmt.Sprintf("file %s -> no program", fileName))
		}

		sourceFile := program.GetSourceFileByPath(path)
		if sourceFile == nil {
			panic(fmt.Sprintf("file %s -> %s missing source file", fileName, debugProgramName(program)))
		}

		programId, ok := seenPrograms[program]
		if !ok {
			programId = uint32(len(orderedPrograms))
			seenPrograms[program] = programId
			orderedPrograms = append(orderedPrograms, program)
		}

		createProgramsDebug.Printf("file %s -> program[%d] %s", fileName, programId, debugProgramName(program))
		result[fileName] = ProgramSourceFile{
			Program:    program,
			ProgramId:  programId,
			SourceFile: sourceFile,
		}
	}

	return result, orderedPrograms
}

type programSearchState struct {
	fs               vfs.FS
	host             compiler.CompilerHost
	extendedConfig   tsoptions.ExtendedConfigCache
	currentDirectory string
	useCaseSensitive bool

	nearestConfigByDir     map[string]string
	ancestorConfigByConfig map[string]string
	parsedConfigByPath     map[tspath.Path]*tsoptions.ParsedCommandLine
	programByConfigPath    map[tspath.Path]*compiler.Program

	fileNameByPath map[tspath.Path]string

	inferredProgram *compiler.Program

	sourceFileCache collections.SyncMap[xxh3.Uint128, *ast.SourceFile]
}

func (s *programSearchState) toPath(fileName string) tspath.Path {
	return tspath.ToPath(fileName, s.currentDirectory, s.useCaseSensitive)
}

func (s *programSearchState) sortPaths(paths []tspath.Path) {
	slices.SortFunc(paths, func(a, b tspath.Path) int {
		return strings.Compare(string(a), string(b))
	})
}

func (s *programSearchState) nearestConfigForFile(fileName string) string {
	return s.nearestConfigForDirectory(tspath.GetDirectoryPath(fileName))
}

func (s *programSearchState) nearestConfigForDirectory(startDirectory string) string {
	if configName, ok := s.nearestConfigByDir[startDirectory]; ok {
		return configName
	}

	visited := make([]string, 0, 8)
	directory := startDirectory
	for {
		if configName, ok := s.nearestConfigByDir[directory]; ok {
			for _, path := range visited {
				s.nearestConfigByDir[path] = configName
			}
			return configName
		}

		tsconfigPath := tspath.CombinePaths(directory, "tsconfig.json")
		if s.fs.FileExists(tsconfigPath) {
			s.nearestConfigByDir[directory] = tsconfigPath
			for _, path := range visited {
				s.nearestConfigByDir[path] = tsconfigPath
			}
			return tsconfigPath
		}

		jsconfigPath := tspath.CombinePaths(directory, "jsconfig.json")
		if s.fs.FileExists(jsconfigPath) {
			s.nearestConfigByDir[directory] = jsconfigPath
			for _, path := range visited {
				s.nearestConfigByDir[path] = jsconfigPath
			}
			return jsconfigPath
		}

		if strings.HasSuffix(directory, "/node_modules") {
			s.nearestConfigByDir[directory] = ""
			for _, path := range visited {
				s.nearestConfigByDir[path] = ""
			}
			return ""
		}

		visited = append(visited, directory)
		parent := tspath.GetDirectoryPath(directory)
		if parent == directory {
			s.nearestConfigByDir[directory] = ""
			for _, path := range visited {
				s.nearestConfigByDir[path] = ""
			}
			return ""
		}
		directory = parent
	}
}

func (s *programSearchState) ancestorConfig(configFileName string) string {
	if ancestor, ok := s.ancestorConfigByConfig[configFileName]; ok {
		return ancestor
	}

	configDirectory := tspath.GetDirectoryPath(configFileName)
	if strings.HasSuffix(configDirectory, "/node_modules") {
		s.ancestorConfigByConfig[configFileName] = ""
		return ""
	}

	parentDirectory := tspath.GetDirectoryPath(configDirectory)
	if parentDirectory == configDirectory {
		s.ancestorConfigByConfig[configFileName] = ""
		return ""
	}

	ancestor := s.nearestConfigForDirectory(parentDirectory)
	s.ancestorConfigByConfig[configFileName] = ancestor
	return ancestor
}

func (s *programSearchState) parsedConfig(configFileName string) *tsoptions.ParsedCommandLine {
	configPath := s.toPath(configFileName)
	if parsedConfig, ok := s.parsedConfigByPath[configPath]; ok {
		return parsedConfig
	}

	parsedConfig := s.host.GetResolvedProjectReference(configFileName, configPath)
	s.parsedConfigByPath[configPath] = parsedConfig
	return parsedConfig
}

func (s *programSearchState) configuredProgram(configFileName string, parsedConfig *tsoptions.ParsedCommandLine) *compiler.Program {
	configPath := s.toPath(configFileName)
	if program, ok := s.programByConfigPath[configPath]; ok {
		return program
	}

	programConfig := parsedConfig
	if parsedConfig.CompilerOptions().DisableSourceOfProjectReferenceRedirect.IsTrue() {
		forcedConfig := parsedConfig.ReloadFileNamesOfParsedCommandLine(s.fs)
		forcedOptions := parsedConfig.CompilerOptions().Clone()
		forcedOptions.DisableSourceOfProjectReferenceRedirect = core.TSFalse
		forcedConfig.SetCompilerOptions(forcedOptions)
		programConfig = forcedConfig
	}

	program := compiler.NewProgram(compiler.ProgramOptions{
		Host:                        utils.NewIndexedCompilerHost(golar.NewCompilerHost(s.host, programConfig.ConfigName(), &s.sourceFileCache)),
		Config:                      programConfig,
		UseSourceOfProjectReference: true,
	})
	s.programByConfigPath[configPath] = program
	return program
}

func (s *programSearchState) shouldCreateProgram(parsedConfig *tsoptions.ParsedCommandLine, unresolved map[tspath.Path]struct{}) bool {
	if len(unresolved) == 0 || len(parsedConfig.FileNames()) == 0 {
		return false
	}

	if parsedConfig.CompilerOptions().Composite.IsTrue() {
		fileNamesByPath := parsedConfig.FileNamesByPath()
		for path := range unresolved {
			if _, ok := fileNamesByPath[path]; ok {
				return true
			}
		}
		return false
	}

	for path := range unresolved {
		if parsedConfig.PossiblyMatchesFileName(s.fileNameByPath[path]) {
			return true
		}
	}

	return false
}

func (s *programSearchState) assignConfiguredProgramForFiles(startConfig string, paths []tspath.Path) (map[tspath.Path]*compiler.Program, []tspath.Path) {
	assignments := make(map[tspath.Path]*compiler.Program, len(paths))
	fallback := make(map[tspath.Path]*compiler.Program)
	unresolved := make(map[tspath.Path]struct{}, len(paths))
	for _, path := range paths {
		unresolved[path] = struct{}{}
	}

	visitedConfigs := map[string]struct{}{}

	for configFileName := startConfig; configFileName != "" && len(unresolved) > 0; configFileName = s.ancestorConfig(configFileName) {
		rootParsedConfig := s.parsedConfig(configFileName)

		queue := []string{configFileName}
		for len(queue) > 0 && len(unresolved) > 0 {
			nodeConfigFileName := queue[0]
			queue = queue[1:]

			if _, seen := visitedConfigs[nodeConfigFileName]; seen {
				continue
			}
			visitedConfigs[nodeConfigFileName] = struct{}{}

			parsedConfig := s.parsedConfig(nodeConfigFileName)
			if parsedConfig == nil {
				continue
			}

			if s.shouldCreateProgram(parsedConfig, unresolved) {
				program := s.configuredProgram(nodeConfigFileName, parsedConfig)
				s.assignProgramMatches(program, unresolved, assignments, fallback)
			}

			for _, referenceConfigFileName := range parsedConfig.ResolvedProjectReferencePaths() {
				if _, seen := visitedConfigs[referenceConfigFileName]; !seen {
					queue = append(queue, referenceConfigFileName)
				}
			}
		}

		if rootParsedConfig != nil && rootParsedConfig.CompilerOptions().DisableSolutionSearching.IsTrue() {
			break
		}
	}

	if len(unresolved) > 0 {
		for path := range unresolved {
			if program, ok := fallback[path]; ok {
				assignments[path] = program
				delete(unresolved, path)
			}
		}
	}

	remaining := make([]tspath.Path, 0, len(unresolved))
	for path := range unresolved {
		remaining = append(remaining, path)
	}
	return assignments, remaining
}

func (s *programSearchState) assignProgramMatches(program *compiler.Program, unresolved map[tspath.Path]struct{}, assignments map[tspath.Path]*compiler.Program, fallback map[tspath.Path]*compiler.Program) {
	if len(unresolved) == 0 {
		return
	}

	sourceFiles := program.SourceFiles()
	if len(unresolved) <= len(sourceFiles) {
		for path := range unresolved {
			if program.GetSourceFileByPath(path) == nil {
				continue
			}
			s.assignProgramMatch(program, path, unresolved, assignments, fallback)
		}
		return
	}

	for _, sourceFile := range sourceFiles {
		path := sourceFile.Path()
		if _, ok := unresolved[path]; !ok {
			continue
		}
		s.assignProgramMatch(program, path, unresolved, assignments, fallback)
	}
}

func (s *programSearchState) assignProgramMatch(program *compiler.Program, path tspath.Path, unresolved map[tspath.Path]struct{}, assignments map[tspath.Path]*compiler.Program, fallback map[tspath.Path]*compiler.Program) {
	if program.IsSourceFromProjectReference(path) {
		if _, seen := fallback[path]; !seen {
			fallback[path] = program
		}
		return
	}

	assignments[path] = program
	delete(unresolved, path)
	delete(fallback, path)
}

func (s *programSearchState) inferredProgramForFiles(paths []tspath.Path) *compiler.Program {
	if s.inferredProgram != nil {
		return s.inferredProgram
	}

	rootFileNames := make([]string, 0, len(paths))
	for _, path := range paths {
		rootFileNames = append(rootFileNames, s.fileNameByPath[path])
	}
	sort.Strings(rootFileNames)

	compilerOptions := &core.CompilerOptions{
		AllowJs:                    core.TSTrue,
		Module:                     core.ModuleKindESNext,
		ModuleResolution:           core.ModuleResolutionKindBundler,
		Target:                     core.ScriptTargetLatestStandard,
		Jsx:                        core.JsxEmitReactJSX,
		AllowImportingTsExtensions: core.TSTrue,
		StrictNullChecks:           core.TSTrue,
		StrictFunctionTypes:        core.TSTrue,
		SourceMap:                  core.TSTrue,
		AllowNonTsExtensions:       core.TSTrue,
		ResolveJsonModule:          core.TSTrue,
		NoEmit:                     core.TSTrue,
	}

	commandLine := tsoptions.NewParsedCommandLine(
		compilerOptions,
		rootFileNames,
		tspath.ComparePathsOptions{
			UseCaseSensitiveFileNames: s.fs.UseCaseSensitiveFileNames(),
			CurrentDirectory:          s.currentDirectory,
		},
	)

	s.inferredProgram = compiler.NewProgram(compiler.ProgramOptions{
		Host:                        utils.NewIndexedCompilerHost(golar.NewCompilerHost(s.host, "/dev/null/inferred", &s.sourceFileCache)),
		Config:                      commandLine,
		UseSourceOfProjectReference: true,
	})
	return s.inferredProgram
}
