package utils

import (
	"math"
)

func GeneratedbName(filepath string) string {
	return ""
}

func ComputeChunks(totalsize int64, slicesize int64) int {
	num := int(math.Ceil(float64(totalsize) / float64(slicesize)))
	return num
}

func GetRelativePath() {

}
