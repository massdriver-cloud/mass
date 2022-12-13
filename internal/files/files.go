package files

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const USER_RW = 0600

func Write(path string, data interface{}) error {
	var formattedData []byte
	ext := filepath.Ext(path)

	switch ext {
	case ".json":
		json, err := json.MarshalIndent(data, "", "  ")
		formattedData = json
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	return os.WriteFile(path, formattedData, USER_RW)
}

func Read(path string, v any) error {
	ext := filepath.Ext(path)

	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	switch ext {
	case ".json":
		if err := json.Unmarshal(contents, &v); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	return nil
}
