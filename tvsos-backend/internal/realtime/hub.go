package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/kiritosuki/mover/internal/Logger"
	"go.uber.org/zap"
)

type Envelope struct {
	Event   string      `json:"event"`
	Ts      int64       `json:"ts"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	Send chan []byte
}

type Hub struct {
	mu sync.RWMutex

	vehicleClients map[*Client]struct{}
	statsClients   map[*Client]struct{}
	pathClients    map[uint]map[*Client]struct{}
}

var (
	globalHub *Hub
	once      sync.Once
)

func GlobalHub() *Hub {
	once.Do(func() {
		globalHub = &Hub{
			vehicleClients: make(map[*Client]struct{}),
			statsClients:   make(map[*Client]struct{}),
			pathClients:    make(map[uint]map[*Client]struct{}),
		}
	})
	return globalHub
}

func (h *Hub) NewVehicleClient() *Client {
	c := &Client{Send: make(chan []byte, 64)}
	h.mu.Lock()
	h.vehicleClients[c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *Hub) NewStatsClient() *Client {
	c := &Client{Send: make(chan []byte, 32)}
	h.mu.Lock()
	h.statsClients[c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *Hub) NewPathClient(vehicleID uint) *Client {
	c := &Client{Send: make(chan []byte, 64)}
	h.mu.Lock()
	if _, ok := h.pathClients[vehicleID]; !ok {
		h.pathClients[vehicleID] = make(map[*Client]struct{})
	}
	h.pathClients[vehicleID][c] = struct{}{}
	h.mu.Unlock()
	return c
}

func (h *Hub) RemoveVehicleClient(c *Client) {
	h.mu.Lock()
	delete(h.vehicleClients, c)
	h.mu.Unlock()
	close(c.Send)
}

func (h *Hub) RemoveStatsClient(c *Client) {
	h.mu.Lock()
	delete(h.statsClients, c)
	h.mu.Unlock()
	close(c.Send)
}

func (h *Hub) RemovePathClient(vehicleID uint, c *Client) {
	h.mu.Lock()
	if clients, ok := h.pathClients[vehicleID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.pathClients, vehicleID)
		}
	}
	h.mu.Unlock()
	close(c.Send)
}

func (h *Hub) StatsSubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.statsClients)
}

func (h *Hub) BroadcastVehicles(event string, payload interface{}) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.vehicleClients))
	for c := range h.vehicleClients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	h.broadcast(clients, event, payload)
}

func (h *Hub) BroadcastStats(event string, payload interface{}) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.statsClients))
	for c := range h.statsClients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	h.broadcast(clients, event, payload)
}

func (h *Hub) BroadcastPath(vehicleID uint, event string, payload interface{}) {
	h.mu.RLock()
	pathSet := h.pathClients[vehicleID]
	clients := make([]*Client, 0, len(pathSet))
	for c := range pathSet {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	h.broadcast(clients, event, payload)
}

func (h *Hub) SendClient(c *Client, event string, payload interface{}) {
	msg, err := json.Marshal(Envelope{Event: event, Ts: time.Now().UnixMilli(), Payload: payload})
	if err != nil {
		Logger.Logger.Error("序列化WebSocket消息失败", zap.Error(err), zap.String("event", event))
		return
	}
	select {
	case c.Send <- msg:
	default:
		Logger.Logger.Warn("客户端发送队列已满，丢弃单播消息", zap.String("event", event))
	}
}

func (h *Hub) broadcast(clients []*Client, event string, payload interface{}) {
	if len(clients) == 0 {
		return
	}
	msg, err := json.Marshal(Envelope{Event: event, Ts: time.Now().UnixMilli(), Payload: payload})
	if err != nil {
		Logger.Logger.Error("序列化WebSocket消息失败", zap.Error(err), zap.String("event", event))
		return
	}
	for _, c := range clients {
		select {
		case c.Send <- msg:
		default:
			Logger.Logger.Warn("客户端发送队列已满，丢弃广播消息", zap.String("event", event))
		}
	}
}
