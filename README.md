# Web proxy for strimertul

Self-hostable endpoint for allowing webpage interactions with strimertul clients. There is no replication/sync logic, it just forwards messages between clients and host.

To set a channel, add a `channelname:hostpassword` pair to the env var `AUTH` (comma-delimited for multiple channels).

To connect from strimertul, create an extension with this code:
```ts
// ==Extension==
// @name        Web proxy
// @version     1.0
// @author      Ash Keel
// @description Use a self-hosted proxy to integrate client pages with strimertul
// @apiversion  3.1.0
// ==/Extension==

const https = true;
const host = "your.ws.endpoint";
const channel = "channelname";
const password = "hostpassword";

var ws = new WebSocket(`${https ? "wss" : "ws"}://${host}/host/${channel}`);
ws.addEventListener("open", () => {
    console.log("Connected to proxy");
    ws.send(JSON.stringify({ password }));
});
ws.addEventListener("message", (event: MessageEvent) => {
    // event.data is your client's message
});
```

The entire project is licensed under [AGPL-3.0-only](LICENSE) (see `LICENSE`).