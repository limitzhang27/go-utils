package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	datToImgDepth()
}

func datToImg()  {
	pwd, _ := os.Getwd()
	folder, err := ioutil.ReadDir(pwd)
	if err != nil {
		fmt.Println("read folder err", err)
		return
	}
	for _, info := range folder {
		if strings.HasSuffix(info.Name(), ".dat") {
			allPath := pwd + "/" + info.Name()
			datDatSaveImg(allPath)
		}
	}
}

func datToImgDepth()  {
	pwd, _ := os.Getwd()
	datList := searchDat(pwd)

	for _, file := range datList {
		datDatSaveImg(file)
	}
}

func searchDat(path string) (datFiles []string) {
	folder, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}
	for _, info := range folder {
		if strings.HasSuffix(info.Name(), ".dat") {
			datFiles = append(datFiles, filepath.Join(path, info.Name()))
		} else if info.IsDir()  {
			tmp := searchDat(filepath.Join(path, info.Name()))
			datFiles = append(datFiles, tmp...)
		}
	}
	return
}

func datDatSaveImg(fileName string)  {
	data, _ := ioutil.ReadFile(fileName)
	l := len(data)
	new_data :=  make([]byte, l, l)

	var signJpgA byte = 0xFF
	var signJpgB byte = 0xD8
	var signPngA byte = 0x89
	var signPngB byte = 0x50
	var signGifA byte = 0x47
	var signGifB byte = 0x49

	suffix := ""

	var sign byte

	if signJpgA ^ data[0] == signJpgB ^ data[1] {
		sign = signJpgA ^ data[0]
		suffix = "jpg"
	} else if signGifA ^ data[0] == signGifB ^ data[1] {
		sign = signGifA ^ data[0]
		suffix = "jpg"
	} else if  signPngA ^ data[0] == signPngB ^ data[1] {
		sign = signPngA ^ data[0]
		suffix = "jpg"
	} else {
		return
	}
	for i := 0; i < l; i++ {
		newByte := sign ^ data[i]
		new_data[i] = newByte
	}
	newFileName := fileName + "." + suffix
	path := filepath.Join(filepath.Dir(newFileName), "img")

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(path, 0777)
		}
	}

	newFileName = filepath.Join(path, filepath.Base(newFileName))
	fmt.Println(newFileName)
	_ = ioutil.WriteFile(newFileName, new_data,0777)
}
