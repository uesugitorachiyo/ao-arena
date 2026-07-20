package arena

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

func readStrictBoundedJSON(path, label string, limit int64, target any) ([]byte, error) {
	return readStrictBoundedJSONWithHook(path, label, limit, target, nil)
}

func readStrictBoundedJSONFromRoot(root *os.Root, name, label string, limit int64, target any) ([]byte, error) {
	return readStrictBoundedJSONFromRootWithHook(root, name, label, limit, target, nil)
}

func readStrictBoundedJSONFromRootWithHook(root *os.Root, name, label string, limit int64, target any, afterLstat func() error) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular non-link file", label)
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("%s size limit exceeded", label)
	}
	if afterLstat != nil {
		if err := afterLstat(); err != nil {
			return nil, err
		}
	}

	file, err := openRootReadNoFollow(root, name, info)
	if err != nil {
		return nil, fmt.Errorf("open %s as a rooted regular non-link file: %w", label, err)
	}
	defer file.Close()
	return readStrictBoundedJSONFile(file, info, label, limit, target)
}

func readStrictBoundedJSONWithHook(path, label string, limit int64, target any, afterLstat func() error) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular non-link file", label)
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("%s size limit exceeded", label)
	}
	if afterLstat != nil {
		if err := afterLstat(); err != nil {
			return nil, err
		}
	}

	file, err := openReadNoFollow(path, info)
	if err != nil {
		return nil, fmt.Errorf("open %s as a regular non-link file: %w", label, err)
	}
	defer file.Close()
	return readStrictBoundedJSONFile(file, info, label, limit, target)
}

func readStrictBoundedJSONFile(file *os.File, expected os.FileInfo, label string, limit int64, target any) ([]byte, error) {
	openedInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(expected, openedInfo) {
		return nil, fmt.Errorf("%s must remain the same regular non-link file", label)
	}

	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s size limit exceeded", label)
	}
	if err := decodeStrictJSON(data, target); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return data, nil
}

func decodeStrictJSON(data []byte, target any) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("malformed UTF-8")
	}
	if err := rejectDuplicateJSONFields(data); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return err
	}
	return nil
}

func rejectDuplicateJSONFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return err
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}

	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object field name must be a string")
			}
			if seen[key] {
				return fmt.Errorf("duplicate field %q", key)
			}
			seen[key] = true
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return fmt.Errorf("invalid object terminator")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return fmt.Errorf("invalid array terminator")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	return nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("trailing JSON value")
	}
	return err
}
