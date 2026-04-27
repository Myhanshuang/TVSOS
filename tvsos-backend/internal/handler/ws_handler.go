package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/realtime"
	"github.com/kiritosuki/mover/internal/repository"
	"github.com/kiritosuki/mover/internal/task"
	"go.uber.org/zap"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

func WsVehicles(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		Logger.Logger.Error("升级车辆WebSocket连接失败", zap.Error(err))
		return
	}

	hub := realtime.GlobalHub()
	client := hub.NewVehicleClient()
	defer hub.RemoveVehicleClient(client)
	defer conn.Close()

	vehicles, err := repository.ListVehicles(0)
	if err == nil {
		hub.SendClient(client, "snapshot", map[string]interface{}{"vehicles": repository.BuildVehicleViews(vehicles)})
	}

	runWSLoop(conn, client)
}

func WsVehiclePath(c *gin.Context) {
	vehicleID, err := strconv.Atoi(c.Param("vehicleId"))
	if err != nil || vehicleID <= 0 {
		c.Status(http.StatusBadRequest)
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		Logger.Logger.Error("升级路径WebSocket连接失败", zap.Error(err))
		return
	}

	hub := realtime.GlobalHub()
	client := hub.NewPathClient(uint(vehicleID))
	defer hub.RemovePathClient(uint(vehicleID), client)
	defer conn.Close()

	if snapshot, ok := task.GetVehicleRouteSnapshot(uint(vehicleID)); ok {
		hub.SendClient(client, "full_path", snapshot)
	}

	runWSLoop(conn, client)
}

func WsStats(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		Logger.Logger.Error("升级统计WebSocket连接失败", zap.Error(err))
		return
	}

	hub := realtime.GlobalHub()
	client := hub.NewStatsClient()
	defer hub.RemoveStatsClient(client)
	defer conn.Close()

	snapshot, err := task.BuildRealtimeDashboard()
	if err == nil {
		hub.SendClient(client, "snapshot", snapshot)
	}

	runWSLoop(conn, client)
}

func runWSLoop(conn *websocket.Conn, client *realtime.Client) {
	readDone := make(chan struct{})
	writeDone := make(chan struct{})

	go func() {
		defer close(readDone)
		conn.SetReadLimit(1024)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	go func() {
		defer close(writeDone)
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-client.Send:
				if !ok {
					_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	select {
	case <-readDone:
	case <-writeDone:
	}
}
