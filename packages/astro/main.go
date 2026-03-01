package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/auvred/golar/plugin"

	"github.com/withastro/compiler/pkg"
	"github.com/withastro/compiler/pkg/handler"
	"github.com/withastro/compiler/pkg/loc"
	"github.com/withastro/compiler/pkg/printer"
	"github.com/withastro/compiler/pkg/transform"
)

func getAstroInstallation(cwd, configFileName string) (string, string, error) {
	dir := configFileName
	if dir == "" {
		dir = cwd
	}

	var packageJson struct{ Version string }
	var astroDir string
	for {
		content, err := os.ReadFile(filepath.Join(dir, "node_modules", "astro", "package.json"))
		if err == nil {
			if err := json.Unmarshal(content, &packageJson); err != nil {
				panic(err)
			}
			astroDir = filepath.Join(dir, "node_modules", "astro")
			break
		}
		parentDir := filepath.Dir(dir)
		if dir == parentDir {
			return "", "", errors.New("Cannot find 'astro' package.")
		}
		dir = parentDir
	}

	return astroDir, packageJson.Version, nil
}

func main() {
	type projectKey struct {
		cwd            string
		configFileName string
	}

	plugin.Run(plugin.PluginOptions{
		Input:  os.Stdin,
		Output: os.Stdout,
		Extensions: []plugin.Extension{
			{
				Extension:                 ".astro",
				AllowExtensionlessImports: false,
			},
		},
		Setup: func() plugin.PluginInstance {
			astroDirsByProject := map[projectKey]string{}

			return plugin.PluginInstance{
				CreateServiceCode: func(cwd, configFileName, fileName string, sourceText string) *plugin.ServiceCode {
					project := projectKey{cwd, configFileName}
					astroDir, ok := astroDirsByProject[project]
					if !ok {
						var err error
						astroDir, _, err = getAstroInstallation(cwd, configFileName)
						if err != nil {
							return &plugin.ServiceCode{
								Errors: []plugin.ServiceCodeError{
									{
										Message: err.Error(),
										Start:   0,
										End:     0,
									},
								},
							}
						}
						astroDirsByProject[project] = astroDir
					}
					astroFooter := fmt.Appendf(nil, "\n\nimport %q\n import %q\n", filepath.Join(astroDir, "env.d.ts"), filepath.Join(astroDir, "env.d.ts"))
					transformOptions := transform.TransformOptions{
						Scope:              "xxxxxx",
						Filename:           fileName,
						NormalizedFilename: fileName,
					}
					h := handler.NewHandler(sourceText, transformOptions.Filename)

					var doc *astro.Node
					doc, err := astro.ParseWithOptions(strings.NewReader(sourceText), astro.ParseOptionWithHandler(h), astro.ParseOptionEnableLiteral(true))
					if err != nil {
						h.AppendError(err)
					}

					tsxOptions := printer.TSXOptions{}

					printed := printer.PrintToTSX(sourceText, doc, tsxOptions, transformOptions, h)

					errs := h.ErrorsRaw()
					if len(errs) > 0 {
						result := &plugin.ServiceCode{}
						for _, err := range errs {
							var rangedErr *loc.ErrorWithRange
							if errors.As(err, &rangedErr) {
								result.Errors = append(result.Errors, plugin.ServiceCodeError{
									Message: rangedErr.Text,
									Start:   rangedErr.Range.Loc.Start,
									End:     rangedErr.Range.End(),
								})
							}
						}
						if len(result.Errors) > 0 {
							return result
						}
					}

					printed.Output = append(printed.Output, astroFooter...)

					result := &plugin.ServiceCode{
						ScriptKind:  plugin.ScriptKindTSX,
						ServiceText: printed.Output,
						Mappings:    plugin.SourceMapToMappings(sourceText, string(printed.Output), string(printed.SourceMapChunk.Buffer)),
					}

					// .astro files located in node_modules sometimes import virtual: files
					// and we obviously don't have declaration files for them, so they're
					// always errored
					//
					// astro-check doesn't collect diagnostics from files in node_modules,
					// so this is a little hack to comply with this behavior
					if strings.Contains(fileName, "/node_modules/") {
						result.DeclarationFile = true
					}

					return result
				},
			}
		},
	})
}
