package files

import (
	"encoding/json"
	"os"
)

const USER_RW = 0600

func Write(path string, data interface{}) error {
	// TODO: check file type first
	json, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, json, USER_RW)
}

func Read(path string, v any) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// TODO: check file type first
	err = json.Unmarshal(contents, &v)

	if err != nil {
		return err
	}

	return nil
}
