package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func main() {
	//fileName := "C:\\Users\\47029\\Desktop\\aaa"
	fileName := "."
	fileName = strings.Replace(fileName, "\\", "/", -1)
	folders, err := readFolder(fileName)
	if err != nil {
		log.Fatal(err)
	}
	wg := &sync.WaitGroup{}
	for _, folder := range folders {
		wg.Add(1)
		go func(f string) {
			_ = merge(f)
			wg.Done()
		}(folder)
	}
	fmt.Println("START")
	wg.Wait()
}

// 将视频缓存文件合并成mp4

func readFolder(folderPath string) ([]string, error) {
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return []string{}, err
	}
	folders := make([]string, 0)
	for _, file := range files {
		if file.IsDir() {
			folders = append(folders, folderPath+"/"+file.Name())
		}
	}
	return folders, nil
}

func merge(folderPath string) error {
	lastIndex := strings.LastIndex(folderPath, "/")
	rootPath := folderPath[:lastIndex]
	saveFolder := rootPath + "/video"
	if !isDir(saveFolder) {
		err := os.Mkdir(saveFolder, os.ModePerm)
		if err != nil {
			return err
		}
	}
	videoName := saveFolder + folderPath[lastIndex:] + ".mp4"

	// 遍历当前文件夹下面的所有文件
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}
	fileNameList := make([]string, 0)
	for _, file := range files {
		if !file.IsDir() {
			fileNameList = append(fileNameList, file.Name())
		}
	}
	if len(fileNameList) == 0 {
		return nil
	}
	total := make([]byte, 0)
	if len(fileNameList) == 1 {
		tFileName := folderPath + "/" + fileNameList[0]
		tc, err := ioutil.ReadFile(tFileName)
		if err != nil {
			return nil
		}
		total = append(total, tc...)
	} else {
		// 需要将文件名重新排序
		// 1. 提取公共部分，为了得到后面的序号
		f1, f2 := fileNameList[0], fileNameList[1]
		samePos := 0
		for i, c := range f1 {
			samePos = i
			if c != int32(f2[i]) {
				break
			}
		}

		newFileNameList := make(map[int]string)
		indexList := make([]int, 0)
		for _, fileName := range fileNameList {
			sIndex := fileName[samePos:]
			index, _ := strconv.Atoi(sIndex)
			newFileNameList[index] = fileName
			indexList = append(indexList, index)
		}
		sort.Ints(indexList)
		for _, i := range indexList {
			tFileName := folderPath + "/" + newFileNameList[i]
			tc, err := ioutil.ReadFile(tFileName)
			if err != nil {
				continue
			}
			total = append(total, tc...)
		}
	}
	if len(total) > 0 {
		_ = ioutil.WriteFile(videoName, total, 0755)
		fmt.Println("SUCCESS:", videoName)
	}
	return nil
}

func isDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}
