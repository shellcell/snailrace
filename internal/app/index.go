package app

import (
	"fmt"
	"strings"

	"github.com/shellcell/snailrace/internal/analysis"
)

type indexValues struct {
	values []string
	set    bool
}

func newIndexValues() indexValues {
	return indexValues{values: append([]string(nil), analysis.DefaultIndexDimensions...)}
}

func (index *indexValues) String() string {
	return strings.Join(index.values, ",")
}

func (index *indexValues) Set(value string) error {
	if !index.set {
		index.values = nil
		index.set = true
	}
	for _, item := range strings.Split(value, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if !analysis.ValidIndexDimension(item) {
			return fmt.Errorf(
				"unknown index dimension %q; choose from %s",
				item, strings.Join(analysis.IndexDimensions, ", "),
			)
		}
		if !contains(index.values, item) {
			index.values = append(index.values, item)
		}
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
