package utils

import "fmt"

var bSizes = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
var sizes = []string{"", "K", "M", "G", "T", "P", "E"}

func HumanReadableBytes(amount float64) string {
	return humanReadableSize(amount, 1024, "%.2f%s", bSizes)
}
func HumanReadableBytesPrecision(amount float64, precision int) string {
	return humanReadableSize(amount, 1024, "%."+fmt.Sprintf("%d", precision)+"f%s", bSizes)
}

func HumanReadableSize(amount float64) string {
	return humanReadableSize(amount, 1000, "%.2f%s", sizes)
}

func HumanReadableSizePrecision(amount float64, precision int) string {
	return humanReadableSize(amount, 1000, "%."+fmt.Sprintf("%d", precision)+"f%s", sizes)
}

func humanReadableSize(amount float64, power float64, prec string, pSizes []string) string {
	unitsLimit := len(pSizes)
	i := 0
	size := amount
	for size >= power && i < unitsLimit {
		size = size / power
		i++
	}
	return fmt.Sprintf(prec, size, pSizes[i])
}
