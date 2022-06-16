package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/hpcloud/tail"
	"os"
	"strings"
	"time"
)

var (
	filePath string
	level    string
	levelMap map[string]struct{}
)

func init() {
	flag.StringVar(&filePath, "filePath", "default", "log file path")
	flag.StringVar(&level, "level", "", "filter log level")
}

func main() {
	flag.Parse()
	_, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("file (%s) not exit\n", filePath)
		return
	}
	fmt.Printf("Start tail file (fileter %s): %s\n", level, filePath)

	levelList := strings.Split(level, ",")
	levelMap = make(map[string]struct{})
	for _, le := range levelList {
		if len(le) > 0 {
			levelMap[le] = struct{}{}
		}
	}
	goTail(filePath)
}

type FilterStruct struct {
	Level string `json:"_level_"`
}

func goTail(filePath string) {
	config := tail.Config{
		Location: &tail.SeekInfo{
			Offset: 0, // 从文件的哪个地方开始读
			Whence: 2,
		},
		ReOpen:    true,  // 重新打开
		MustExist: false, // 文件不存在不抱错
		Poll:      true,
		Follow:    true, // 是否跟谁
	}

	tails, err := tail.TailFile(filePath, config)
	if err != nil {
		fmt.Println("tail file tailed, err: ", err)
		return
	}

	var (
		line *tail.Line
		ok   bool
	)
	for {
		line, ok = <-tails.Lines // 遍历chan, 读取日志内容
		if !ok {
			fmt.Printf("tail file close reopen, fileName %s\n", tails.Filename)
			time.Sleep(time.Second)
			continue
		}

		if len(levelMap) > 0 {
			filterStruct := FilterStruct{}
			err = json.Unmarshal([]byte(line.Text), &filterStruct)
			if err == nil {
				if _, ok := levelMap[filterStruct.Level]; !ok {
					continue
				}
			}
		}

		var str bytes.Buffer
		_ = json.Indent(&str, []byte(line.Text), "", "    ")
		fmt.Printf("\033[1;37;41m%s\033[0m", "LINE: ")
		fmt.Println(str.String())
	}
}
