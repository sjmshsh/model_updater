package model_updater

import (
	"fmt"
	"sync"
	"time"
)

// v1

type UpdatedFuncType func(startTimestamp int64) ([]interface{}, error)                    // 获取增量更新的函数
type DataProcType func(data interface{}, storeMap *sync.Map) (timestamp int64, err error) // 数据处理函数
type AfterUpdatedFunc func(updatedList []interface{}, count int64) error                  // 更新完成后的处理

// v2

type UpdatedFuncTypeV2 func(startTimestamp int64) ([]UpdatedStruct, error)

// DataProcTypeV2 用于处理单条数据过程
type DataProcTypeV2 func(data UpdatedStruct, storeMap *sync.Map) (err error)

// 基于版本号更新的

// VersionUpdatedFuncType 根据版本号获取最新数据和对应版本号
type VersionUpdatedFuncType[T UpdaterWithVersion] func(curVerHashes map[string]string) (data map[string]T, newVerHashes map[string]string, err error)

// VersionDataProcType 如果有更新需要遍历处理
type VersionDataProcType[T UpdaterWithVersion] func(data T, verHash string, storeMap *sync.Map) (newVerHash string, err error)

// VersionAfterUpdatedFunc 批处理列表 并返回最新版本号
type VersionAfterUpdatedFunc[T UpdaterWithVersion] func(updatedList map[string]T, storeMap *sync.Map) (err error) // 更新完成后的处理

type UpdatedStruct interface {
	GetName() string
	GetMtime() time.Time
	GetIsDelete() int64
}

type UpdaterWithVersion interface {
	GetName() string            // 唯一标识符
	GenHash() string            // 更新Hash 并返回最新的
	GetHash() string            // 获取当前Hash 如果Hash不存在 会先更新
	Marshal() ([]byte, error)   // 序列化方法
	Unmarshal(src []byte) error // 反序列化方法
}

var (
	LUDUpdatedErr      = fmt.Errorf("LoadUpdatedData load update data err")
	LUDProcErr         = fmt.Errorf("LoadUpdatedData process data err")
	LUDAfterProcErr    = fmt.Errorf("LoadUpdatedData after process data err")
	LUDVersionCheckErr = fmt.Errorf("LoadUpdatedData version check with err")
)
