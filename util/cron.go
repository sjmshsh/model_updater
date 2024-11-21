package util

import (
	"github.com/robfig/cron"
	"sync"
)

// StartInterval 用于创建周期任务
// cycle周期 形如 1s 2m 3h等
func StartInterval(processFunc func(), cycle string) {
	c := cron.New()
	processFunc() // 先执行一次
	mutex := sync.Mutex{}
	if err := c.AddFunc("@every "+cycle, func() {
		mutex.Lock()
		defer mutex.Unlock()
		processFunc()
	}); err != nil {
		return
	}
	c.Start()
}

func StartCron(prcessFunc func(), spec string) {
	c := cron.New()
	mutex := sync.Mutex{}
	if err := c.AddFunc(spec, func() {
		mutex.Lock()
		defer mutex.Unlock()
		prcessFunc()
	}); err != nil {
		return
	}

	c.Start()
}
