# Eywa-relayer

This is a relayer for Eywa. It is a standalone application that listens to Eywa-clients relays between them for fast and secure communication. Also, the main feature of this eywa-relayer is to copy and broadcast messages to Cosmos network with transactions.

### Using Docker

You must set docker environment variables before running eywa-relayer. Which is `ACCOUNT_NAME` and `NODE_ADDRESS`. recommended values are `neytiri` and `http://localhost:26657` respectively.

```bash
docker pull jaehong21/eywa-relayer:latest
```
