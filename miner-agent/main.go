package main

import (
	ws "hos-device/miner-agent/srv/websocketservice"
	"hos/go-lib-websocket/wsc"
	"hos/go-lib-websocket/wss"
	"log"
	"net/url"
	"time"
)

func main() {
	log.Println("websocket start")

	wssInstance := wss.New(
		wss.WssInstance(wss.NewBasicWss),
		wss.Port(16688),
	)

	wscInstance := wsc.New(
		wsc.Url(&url.URL{
			Scheme: "ws",
			Host:   "xxxxxxxxxxxx",
			Path:   "/ws",
		}),
		wsc.WscInstance(wsc.NewBasicWsc),
		wsc.ReconnectInitialInterval(30*time.Second),
		wsc.ReconnectMaxInterval(1*time.Minute),
	)

	websocketservice := ws.NewWebsockerService(
		ws.HeartBeatPeriod(20*time.Second),
		ws.Wsc(wscInstance),
		ws.Wss(wssInstance),
	)
}
