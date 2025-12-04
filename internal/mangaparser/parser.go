package mangaparser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Another0Noob/mangadex-import/internal/mangaparser/comickparser"
	"github.com/Another0Noob/mangadex-import/internal/mangaparser/malparser"
)

func Parse(path string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".csv":
		out, err := comickparser.ParseComickFile(path)
		if err != nil {
			return nil, err
		}
		return out, nil
	case ".xml":
		out, err := malparser.ParseMALFile(path)
		if err != nil {
			return nil, err
		}
		return malparser.ReturnMALTitles(out), nil
	default:
		return nil, fmt.Errorf("unknown file format: %s (must be .csv or .xml)", ext)
	}
}
