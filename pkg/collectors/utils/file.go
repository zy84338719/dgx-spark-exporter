package utils

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func ReadFileString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func ReadFileFloat(path string) (float64, error) {
	s, err := ReadFileString(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

func ReadFileUint64(path string) (uint64, error) {
	s, err := ReadFileString(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(s, 10, 64)
}

func ReadFileInt(path string) (int, error) {
	s, err := ReadFileString(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(s)
}

func ParseKeyValueFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			result[key] = val
		}
	}
	return result, scanner.Err()
}

func ParseKeyValueFileUint64(path string) (map[string]uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]uint64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")
		valStr = strings.TrimSpace(valStr)
		if val, err := strconv.ParseUint(valStr, 10, 64); err == nil {
			result[key] = val
		}
	}
	return result, scanner.Err()
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
