package utils

import "fmt"

var bSizes = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
var sizes = []string{"", "K", "M", "G", "T", "P", "E"}

func HumanReadableBytes(origSize float64) string {
	return humanReadableSize(origSize, 1024, bSizes)
}

func HumanReadableSize(origSize float64) string {
	return humanReadableSize(origSize, 1000, sizes)
}

func humanReadableSize(origSize float64, power float64, pSizes []string) string {
	unitsLimit := len(pSizes)
	i := 0
	size := origSize
	for size >= power && i < unitsLimit {
		size = size / power
		i++
	}

	return fmt.Sprintf("%.2f%s", size, pSizes[i])
}
