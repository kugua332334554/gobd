package layerx

import "errors"

var (
	// ErrLayer 未知层
	ErrLayer = errors.New("layerx: unknown layer")
	// ErrAbsent 层内不存在
	ErrAbsent = errors.New("layerx: absent in layer")
	// ErrNoPlan 无计划
	ErrNoPlan = errors.New("layerx: no plan")
	// ErrHook 待决策
	ErrHook = errors.New("layerx: needs decision")
	// ErrMismatch 类型不符
	ErrMismatch = errors.New("layerx: type mismatch")
	// ErrNil 空值
	ErrNil = errors.New("layerx: nil value")
	// ErrStale 过期
	ErrStale = errors.New("layerx: stale value")
	// ErrNoZip 未开解压
	ErrNoZip = errors.New("layerx: gzip disabled")
	// ErrOdd 未知方法
	ErrOdd = errors.New("layerx: unknown method")
	// ErrLimit 超限
	ErrLimit = errors.New("layerx: limit exceeded")
)
