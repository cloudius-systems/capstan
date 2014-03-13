package capstan

import (
	"fmt"
	"regexp"
	"strconv"
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
