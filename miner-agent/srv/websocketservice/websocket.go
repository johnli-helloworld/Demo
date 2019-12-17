package websocketservice

import (
	"hos/go-lib-websocket/wss"
)

type WebsocketService struct {
	opts          Options
	heartBeatDone chan struct{}
}

func (w *WebsocketService) Init() {
	w.opts.Wss.SetServeHandler(w.WssServe)
}

func (w *WebsocketService) WssServe(c wss.Client, data []byte) []byte {
	return nil
}

func NewWebsockerService(opts ...Option) *WebsocketService {
	options := newOptions(opts...)
	return &WebsocketService{
		opts:          options,
		heartBeatDone: make(chan struct{}),
	}
}
