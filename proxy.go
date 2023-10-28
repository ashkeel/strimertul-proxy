package strimertul_proxy

import (
	"context"
	"log"
	"net/http"
	"time"

	"nhooyr.io/websocket"

	"nhooyr.io/websocket/wsjson"

	"git.sr.ht/~hamcha/containers/sync"
)

type Proxy struct {
	auth    map[string]string
	mux     *http.ServeMux
	clients *sync.Map[string, clientList]
	hosts   *sync.Map[string, clientList]
}

func NewProxy(auth map[string]string) *Proxy {
	proxy := &Proxy{
		auth:    auth,
		clients: sync.NewMap[string, clientList](),
		hosts:   sync.NewMap[string, clientList](),
	}
	for host := range proxy.auth {
		proxy.clients.SetKey(host, make(clientList))
		proxy.hosts.SetKey(host, make(clientList))
	}

	proxy.mux = http.NewServeMux()
	proxy.mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	proxy.mux.HandleFunc("/client/", proxy.clientWS)
	proxy.mux.HandleFunc("/host/", proxy.hostWS)
	return proxy
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	p.mux.ServeHTTP(w, req)
}

// Used by webclients
func (p *Proxy) clientWS(w http.ResponseWriter, req *http.Request) {
	// Check URL for host name
	host := getChannel(req.URL.Path, "/client/")
	if _, ok := p.auth[host]; !ok {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	}

	// Accept connection
	c, err := websocket.Accept(w, req, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}
	defer c.CloseNow()

	ctx, cancel := context.WithTimeout(req.Context(), time.Second*10)
	defer cancel()
	client := Client{ctx, c}

	// Register client
	var id uint64
	p.clients.BlockingSetKey(host, func(list clientList) clientList {
		id = list.Assign(client)
		return list
	})

	// Remove client from client list
	defer p.clients.BlockingSetKey(host, func(list clientList) clientList {
		list.Delete(id)
		return list
	})

	// Send host status as handshake
	_ = p.sendHostStatus(client, host)

	// Read messages
	for {
		var v any
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			break
		}
		hosts, _ := p.hosts.GetKey(host)
		broadcast(hosts, Message{"ClientMessage", v})
	}

	_ = c.Close(websocket.StatusNormalClosure, "")
}

// Used by the strimertul plugin
func (p *Proxy) hostWS(w http.ResponseWriter, req *http.Request) {
	// Check URL for host name
	host := getChannel(req.URL.Path, "/host/")
	auth, ok := p.auth[host]
	if !ok {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	}

	// Accept connection
	c, err := websocket.Accept(w, req, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusNotAcceptable)
		return
	}
	defer c.CloseNow()

	ctx, cancel := context.WithTimeout(req.Context(), time.Second*10)
	defer cancel()
	client := Client{ctx, c}

	// Wait for auth
	var authReq AuthRequest
	err = wsjson.Read(ctx, c, &authReq)
	if err != nil || authReq.Password != auth {
		_ = c.Write(ctx, websocket.MessageText, []byte("Authentication failed"))
		return
	}

	// Register client
	var id uint64
	p.hosts.BlockingSetKey(host, func(list clientList) clientList {
		id = list.Assign(client)
		return list
	})
	defer func() {
		p.hosts.BlockingSetKey(host, func(list clientList) clientList {
			list.Delete(id)
			return list
		})
		p.broadcastHostStatus(host)
	}()

	// Broadcast status change to everyone
	p.broadcastHostStatus(host)

	// Read messages
	for {
		var v any
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			break
		}
		clients, _ := p.clients.GetKey(host)
		broadcast(clients, Message{"HostMessage", v})
	}

	_ = c.Close(websocket.StatusNormalClosure, "")
}

type AuthRequest struct {
	Password string `json:"password"`
}

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type HostStatus struct {
	Connected bool `json:"connected"`
}

func (p *Proxy) sendHostStatus(client Client, host string) error {
	// Check if host is connected
	hosts, found := p.hosts.GetKey(host)

	return client.sendMessage(Message{"HostStatus", HostStatus{found && len(hosts) > 0}})
}

func (p *Proxy) broadcastHostStatus(host string) {
	// Check if host is connected
	hosts, found := p.hosts.GetKey(host)

	// Get clients
	clients, _ := p.clients.GetKey(host)

	broadcast(clients, Message{"HostStatus", HostStatus{found && len(hosts) > 0}})
}
