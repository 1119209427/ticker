package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	timerMap = map[string]*Timer{}
	scanner  *bufio.Scanner
)
var (
	errUnknown      = errors.New("unknown error")
	errNoSuchTimer  = errors.New("no such timer")
	errInputIllegal = errors.New("input error, please input legal digital")
)

const (
	disposable = iota + 1
	repeat
	ticker
)

type Timer struct {
	Title      string
	timeType   int
	time       time.Time
	running    bool
	cancelOnce bool //用户可以在不删除定时器（重复提醒的）的情况下，取消定时器的下一次提醒
	cancel     chan struct{}
	f          func()
}

func main() {
	timerMap = make(map[string]*Timer)
	scanner = bufio.NewScanner(os.Stdin)
	mean()

}

func mean() {
	for {
		choice := getOptionSelect("提醒功能如下", []string{"单次日程提醒功能", "重复日程（每日）提醒功能",
			"删除日程的提示功能", "取消重复日程的下次提醒", "退出"})

		switch choice {
		case 1:
			setDisposableTimer()
		case 2:
			setRepeatTimer()
		case 3:
			delTimer()
		case 4:
			setRepeatTimerOnce()
		case 5:
			return
		default:
			fmt.Println(errInputIllegal)
			return
		}
	}

}

func NewTicker(title string, timeType int, time time.Time, f func()) Timer {
	switch timeType {
	case disposable:
		title = "[单次]" + title
	case repeat:
		title = "[重复}" + title
	case ticker:
		title = "[有间隔]" + title
	default:
		return Timer{}
	}
	return Timer{
		Title:    title,
		f:        f,
		time:     time,
		timeType: timeType,
	}
}

// AddTicker 添加定时器
func AddTicker(t Timer) {
	timerMap[t.Title] = &t
	go t.Run()
}

//计算下一次时间
func (t *Timer) nextTime() time.Time {
	if t.timeType == disposable {
		return t.time
	}
	now := time.Now()
	if t.timeType == repeat {
		nextTime := time.Date(now.Year(), now.Month(), now.Day(), t.time.Hour(), t.time.Minute(), t.time.Second(), 0, now.Location())
		//如果是当天
		if now.Before(nextTime) {
			return nextTime
		}

		//不是当天
		return nextTime.AddDate(0, 0, 1)
	}
	//间隔时间
	tick := time.Duration(t.time.Hour())*time.Hour + time.Duration(t.time.Minute())*time.Minute + time.Duration(t.time.Second())*time.Second
	return now.Add(tick)
}

func (t *Timer) Run() {
	if t.running {
		return
	}
	t.running = true
	t.cancel = make(chan struct{})
	for {
		nextTime := t.nextTime()
		sub := time.Since(nextTime)
		if t.timeType == disposable {
			s := fmt.Sprintf("你预定的日程%s将于%f分钟后开始", t.Title, sub.Minutes())
			fmt.Println(s)
		}

		timer := time.NewTimer(sub)
		select {
		case <-timer.C:
			if t.timeType == disposable {
				t.f()
				t.DeleteTimer()
				return
			}
			if t.cancelOnce {
				t.cancelOnce = false
				continue
			}
			t.f()

		case <-t.cancel:
			return
		}
	}
}

func (t *Timer) DeleteTimer() {
	if t == nil {
		return
	}

	delete(timerMap, t.Title)
	t.Cancel()
}

func (t *Timer) Cancel() {
	if t == nil {
		return
	}

	close(t.cancel)
	t.running = false
}

func (t *Timer) CancelOnce() {
	if t == nil {
		return
	}

	t.cancelOnce = true
}

func setDisposableTimer() {
	fmt.Println("请设置提醒的时间(格式为: 2006-01-02 15:04:05)")
	input := getInput()
	parse, err := time.Parse(`2006-01-02 15:04:05`, input)
	if err != nil {
		log.Println(errInputIllegal)
	}
	fmt.Println("请设置提醒的内容")
	input = getInput()
	ticker := NewTicker(input, disposable, parse, func() {})
	AddTicker(ticker)
	fmt.Println("添加成功")
}

func setRepeatTimer() {
	fmt.Println("请设置提醒的时间(格式为: 15:04:05)")
	input := getInput()
	parse, err := time.Parse(`15:04:05`, input)
	if err != nil {
		log.Println(errInputIllegal)
	}
	fmt.Println("请设置提醒的内容")
	input = getInput()
	ticker := NewTicker(input, repeat, parse, func() {})
	AddTicker(ticker)
	fmt.Println("添加成功")

}

func setRepeatTimerOnce() {
	//获取所有得重复定时器
	ts := make([]string, 0, len(timerMap))
	for title, timer := range timerMap {
		if timer.timeType == repeat {
			ts = append(ts, title)
		}
	}
	if len(ts) == 0 {
		fmt.Println("现在没有重复日程提醒器")
	}
	choice := getOptionSelect("请输入要取消下次重复日程提醒器的内容", ts)
	if choice == 0 {
		fmt.Println(errUnknown)
		return
	}
	timerMap[ts[choice-1]].CancelOnce()
	fmt.Println("取消下次提醒成功")

}

func delTimer() {
	ts := make([]string, 0, len(timerMap))
	for title := range timerMap {
		ts = append(ts, title)
	}
	if len(ts) == 0 {
		fmt.Println("现在没有日程提醒器")
	}
	choice := getOptionSelect("请输入要删除得日程提醒器的内容", ts)
	if choice == 0 {
		fmt.Println(errUnknown)
		return
	}
	timerMap[ts[choice-1]].DeleteTimer()
	fmt.Println("删除成功")
}

func getInput() string {
	scanner.Scan()
	return scanner.Text()
}

func getOptionSelect(title string, sList []string) int {
	if len(sList) == 0 {
		return 0
	}
	fmt.Println(title)
	for i, s := range sList {
		fmt.Printf("%d.%s\n", i+1, s)
	}
	fmt.Println("请输入序号")
	for {
		content := getInput()
		choice, err := strconv.Atoi(content)
		if err != nil || choice <= 0 || choice > len(sList) {
			fmt.Println(errInputIllegal)
			continue
		}

		return choice
	}

}
