RPC API
======
Start service with configuration file that enables API:

```
./bin/status-term-client -node-config ./_assets/api.conf -no-ui --keyhex=0x123
```

Configuration example:

```json
{
  "HTTPHost": "localhost",
  "HTTPPort": 8777,
  "HTTPEnabled": true,
  "APIModules": "ssm"
}
```

Add contact
===========

```
curl -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"ssm_addContact","params":[{"name": "222"}],"id":1}' http://localhost:8777
```


Returns error if failed otherwise null.

Send to contact
===============

```
curl -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"ssm_sendToContact","params":[{"name": "222"}, "plain text"],"id":1}' http://localhost:8777
```

Returns error if failed otherwise null.

Read all (including ours) from contact
======================================


To read all messages use 0 as offset:

```
curl -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"ssm_readContactMessages","params":[{"name": "222"}, 0],"id":1}' http://localhost:8777
```

To read new messages use last known offset:

```
curl -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"ssm_readContactMessages","params":[{"name": "222"}, 2],"id":1}' http://localhost:8777
```
