package formatters

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	. "code.cloudfoundry.org/cli/cf/i18n"
)

const (
	BYTE     = 1.0
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE
)

func ByteSize(bytes int64) string {
	unit := ""
	value := float32(bytes)

	switch {
	case bytes >= TERABYTE:
		unit = "T"
		value = value / TERABYTE
	case bytes >= GIGABYTE:
		unit = "G"
		value = value / GIGABYTE
	case bytes >= MEGABYTE:
		unit = "M"
		value = value / MEGABYTE
	case bytes >= KILOBYTE:
		unit = "K"
		value = value / KILOBYTE
	case bytes == 0:
		return "0"
	case bytes < KILOBYTE:
		unit = "B"
	}

	stringValue := fmt.Sprintf("%.1f", value)
	stringValue = strings.TrimSuffix(stringValue, ".0")
	return fmt.Sprintf("%s%s", stringValue, unit)
}

func ToMegabytes(s string) (int64, error) {
	parts := bytesPattern.FindStringSubmatch(strings.TrimSpace(s))
	if len(parts) < 3 {
		return 0, invalidByteQuantityError()
	}

	value, err := strconv.ParseInt(parts[1], 10, 0)
	if err != nil {
		return 0, invalidByteQuantityError()
	}

	var bytes int64
	unit := strings.ToUpper(parts[2])
	switch unit {
	case "T":
		bytes = value * TERABYTE
	case "G":
		bytes = value * GIGABYTE
	case "M":
		bytes = value * MEGABYTE
	case "K":
		bytes = value * KILOBYTE
	}

	return bytes / MEGABYTE, nil
}

var (
	bytesPattern = regexp.MustCompile(`(?i)^(-?\d+)([KMGT])B?$`)
)

func invalidByteQuantityError() error {
	return errors.New(T("Byte quantity must be an integer with a unit of measurement like M, MB, G, or GB"))
}
