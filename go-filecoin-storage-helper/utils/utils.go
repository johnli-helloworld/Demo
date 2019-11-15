package utils

import (
	"fmt"
	"math"
	"os"
	"path"
	"strings"
)

//生成数据库名称，用
func GeneratedbName(filepath string) string {
	new := strings.Replace(filepath, "/", "-", -1)
	return strings.TrimPrefix(new, "-")
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

// 判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

//根据给定的文件路径名，创建文件
func GenerateFileByPath(fullpath string) error {
	if Exists(fullpath) {
		return nil
	}
	dirs := path.Dir(fullpath)
	if err := os.MkdirAll(dirs, 0755); err != nil {
		return err
	}
	_, err := os.Create(fullpath)
	if err != nil {
		fmt.Println("GenerateFileByPath err:", err)
		return err
	}
	return nil
}

func RemoveFileOrDir(path string) error {
	if !Exists(path) {
		fmt.Println("RemoveFileOrDir: file or dir not found")
		return nil
	}
	f, err := os.Stat(path)
	if err != nil {
		fmt.Println("RemoveFileOrDir: stat err", err)
		return err
	}
	if f.IsDir() {
		os.RemoveAll(path)
	} else {
		os.Remove(path)
	}
	return nil
}
