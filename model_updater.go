package model_updater

import (
	"fmt"
	"github.com/sjmshsh/model_updater/util"
	"math"
	"math/rand"
	"sync"
	"time"
)

type LoadUpdatedData struct {
	sync.RWMutex
	DataMap          sync.Map         // 存储数据用的map
	Name             string           // 名称
	TimeCycle        string           // 加载周期
	UpdatedFunc      UpdatedFuncType  // 获取增量更新的函数
	DataProc         DataProcType     // 数据处理函数
	AfterUpdatedFunc AfterUpdatedFunc // 更新完成后的处理

	timeOffset  int64
	processFunc func() error
	startTime   time.Time

	options *Options
}

func (l *LoadUpdatedData) GetTimeOffset() int64 {
	l.RLock()
	defer l.RUnlock()
	return l.timeOffset
}

func (l *LoadUpdatedData) SetTimeOffset(timeOffset int64) {
	l.Lock()
	defer l.Unlock()
	l.timeOffset = timeOffset
}

func NewLoadUpdateDataNormal(name, timeCycle string, updatedFunc UpdatedFuncTypeV2, opts ...Option) *LoadUpdatedData {
	o := &Options{}
	for _, op := range opts {
		op(o)
	}
	l := &LoadUpdatedData{
		DataMap:    sync.Map{},
		Name:       name,
		TimeCycle:  timeCycle,
		timeOffset: 0,
		options:    o,
	}
	updatedFuncV1 := func(startTimestamp int64) ([]interface{}, error) {
		structs, err := updatedFunc(startTimestamp)
		if err != nil {
			return nil, err
		}
		structList := make([]interface{}, 0, len(structs))
		for _, s := range structs {
			structList = append(structList, s)
		}
		return structList, nil
	}
	l.UpdatedFunc = updatedFuncV1
	l.DataProc = func(data interface{}, storeMap *sync.Map) (timestamp int64, err error) {
		uData, ok := data.(UpdatedStruct)
		if !ok {
			return 0, fmt.Errorf("data type err: %+v %T", data, data)
		}
		if l.options.dataProc != nil {
			err = l.options.dataProc(uData, storeMap)
			if err != nil {
				return 0, err
			}
		}

		return uData.GetMtime().Unix(), nil
	}
	l.processFunc = defaultProcFunc(l)
	return l
}

func NewLoadUpdatedData(name, timeCycle string, updatedFunc UpdatedFuncType, dataProc DataProcType) *LoadUpdatedData {
	l := &LoadUpdatedData{
		DataMap:     sync.Map{},
		Name:        name,
		TimeCycle:   timeCycle,
		UpdatedFunc: updatedFunc,
		DataProc:    dataProc,
		timeOffset:  0,
	}
	l.processFunc = defaultProcFunc(l)
	return l
}

func defaultProcFunc(l *LoadUpdatedData) func() error {
	return func() error {
		l.startTime = time.Now()
		timeOffset := l.GetTimeOffset()
		resList, err := l.UpdatedFunc(timeOffset)
		if err != nil {
			return LUDUpdatedErr
		}
		if len(resList) == 0 {
			return nil
		}
		// 全部更新处理完成才更新offset 有任何错误都会重新触发更新
		for _, res := range resList {
			timestamp, err := l.DataProc(res, &l.DataMap)
			if err != nil {
				return LUDProcErr
			}
			timeOffset = int64(math.Max(float64(timestamp), float64(timeOffset)))
		}
		if l.AfterUpdatedFunc != nil {
			// 事后处理 如果处理失败会重新触发更新
			err = l.AfterUpdatedFunc(resList, int64(len(resList)))
			if err != nil {
				return LUDAfterProcErr
			}
		}
		l.SetTimeOffset(timeOffset)
		return nil
	}
}

func (l *LoadUpdatedData) SetAfterUpdated(proc AfterUpdatedFunc) {
	l.AfterUpdatedFunc = proc
}

// Start 开始周期加载增量更新
func (l *LoadUpdatedData) Start() {
	util.StartInterval(func() {
		_ = l.processFunc
	}, l.TimeCycle)
}

// StartWithErr 开始周期加载增量更新 第一次执行如果失败会抛错
func (l *LoadUpdatedData) StartWithErr() error {
	if err := l.processFunc(); err != nil {
		return err
	}
	l.Start()
	return nil
}

// Do 立即执行一次
func (l *LoadUpdatedData) Do() error {
	return l.processFunc()
}

// ResetTimeOffset 重置增量更新的时间标志 下一次全量更新
func (l *LoadUpdatedData) ResetTimeOffset() {
	l.SetTimeOffset(0)
}

func (l *LoadUpdatedData) ResetTimeOffsetWithCron(spec string) {
	util.StartCron(func() {
		// 60s 内随机浮动 防止实例集中请求MySQL
		time.Sleep(time.Duration(int(time.Second) * rand.Intn(60)))
		l.ResetTimeOffset()
	}, spec)
}

// GetData 获取Map中存储的指定Data
func (l *LoadUpdatedData) GetData(key string) (data any, exist bool) {
	return l.DataMap.Load(key)
}
