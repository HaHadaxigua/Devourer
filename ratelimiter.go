package main

import "time"

/**
速度限制器
*/

type rateLimiter struct {
	readNum  int64
	pastTime time.Time
	lim      int64
}

func (r *rateLimiter) wait(readNum int64) {
	if int(time.Now().UnixNano())-int(r.pastTime.UnixNano()) <= int(time.Second) {
		d := readNum - r.readNum
		if d >= r.lim {
			x := time.Second.Nanoseconds() - (time.Now().UnixNano() - r.pastTime.UnixNano())
			time.Sleep(time.Duration(x))
			r.readNum = readNum
			r.pastTime = time.Now()
		}
	}else{
		r.readNum = readNum
		r.pastTime = time.Now()
	}
}
