package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func ParseMemSize(memory string) (int64, error) {
	r, _ := regexp.Compile("([0-9]+)(m|mb|M|MB|g|gb|G|GB)$")
	match := r.FindStringSubmatch(memory)
	if len(match) != 3 {
		return -1, fmt.Errorf("%s: unrecognized memory size", memory)
	}
	size, _ := strconv.ParseInt(match[1], 10, 64)
	unit := match[2]
	switch unit {
	case "g", "gb", "G", "GB":
		size *= 1024
	}
	if size == 0 {
		return -1, fmt.Errorf("%s: memory size must be larger than zero", memory)
	}
	return size, nil
}

func ParseEnvironmentList(envList []string) (map[string]string, error) {
	res := make(map[string]string)

	for _, part := range envList {
		if keyValue := strings.SplitN(part, "=", 2); len(keyValue) < 2 {
			return nil, fmt.Errorf("failed to parse --env argument '%s': missing =", part)
		} else if strings.Contains(keyValue[0], " ") {
			return nil, fmt.Errorf("failed to parse --env argument '%s': key must not contain spaces", part)
		} else if strings.Contains(keyValue[1], " ") {
			return nil, fmt.Errorf("failed to parse --env argument '%s': value must not contain spaces", part)
		} else {
			res[keyValue[0]] = keyValue[1]
		}
	}
	return res, nil
}
