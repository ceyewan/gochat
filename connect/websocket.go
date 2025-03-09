package connect

import (
	"gochat/clog"
	"gochat/config"
	"net/http"

	"github.com/gorilla/websocket"
)

func InitWebSocket() error {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(DefaultServer, w, r)
	})
	err := http.ListenAndServe(config.Conf.Connect.ConnectWebsocket.Bind, nil)
	return err
}

func serveWs(server *Server, w http.ResponseWriter, r *http.Request) {
	var upGrader = websocket.Upgrader{
		ReadBufferSize:  server.Options.ReadBufferSize,
		WriteBufferSize: server.Options.WriteBufferSize,
	}
	//cross origin domain support
	upGrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upGrader.Upgrade(w, r, nil)

	if err != nil {
		clog.Error("serverWs err:%s", err.Error())
		return
	}
	//default broadcast size eq 512
	ch := NewChannel(server.Options.BroadcastSize)
	ch.conn = conn
	//send data to websocket conn
	go server.writePump(ch, c)
	//get data from websocket conn
	go server.readPump(ch, c)
}
