package cmd

import (
	"fmt"
	"io"
	"path"
	"sort"

	"github.com/spf13/cobra"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
	scriptpkg "github.com/yaoapp/yao/script"
)

var scriptMatchPatterns []string
var scriptErrorOnly bool

var loadRegisteredScripts = func() error {
	Boot()
	return engine.Load(config.Conf, engine.LoadOption{Action: "scripts"})
}

var loadRegisteredScriptErrors = func() error {
	Boot()
	return engine.Load(config.Conf, engine.LoadOption{Action: "scripts.error"})
}

var getRegisteredScriptErrors = func() *scriptpkg.LoadErrors {
	return scriptpkg.LastLoadErrors()
}

var scriptsCmd = &cobra.Command{
	Use:   "scripts",
	Short: L("Show registered scripts"),
	Long:  L("Show registered scripts"),
	RunE: func(cmd *cobra.Command, args []string) error {
		if scriptErrorOnly {
			return printRegisteredScriptErrors(cmd.OutOrStdout())
		}
		return printRegisteredScripts(cmd.OutOrStdout())
	},
}

func printRegisteredScripts(w io.Writer) error {
	err := loadRegisteredScripts()
	if err != nil {
		return err
	}

	names, err := filterScriptNames(registeredScriptNames(), scriptMatchPatterns)
	if err != nil {
		return err
	}

	if len(scriptMatchPatterns) == 0 {
		_, err := fmt.Fprintln(w, len(names))
		return err
	}

	for _, name := range names {
		_, err := fmt.Fprintln(w, name)
		if err != nil {
			return err
		}
	}

	return nil
}

func printRegisteredScriptErrors(w io.Writer) error {
	err := loadRegisteredScriptErrors()
	loadErrs := getRegisteredScriptErrors()
	if loadErrs == nil || len(loadErrs.Items) == 0 {
		return err
	}

	for _, item := range loadErrs.Items {
		_, writeErr := fmt.Fprintln(w, item.Error())
		if writeErr != nil {
			return writeErr
		}
	}

	return loadErrs
}

func registeredScriptNames() []string {
	names := make([]string, 0, len(v8.Scripts))
	for name := range v8.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func filterScriptNames(names []string, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		filtered := append([]string(nil), names...)
		sort.Strings(filtered)
		return filtered, nil
	}

	filtered := make([]string, 0, len(names))
	for _, name := range names {
		matched, err := matchesAnyPattern(name, patterns)
		if err != nil {
			return nil, err
		}
		if matched {
			filtered = append(filtered, name)
		}
	}

	sort.Strings(filtered)
	return filtered, nil
}

func matchesAnyPattern(name string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := path.Match(pattern, name)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func init() {
	scriptsCmd.Flags().StringArrayVarP(&scriptMatchPatterns, "match", "m", nil, L("Match registered scripts"))
	scriptsCmd.Flags().BoolVarP(&scriptErrorOnly, "error", "e", false, L("Show registered script errors"))
}
