package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func init() {
	flag.Parse()
}

func main() {
	cmdArgs := os.Args
	if len(cmdArgs) != 2 {
		fmt.Println("参数错误")
		return
	}
	fmt.Println("开始下载")
	url := cmdArgs[1]
	//url := "https://cdn77-vid.xvideos-cdn.com/2mFnXYLyrBHerqWxG4VblQ==,1676916797/videos/hls/ce/35/37/ce353706b886dbe240ed6f5f09125ad1/hls-1080p-7ee76.m3u8"
	index := strings.LastIndex(url, "/")
	newUrl := url[:index]
	fileName := url[index+1 : len(url)-5]
	response, err := http.Get(url)
	if err != nil {
		_ = fmt.Errorf("url解析错误")
		return
	}
	fmt.Println("解析url成功")
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := ioutil.ReadAll(response.Body)

	tmp := strings.Split(string(body), "\n")

	list := make([]string, 0, len(tmp)/2)
	for _, s := range tmp {
		if len(s) > 3 && s[len(s)-3:] == ".ts" {
			list = append(list, newUrl+"/"+s)
			//fmt.Println(newUrl+"/"+s)
		}
	}
	fmt.Println("获取ts地址成功")
	downloadM2u8(fileName, list)
}

func downloadM2u8(fileName string, list []string) {
	folderName := fileName

	_ = os.RemoveAll(folderName)

	err := os.Mkdir(folderName, os.ModePerm)

	if err != nil {
		_ = fmt.Errorf("创建文件夹失败")
		return
	}

	for _, url := range list {
		index := strings.LastIndex(url, "/")
		fileName := url[index+1:]
		response, err := http.Get(url)
		if err != nil {
			_ = fmt.Errorf("下载失败")
			return
		}

		filePath := folderName + "/" + fileName
		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			_ = fmt.Errorf("生成视频文件失败")
			return
		}
		body, _ := ioutil.ReadAll(response.Body)
		write := bufio.NewWriter(file)
		_, _ = write.Write(body)
		fmt.Println("下载成功:", url)
		_ = file.Close()
		_ = response.Body.Close()
	}

	err = merge(folderName, fileName)
	if err != nil {
		_ = fmt.Errorf("合并失败")
	}
}

func merge(folderPath string, fileName string) error {
	videoName := fileName + ".mp4"

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
	for i, _ := range fileNameList {
		tFileName := folderPath + "/" + fileName + strconv.Itoa(i) + ".ts"
		tc, err := ioutil.ReadFile(tFileName)
		if err != nil {
			continue
		}
		total = append(total, tc...)
	}
	if len(total) > 0 {
		_ = ioutil.WriteFile(videoName, total, 0755)
		fmt.Println("SUCCESS:", videoName)
	}
	return nil
}
