package model_updater

import (
	"fmt"
	"github.com/sjmshsh/model_updater/util"
	"sync"
)

type LoadUpdatedVersionedData[T UpdaterWithVersion] struct {
	sync.RWMutex
	DataMap          *sync.Map                  // 存储数据用的map
	Name             string                     // 名称
	TimeCycle        string                     // 加载周期
	UpdatedFunc      VersionUpdatedFuncType[T]  // 获取增量更新的函数
	DataProcFunc     VersionDataProcType[T]     // 数据处理函数 如果处理保存，则对应数据及其版本跳过更新
	AfterUpdatedFunc VersionAfterUpdatedFunc[T] // 处理后批处理函数

	curVerHashes map[string]string
	processFunc  func() error
}

// NewVersionUpdater 新建一个基于版本号更新的更新器。需要指定获取更新的函数updatedFunc。
// 数据处理函数如不提供可选择
func NewVersionUpdater[T UpdaterWithVersion](name, timeCycle string, updatedFunc VersionUpdatedFuncType[T], dataProcFunc VersionDataProcType[T]) (*LoadUpdatedVersionedData[T], error) {
	if updatedFunc == nil {
		return nil, fmt.Errorf("NewVersionUpdater require a updatedFunc")
	}
	if dataProcFunc == nil {
		dataProcFunc = defaultVersionDataProc[T]
	}
	l := &LoadUpdatedVersionedData[T]{
		DataMap:      &sync.Map{},
		Name:         name,
		TimeCycle:    timeCycle,
		UpdatedFunc:  updatedFunc,
		DataProcFunc: dataProcFunc,

		curVerHashes: make(map[string]string),
	}

	l.processFunc = defaultProcessFunc(l)
	return l, nil
}

func defaultProcessFunc[T UpdaterWithVersion](l *LoadUpdatedVersionedData[T]) func() error {
	return func() error {
		l.RLock()
		curVerHashes := l.curVerHashes
		l.RUnlock()
		updatedList, newHashes, err := l.UpdatedFunc(curVerHashes)
		if err != nil {
			return LUDUpdatedErr
		}
		if len(updatedList) == 0 {
			return nil
		}

		for _, item := range updatedList {
			if dataHash, err := l.DataProcFunc(item, newHashes[item.GetName()], l.DataMap); err != nil {
				continue
			} else {
				l.Lock()
				l.curVerHashes[item.GetName()] = dataHash
				l.Unlock()
			}
		}

		return nil
	}
}

func defaultVersionDataProc[T UpdaterWithVersion](data T, verHash string, storeMap *sync.Map) (newVerHash string, err error) {
	dataHash := data.GetHash()
	if verHash != dataHash {
		// 版本号校验，如果数据版本不一致 跳过此次更新
		return "", LUDVersionCheckErr
	}

	storeMap.Store(data.GetName(), data)
	return dataHash, nil
}

func (l *LoadUpdatedVersionedData[T]) GetVal(name string) (val T, isExist bool) {
	value, ok := l.DataMap.Load(name)
	if !ok {
		return val, false
	}
	val, isExist = value.(T)
	return
}

// Start 开始周期加载增量更新
func (l *LoadUpdatedVersionedData[T]) Start() {
	util.StartInterval(func() {
		_ = l.processFunc()
	}, l.TimeCycle)
}

// StartWithErr 开始周期加载增量更新 第一次执行如果失败会抛错
func (l *LoadUpdatedVersionedData[T]) StartWithErr() error {
	if err := l.processFunc(); err != nil {
		return err
	}
	l.Start()
	return nil
}

// Do 立即执行一次
func (l *LoadUpdatedVersionedData[T]) Do() error {
	return l.processFunc()
}
