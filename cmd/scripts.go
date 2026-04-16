package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
)

var loadRegisteredScripts = func() error {
	Boot()
	return engine.Load(config.Conf, engine.LoadOption{Action: "scripts"})
}

var scriptsCmd = &cobra.Command{
	Use:   "scripts",
	Short: L("Show registered scripts"),
	Long:  L("Show registered scripts"),
	RunE: func(cmd *cobra.Command, args []string) error {
		return printRegisteredScripts(cmd.OutOrStdout())
	},
}

func printRegisteredScripts(w io.Writer) error {
	err := loadRegisteredScripts()
	if err != nil {
		return err
	}

	for _, name := range registeredScriptNames() {
		_, err := fmt.Fprintln(w, name)
		if err != nil {
			return err
		}
	}

	return nil
}

func registeredScriptNames() []string {
	names := make([]string, 0, len(v8.Scripts))
	for name := range v8.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
