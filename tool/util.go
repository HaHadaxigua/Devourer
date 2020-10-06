package tool

/**
MIME(Multipurpose Internet Mail Extensions)多用途互联网邮件扩展类型,
设计的最初目的是为了在发送电子邮件时附加多媒体数据，让邮件客户程序能根据其类型进行处理
之后则是用来设定某种扩展名的文件用一种应用程序来打开的方式类型，
该扩展名文件被访问的时候，浏览器会自动使用指定应用程序来打开。
多用于指定一些客户端自定义的文件名，以及一些媒体文件打开方式。
*/

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	kib = 1024
	mib = 1048576
	gib = 1073741824
	tib = 1099511627776
)

var (
	errNoFilename = errors.New("no filename cloud be determined")
)

func FormatBytes(l int64) (res string) {
	switch {
	case l >= tib:
		res = fmt.Sprintf("%6.2fTB", float64(l)/tib)
	case l >= gib:
		res = fmt.Sprintf("%6.2fGB", float64(l)/gib)
	case l >= mib:
		res = fmt.Sprintf("%6.2fMB", float64(l)/mib)
	case l >= kib:
		res = fmt.Sprintf("%6.2fKB", float64(l)/kib)
	default:
		res = fmt.Sprintf("%7dB", l)
	}
	if len(res) > 8 {
		res = strings.Join([]string{res[:6], res[7:]}, "")
	}
	return
}

func GetLimitFromUrl(url string) (int64, string) {
	s := strings.Split(url, ":")
	if len(s) >= 2 {
		i, err := strconv.ParseInt(s[0], 0, 0)
		if err != nil {
			return -1, url
		} else {
			return i, strings.Join(s[1:], ":")
		}
	}
	return -1, url
}

func GuessFilename(resp *http.Response) (string, error) {
	filename := resp.Request.URL.Path
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		/**
		mime.ParseMediaType函数根据RFC 1521解析一个媒体类型值以及可能的参数，参数即http header中的content-type。
		媒体类型值一般应为Content-Type和Conten-Disposition头域的值（参见RFC 2183）。
		成功的调用会返回小写字母、去空格的媒体类型和一个非空的map。
		返回的map映射小写字母的属性和对应的属性值。
		*/
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			filename = params["filename"]
		}
	}

	if filename == "" || strings.HasSuffix(filename, "/") || strings.Contains(filename, "\x00") {
		return "", errNoFilename
	}

	// filepath.Base() 获取路径中最后一个分隔符之后的元素
	// path.Clean() 返回路径的最后一个元素  ”/a/b“ 会返回b
	filename = filepath.Base(path.Clean("/" + filename))

	if filename == "" || filename == "." || filename == "/"{
		return "",errNoFilename
	}
	return filename, nil
}
