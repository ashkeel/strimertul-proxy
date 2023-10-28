package strimertul_proxy

import (
	"context"

	"git.sr.ht/~hamcha/containers"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type clientList = containers.IDTable[Client]

func broadcast(c clientList, msg Message) {
	for _, client := range c {
		client.sendMessage(msg)
	}
}

type Client struct {
	ctx context.Context
	ws  *websocket.Conn
}

func (c *Client) sendMessage(msg Message) error {
	return wsjson.Write(c.ctx, c.ws, msg)
}
