package mangaparser

import (
	"bytes"
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

// ParseFromBytes parses file content directly from memory
func ParseFromBytes(data []byte, filename string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	reader := bytes.NewReader(data)

	switch ext {
	case ".csv":
		out, err := comickparser.ParseComickReader(reader)
		if err != nil {
			return nil, err
		}
		return out, nil
	case ".xml":
		out, err := malparser.ParseMALReader(reader)
		if err != nil {
			return nil, err
		}
		return malparser.ReturnMALTitles(out), nil
	default:
		return nil, fmt.Errorf("unknown file format: %s (must be .csv or .xml)", ext)
	}
}
