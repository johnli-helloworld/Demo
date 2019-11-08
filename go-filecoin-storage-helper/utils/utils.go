package utils

import (
	"math"
	"os"
	"path"
	"strings"
)

//TODO:暂时为了测试，将/全部替换为_
func GeneratedbName(filepath string) string {
	return filepath
}

func ComputeChunks(totalsize uint64, slicesize uint64) int {
	num := int(math.Ceil(float64(totalsize) / float64(slicesize)))
	return num
}

func NewPath(ns []string) string {
	return strings.Join(ns, "/")
}

//判断给定路径的文件是否存在，不存在则创建
func FileChecker(filepath string) (*os.File, error) {
	f, err := os.Open(filepath)
	if err != nil && os.IsNotExist(err) {
		tf, err := os.Create(filepath)
		if err != nil {
			return nil, err
		}
		return tf, err
	}
	return f, nil
}

//根据给定的文件路径名，创建文件
func GenerateFileByPath(fullpath string) (error) {
	dirs := path.Dir(fullpath)
	if err := os.MkdirAll(dirs, 0755); err != nil {
		return err
	}
	_, err := os.Create(fullpath)
	if err != nil {
		return  nil
	}
	return nil
}
