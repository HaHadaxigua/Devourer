package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	client      = http.Client{Timeout: time.Second * 180}
	wg          sync.WaitGroup
	packageSize uint64
	localPath   string

	urls = []string{
		"http://www.baidu.com",
		"https://music.163.com/",
		"https://download.jetbrains.com/idea/ideaIU-2020.2.1.exe?_ga=2.259014541.663008056.1599903368-377724488.1586414164",
	}
)

func init() {
	// 每个协程下载4MB
	packageSize = 1048576 * 4
	r, _ := os.Getwd()
	// 文件下载路径
	localPath = r + "/lib"
}

func main() {

}

func start(url string){
	var dst *os.File
	var rn, wn int
	var filename string
	var fi os.FileInfo
	req, _ := http.NewRequest("GET", url, nil)

}

func Download(url, filename, dir string, msg chan int) {
	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	if filename == "" {
		value, ok := res.Header["Content-Disposition"]
		if ok {
			temp := strings.Split(value[0], "filename=")
			if len(temp) > 1 {
				filename = temp[1]
			} else {
				filename = temp[0]
			}
		} else {

		}
	}
}

func GetHttpStatus(url string) (httpStatus int) {
	resp, err := http.Head(url)
	if err != nil {
		fmt.Printf("err:%v\n", err)
	}
	httpStatus = resp.StatusCode
	return
}

func GetHttpBody(url string) {
	resp, err := http.Get(url)
	checkError(err)
	data, err := ioutil.ReadAll(resp.Body)
	checkError(err)
	fmt.Printf("Got:%v\n", string(data))
}

func checkError(err error) {
	if err != nil {
		log.Fatalf("Get: %v", err)
	}
}
