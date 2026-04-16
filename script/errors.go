package script

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	v8 "github.com/yaoapp/gou/runtime/v8"
)

// LoadError a script load error
type LoadError struct {
	ID     string
	File   string
	Line   int
	Column int
	Text   string
}

func (err LoadError) Error() string {
	switch {
	case err.Line > 0 && err.Column > 0:
		return fmt.Sprintf("%s:%d:%d %s", err.File, err.Line, err.Column, err.Text)
	case err.Line > 0:
		return fmt.Sprintf("%s:%d %s", err.File, err.Line, err.Text)
	default:
		return fmt.Sprintf("%s %s", err.File, err.Text)
	}
}

// LoadErrors script load error set
type LoadErrors struct {
	Items []LoadError
}

func (errs *LoadErrors) Error() string {
	if errs == nil || len(errs.Items) == 0 {
		return ""
	}

	lines := make([]string, 0, len(errs.Items))
	for _, item := range errs.Items {
		lines = append(lines, item.Error())
	}

	return strings.Join(lines, "\n")
}

func (errs *LoadErrors) Add(file, id string, err error) {
	if err == nil {
		return
	}

	var transformErr *v8.TransformError
	if errors.As(err, &transformErr) {
		for _, item := range transformErr.Items {
			errs.add(LoadError{
				ID:     id,
				File:   normalizeLoadErrorFile(file, item.File),
				Line:   item.Line,
				Column: item.Column,
				Text:   item.Text,
			})
		}
		return
	}

	errs.add(LoadError{
		ID:   id,
		File: normalizeLoadErrorFile(file, ""),
		Text: err.Error(),
	})
}

func (errs *LoadErrors) add(item LoadError) {
	if errs == nil {
		return
	}

	key := item.Error()
	for _, existing := range errs.Items {
		if existing.Error() == key {
			return
		}
	}

	errs.Items = append(errs.Items, item)
}

func (errs *LoadErrors) Sort() {
	if errs == nil {
		return
	}

	sort.Slice(errs.Items, func(i, j int) bool {
		left := errs.Items[i]
		right := errs.Items[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		return left.Text < right.Text
	})
}

var lastLoadErrors *LoadErrors
var lastLoadErrorsLock sync.RWMutex

// LastLoadErrors returns the latest script load errors
func LastLoadErrors() *LoadErrors {
	lastLoadErrorsLock.RLock()
	defer lastLoadErrorsLock.RUnlock()
	return lastLoadErrors
}

func setLastLoadErrors(errs *LoadErrors) {
	lastLoadErrorsLock.Lock()
	defer lastLoadErrorsLock.Unlock()
	lastLoadErrors = errs
}

func normalizeLoadErrorFile(defaultFile, filename string) string {
	if filename == "" {
		filename = defaultFile
	}
	return filepath.ToSlash(filename)
}
