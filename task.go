package main

import (
	"errors"
	"fmt"
	"github.com/HaHadaxigua/Devourer/tool"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type task struct {
	done          chan struct{} // 完成任务则向通道发送空接口数据
	src           io.ReadCloser // 一个接口， 提供了基本的读取和关闭方法
	dst           io.WriteCloser
	bytePreSecond float64 // 每秒byte数量
	err           error
	startTime     time.Time
	endTime       time.Time
	mutex         sync.Mutex
	readNum       int64 // 已读取字符树
	fileSize      int64
	fileName      string
	buffer        []byte
	lim           *rateLimiter // 速度限制器
	url           string
	isResume      bool // 是否可以续传
	header        map[string]string
}

func (t *task) getReadNum() int64 {
	if t == nil {
		return 0
	}
	return atomic.LoadInt64(&t.readNum)
}

func newTask(url string, h map[string]string) *task {
	lim, url := tool.GetLimitFromUrl(url)
	return &task{
		url:    url,
		done:   make(chan struct{}, 1),
		buffer: make([]byte, 1024),
		lim:    &rateLimiter{lim: lim * 1000},
		header: h,
	}
}

func (t *task) start() {
	defer func() {
		if err := recover(); err != nil {
			switch x := err.(type) {
			case string:
				t.err = errors.New(x)
			case error:
				t.err = x
			default:
				t.err = errors.New("Unknow panic")
			}
			close(t.done)
			t.endTime = time.Now()
		}
	}()

	var dst *os.File
	var rn, wn int
	var filename string
	var fi os.FileInfo // A FileInfo describes a file and is returned by Stat and Lstat.
	req, _ := http.NewRequest("GET", t.url, nil)
	if t.header != nil {
		for k, v := range t.header {
			req.Header.Set(k, v)
		}
	}
	// 代表http客户端
	c := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	// 发起请求
	resp, err := c.Do(req)
	if err != nil {
		goto done
	} else if resp.StatusCode != 200 && resp.StatusCode != 206 {
		err = errors.New(fmt.Sprintf("wrong response %d", resp.StatusCode))
		goto done
	}

	filename, err = tool.GuessFilename(resp)

	// Stat returns a FileInfo describing the named file.
	// If there is an error, it will be of type *PathError.
	fi, err = os.Stat(filename)

	if err == nil {
		if !fi.IsDir() {
			// 当你使用标准http库发起请求时，你得到一个http的响应变量。如果你不读取响应主体，你依旧需要关闭它。
			// 为了提高效率，http.Get 等请求的 TCP 连接是不会关闭的（再次向同一个域名请求时，复用连接），所以必须要手动关闭。
			resp.Body.Close()
			if fi.Size() == resp.ContentLength {
				err = errors.New("File is downloaded! ")
				goto done
			}
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", fi.Size()))
			resp, err = c.Do(req)
			if err != nil {
				goto done
			} else if resp.StatusCode != 200 && resp.StatusCode != 206 {
				err = errors.New(fmt.Sprintf("wrong response %d", resp.StatusCode))
				goto done
			}
			if resp.Header.Get("Accept-Ranges") == "bytes" || resp.Header.Get("Content-Range") != "" {
				// 打开一个文件，可读可写，0666：任何人都可以读写，但不能执行
				dst, err = os.OpenFile(filename, os.O_RDWR, 0666)
				if err != nil {
					goto done
				}
				dst.Seek(0, io.SeekEnd)
				t.readNum = fi.Size()
				t.isResume = true
			}
		}
	}

	if dst == nil {
		dst, err = os.Create(filename) // 创建文件
		if err != nil {
			goto done
		}
	}

	t.dst = dst
	t.src = resp.Body
	t.fileName = filename
	if resp.ContentLength > 0 && t.isResume && fi != nil {
		t.fileSize = resp.ContentLength + fi.Size()
	} else {
		t.fileSize = resp.ContentLength
	}

	go t.bps()

	t.startTime = time.Now()

loop:
	if t.lim.lim > 0 {
		t.lim.wait(t.readNum)
	}

	rn, err = t.src.Read(t.buffer)

	if rn > 0 {
		wn, err = t.dst.Write(t.buffer[:rn])
		if err != nil {
			goto done
		} else if rn != wn {
			err = io.ErrShortWrite
			goto done
		} else {
			atomic.AddInt64(&t.readNum, int64(rn))
			goto loop
		}
	}

done:
	t.err = err
	close(t.done)
	t.endTime = time.Now()
	return
}

func (t *task) bps() {

}
