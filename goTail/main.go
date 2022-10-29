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

const (
	LevelError = "error"
	LevelWarn  = "warn"
	LevelInfo  = "info"
)

func init() {
	flag.StringVar(&filePath, "f", "", "to tail file path")
	flag.StringVar(&level, "l", "", "filter level")
}

func main() {
	flag.Parse()
	fmt.Println(filePath)
	if filePath == "" {
		filePath = "./log/service.log.fls"
	}
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

type LogStruct struct {
	Level      string `json:"_level_"`
	DebugStack string `json:"debug_stack"`
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
		fmt.Println("--------")
		line, ok = <-tails.Lines // 遍历chan, 读取日志内容
		if !ok {
			fmt.Printf("tail file close reopen, fileName %s\n", tails.Filename)
			time.Sleep(time.Second)
			continue
		}

		// 简易格式化日志
		var lineByte bytes.Buffer
		_ = json.Indent(&lineByte, []byte(line.Text), "", "    ")
		str := lineByte.String()

		// 日志筛选
		logStruct := LogStruct{}
		err = json.Unmarshal([]byte(line.Text), &logStruct)
		if err != nil {
			fmt.Println(str)
			continue
		}

		logMap := make(map[string]interface{})
		d := json.NewDecoder(bytes.NewReader([]byte(line.Text)))
		d.UseNumber()
		err = d.Decode(&logMap)
		if err != nil {
			fmt.Println(str)
			continue
		}

		// 日志筛选
		level := LevelInfo
		if l, ok := logMap["_level_"]; ok {
			level = l.(string)
			if len(levelMap) > 0 {
				// 过滤日志级别
				if _, ok := levelMap[level]; !ok {
					continue
				}
			}
		}
		fmt.Printf(getColorTag(level), mapToFormatJsonStr(logMap))
		fmt.Println("")
	}
}

func mapToFormatJsonStr(m map[string]interface{}) string {
	res := strings.Builder{}
	res.WriteString("{\r\n")
	for key, value := range m {
		res.WriteString(fmt.Sprintf("    \"%s\" : %v,\r\n", key, value))
	}
	res.WriteString("}\r\n")
	return res.String()
}

func getColorTag(level string) string {
	color := "%s"
	switch level {
	case LevelError:
		color = "\033[1;31m%s\033[0m"
	case LevelWarn:
		color = "\033[1;33m%s\033[0m"
	}
	return color
}
