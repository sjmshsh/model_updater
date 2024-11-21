package model_updater

type Options struct {
	dataProc DataProcTypeV2
}

type Option func(options *Options)

func SetDataProc(dataProc DataProcTypeV2) Option {
	if dataProc == nil {
		panic("model updater setDataProc with nil")
	}
	return func(o *Options) {
		o.dataProc = dataProc
	}
}
