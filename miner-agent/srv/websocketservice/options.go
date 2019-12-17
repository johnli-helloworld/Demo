package websocketservice

import (
	"hos/go-lib-websocket/wsc"
	"hos/go-lib-websocket/wss"
	"time"
)

type Options struct {
	HeartBeatPeriod time.Duration
	PostTimeOut     time.Duration
	Wsc             wsc.Wsc
	Wss             wss.Wss
}

type Option func(o *Options)

func newOptions(opts ...Option) Options {
	options := Options{
		HeartBeatPeriod: 20 * time.Second,
		PostTimeOut:     10 * time.Second,
		Wsc:             nil,
		Wss:             nil,
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}

func HeartBeatPeriod(hbp time.Duration) Option {
	return func(options *Options) {
		options.HeartBeatPeriod = hbp
	}
}

func PostTimeOut(timeout time.Duration) Option {
	return func(options *Options) {
		options.PostTimeOut = timeout
	}
}

func Wsc(wsc wsc.Wsc) Option {
	return func(options *Options) {
		options.Wsc = wsc
	}
}

func Wss(wss wss.Wss) Option {
	return func(options *Options) {
		options.Wss = wss
	}
}
