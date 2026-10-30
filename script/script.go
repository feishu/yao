package script

import (
	"fmt"

	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load all scripts and services (using double-buffering for zero-downtime hot reload)
func Load(cfg config.Config) error {
	v8.CLearModules()
	setLastLoadErrors(nil)
	exts := []string{"*.js", "*.ts"}
	loadErrs := &LoadErrors{}

	draftScripts := make(map[string]*v8.Script)

	err := application.App.Walk("scripts", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		id := share.ID(root, file)
		_, err := v8.LoadTo(file, id, draftScripts)
		if err != nil {
			loadErrs.Add(file, id, err)
		}
		return nil
	}, exts...)

	if err != nil {
		return err
	}

	if has, _ := application.App.Exists("services"); has {
		err = application.App.Walk("services", func(root, file string, isdir bool) error {
			if isdir {
				return nil
			}
			id := fmt.Sprintf("__yao_service.%s", share.ID(root, file))
			_, err := v8.LoadTo(file, id, draftScripts)
			if err != nil {
				loadErrs.Add(file, id, err)
			}
			return nil
		}, exts...)

		if err != nil {
			return err
		}
	}

	// Atomically replace all scripts in global registry: 0 downtime, 0 missing window!
	v8.ReplaceScripts(draftScripts)

	if len(loadErrs.Items) == 0 {
		return nil
	}

	loadErrs.Sort()
	setLastLoadErrors(loadErrs)
	return loadErrs
}
