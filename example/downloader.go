package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

/**
	通过http实现的多线程下载工具
 */

//FileDownloader 新文件任务
type FileDownloader struct {
	fileSize       int
	url            string
	outputDir      string
	outputFilename string
	totalPart      int
	donePart       []*filePart
}

//filePart 文件分片
type filePart struct {
	Index int    //文件分片的序号
	From  int    //开始byte
	To    int    //解决byte
	Data  []byte //http下载得到的文件内容
}

//parseFileInfoFrom 从resp中解析出文件名
func parseFileInfoFrom(resp *http.Response) string {
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		// 函数根据RFC 1521解析一个媒体类型值以及可能的参数，v即http header中的content-type。
		//媒体类型值一般应为Content-Type和Content-Disposition头域的值（参见RFC 2183）。
		//成功的调用会返回小写字母、去空格的媒体类型和一个非空的map。返回的map映射小写字母的属性和对应的属性值。
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err != nil {
			panic(err)
		}
		return params["filename"]
	}
	// Base returns the last element of path.
	filename := filepath.Base(resp.Request.URL.Path)
	return filename
}

//newFileDownloader 构建新的下载任务
func NewFileDownloader(url, outputDir, outputFilename string, totalPart int) *FileDownloader {
	if outputDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			log.Println(err)
		}
		outputDir = wd
	}
	return &FileDownloader{
		fileSize:       0,
		url:            url,
		outputDir:      outputDir,
		outputFilename: outputFilename,
		totalPart:      totalPart,
		donePart:       make([]*filePart, totalPart),
	}
}

func main(){
	startTime := time.Now()
	golandDownloaderUrl := "https://download.jetbrains.com/go/goland-2020.2.3.exe?_ga=2.180828967.1034922733.1600739386-377724488.1586414164"
	maxProcess := runtime.GOMAXPROCS(runtime.NumCPU())
	downloader := NewFileDownloader(golandDownloaderUrl, "", "", maxProcess)
	if err := downloader.Run(); err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("\n文件下载完成耗时: %f second\n", time.Now().Sub(startTime).Seconds())
}

func(downloader *FileDownloader) Run() error{
	fileTotalSize, err := downloader.head()
	if err!=nil{
		return err
	}
	downloader.fileSize = fileTotalSize

	jobs := make([]filePart, downloader.totalPart)
	eachSize := fileTotalSize/downloader.totalPart

	for i:=range jobs{
		jobs[i].Index = i
		if i == 0{
			jobs[i].From = 0
		}else{
			jobs[i].From = jobs[i-1].To + 1
		}
		if i < downloader.totalPart-1{
			jobs[i].To = jobs[i].From + eachSize
		}else{
			jobs[i].To = fileTotalSize-1
		}
	}

	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(job filePart) {
			defer wg.Done()
			err := downloader.downloadPart(&job)
			if err != nil {
				log.Println("文件下载失败", err, job)
			}
		}(j)
	}
	wg.Wait()
	return downloader.mergeFileParts()
}

func(downloader *FileDownloader) mergeFileParts() error {
	log.Println("开始合并文件")
	path := filepath.Join(downloader.outputDir, downloader.outputFilename)
	mergedFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer mergedFile.Close()

	//hash := sha256.New()
	totalSize := 0
	for _, e := range downloader.donePart{
		mergedFile.Write(e.Data)
		//hash.Write(e.Data)
		totalSize += len(e.Data)
	}
	if totalSize != downloader.fileSize {
		return errors.New("文件不完整")
	}

	//if string(hash.Sum(nil))!= "201211b84b2e425d2da0bcec6f070a20688e697ee28ff0c362ba741682072315 *goland-2020.2.3.exe"{
	//	return errors.New("文件损坏")
	//}else{
	//	log.Println("文件SHA-256校验成功")
	//}
	return nil
}


// 下载分片
func(downloader *FileDownloader) downloadPart (part *filePart) error{
	r, err := downloader.getNewRequest("GET")
	if err != nil{
		return err
	}
	log.Printf("开始[%d]下载from:%d to:%d\n", part.Index, part.From, part.To)
	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", part.From, part.To))
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("服务器错误状态码: %v", resp.StatusCode))
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(bs) != (part.To - part.From+1){
		return errors.New("下载分片长度错误")
	}
	part.Data = bs
	downloader.donePart[part.Index] = part
	return nil
}


// 获取下载文件的基本信息 header
// 返回文件大小
func(downloader *FileDownloader) head()(int, error){
	r, err := downloader.getNewRequest("HEAD")
	if err!=nil{
		return 0, err
	}
	resp, err := http.DefaultClient.Do(r)
	if err!=nil{
		return 0, err
	}

	if resp.StatusCode > 299 {
		return 0, errors.New(fmt.Sprintf("Can't process, response is %v", resp.StatusCode))
	}
	// 判断是否支持断点续传
	if resp.Header.Get("Accept-Ranges")!="bytes"{
		return 0, errors.New("服务器不支持文件断点续传")
	}
	downloader.outputFilename = parseFileInfoFrom(resp)
	return strconv.Atoi(resp.Header.Get("Content-Length"))
}

// 从url创建一个request
func (downloader *FileDownloader) getNewRequest(method string)(*http.Request, error){
	r, err := http.NewRequest(method, downloader.url, nil)
	if err!=nil{
		return nil, err
	}
	r.Header.Set("User-Agent", "Mozilla/5.0")
	return r, nil
}

