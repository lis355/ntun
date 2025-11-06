import childProcess from "node:child_process";
import crypto from "node:crypto";
import EventEmitter from "node:events";
import net from "node:net";

import { SocksProxyAgent } from "socks-proxy-agent";
import * as ws from "ws";
import async from "async";
import fetch from "node-fetch";
import mri from "mri";
import msgpack from "msgpack5";
import socks from "socksv5";

import * as bufferSocket from "./bufferSocket.js";

const LOCALHOST = "127.0.0.1";

const packer = msgpack();

function objectToBuffer(obj) {
	return packer.encode(obj);
}

function bufferToObject(buffer) {
	return packer.decode(buffer);
}

function int32md5XorHash(str) {
	const hash = crypto.createHash("md5").update(str).digest();

	let result = 0;
	for (let i = 0; i < 16; i += 4) result ^= hash.readInt32BE(i);

	return result;
}

class Connection {
	constructor(node, options = {}) {
		this.node = node;
		this.options = options;
		this.connections = new Map();
	}

	async start() {
		this.connectionMultiplexer = new ConnectionMultiplexer(this.node.transport);

		this.connectionMultiplexer
			.on("connect", this.handleSocketMultiplexerOnConnect.bind(this))
			.on("close", this.handleSocketMultiplexerOnClose.bind(this))
			.on("data", this.handleSocketMultiplexerOnData.bind(this));
	}

	async stop() {
		for (const [connectionId, connection] of this.connections) {
			this.sendSocketMultiplexerClose(connectionId);

			connection.destroy();
		}

		this.connections.clear();

		this.connectionMultiplexer = null;
	}

	sendSocketMultiplexerConnect(connectionId, destinationHost, destinationPort) {
		this.connectionMultiplexer.sendMessageConnect(connectionId, destinationHost, destinationPort);
	}

	sendSocketMultiplexerClose(connectionId) {
		this.connectionMultiplexer.sendMessageClose(connectionId);
	}

	sendSocketMultiplexerData(connectionId, data) {
		this.connectionMultiplexer.sendMessageData(connectionId, data);
	}

	handleSocketMultiplexerOnConnect(connectionId, destinationHost, destinationPort) { }
	handleSocketMultiplexerOnClose(connectionId) { }
	handleSocketMultiplexerOnData(connectionId, data) { }
}

class ConnectionMultiplexer extends EventEmitter {
	static MESSAGE_TYPE_CONNECT = 0;
	static MESSAGE_TYPE_CLOSE = 1;
	static MESSAGE_TYPE_DATA = 2;

	constructor(transport) {
		super();

		this.transport = transport;

		this.transport.on("buffer", buffer => this.handleTransportOnBuffer(buffer));
	}

	sendMessageConnect(connectionId, destinationHost, destinationPort) {
		this.sendMessage(ConnectionMultiplexer.MESSAGE_TYPE_CONNECT, connectionId, destinationHost, destinationPort);
	}

	sendMessageClose(connectionId) {
		this.sendMessage(ConnectionMultiplexer.MESSAGE_TYPE_CLOSE, connectionId);
	}

	sendMessageData(connectionId, data) {
		this.sendMessage(ConnectionMultiplexer.MESSAGE_TYPE_DATA, connectionId, data);
	}

	sendMessage(type, connectionId, ...args) {
		// console.log("ConnectionMultiplexer", "send", "from", this.transport.localPort, "to", this.transport.remotePort, type, connectionId);

		const message = [type, connectionId, ...args];
		const buffer = objectToBuffer(message);

		this.transport.sendBuffer(buffer);
	}

	async handleTransportOnBuffer(buffer) {
		const message = bufferToObject(buffer);
		const [type, connectionId, ...args] = message;

		// console.log("ConnectionMultiplexer", "receive", "to", this.transport.localPort, "from", this.transport.remotePort, type, connectionId);

		switch (type) {
			case ConnectionMultiplexer.MESSAGE_TYPE_CONNECT: {
				const [destinationHost, destinationPort] = args;
				this.emit("connect", connectionId, destinationHost, destinationPort);

				break;
			}
			case ConnectionMultiplexer.MESSAGE_TYPE_CLOSE: {
				this.emit("close", connectionId);

				break;
			}
			case ConnectionMultiplexer.MESSAGE_TYPE_DATA: {
				const [data] = args;
				this.emit("data", connectionId, data);

				break;
			}
		}
	}
}

class InputConnection extends Connection {
}

class OutputConnection extends Connection {
}

class Node {
	constructor() {
		this.inputConnection = null;
		this.outputConnection = null;
		this.transport = null;
	}

	async start() {
		if (this.inputConnection) await this.inputConnection.start();
		if (this.outputConnection) await this.outputConnection.start();
	}

	async stop() {
		if (this.inputConnection) await this.inputConnection.stop();
		if (this.outputConnection) await this.outputConnection.stop();
	}
}

class Socks5InputConnection extends InputConnection {
	async start() {
		await super.start();

		this.server = socks.createServer(this.onSocksServerConnection.bind(this));
		this.server.useAuth(socks.auth.None());

		await new Promise(resolve => this.server.listen(this.options.port, LOCALHOST, resolve));

		console.log("Socks5InputConnection", "local socks proxy server started on", this.options.port, "port");
	}

	async stop() {
		await super.stop();

		this.server.close();
		this.server = null;

		console.log("Socks5InputConnection", "local socks proxy server stopped");
	}

	onSocksServerConnection(info, accept, deny) {
		console.log("Socks5InputConnection", `input socket ${info.srcAddr}:${info.srcPort} want connect to [${info.dstAddr}:${info.dstPort}]`);

		const socket = accept(true);
		// const connectionId = `${socket.localAddress}:${socket.localPort} <--> ${socket.remoteAddress}:${socket.remotePort}`;
		const connectionId = int32md5XorHash(socket.localAddress + socket.localPort + socket.remoteAddress + socket.remotePort);
		this.connections.set(connectionId, socket);

		this.sendSocketMultiplexerConnect(connectionId, info.dstAddr, info.dstPort);

		socket
			.on("data", data => {
				this.sendSocketMultiplexerData(connectionId, data);
			})
			.on("close", () => {
				this.connections.delete(connectionId);

				this.sendSocketMultiplexerClose(connectionId);
			});
	}

	handleSocketMultiplexerOnClose(connectionId) {
		const socket = this.connections.get(connectionId);

		socket.removeAllListeners("data");
		socket.removeAllListeners("close");

		socket.destroy();

		this.connections.delete(connectionId);
	}

	handleSocketMultiplexerOnData(connectionId, data) {
		const socket = this.connections.get(connectionId);
		socket.write(data);
	}
}

class InternetOutputConnection extends OutputConnection {
	async start() {
		await super.start();

		console.log("InternetOutputConnection", "started");
	}

	async stop() {
		await super.stop();

		console.log("InternetOutputConnection", "stopped");
	}

	handleSocketMultiplexerOnConnect(connectionId, destinationHost, destinationPort) {
		const socket = net.connect(destinationPort, destinationHost);
		this.connections.set(connectionId, socket);

		// we must wait until connect, before send any data to remoteSocket, create asyncQueue for each connection
		socket.asyncQueue = async.queue(async task => task());

		socket.asyncQueue.push(async () => {
			return new Promise((resolve, reject) => {
				socket
					.on("connect", () => {
						console.log("InternetOutputConnection", `connected with [${socket.remoteAddress}:${socket.remotePort}]`);

						return resolve();
					});
			});
		});

		socket
			.on("data", data => {
				this.sendSocketMultiplexerData(connectionId, data);
			})
			.on("close", () => {
				this.connections.delete(connectionId);

				this.sendSocketMultiplexerClose(connectionId);
			});
	}

	handleSocketMultiplexerOnClose(connectionId) {
		const socket = this.connections.get(connectionId);
		socket.asyncQueue.push(async () => {
			socket.removeAllListeners("data");
			socket.removeAllListeners("close");

			socket.destroy();

			this.connections.delete(connectionId);
		});
	}

	handleSocketMultiplexerOnData(connectionId, data) {
		const socket = this.connections.get(connectionId);
		if (!socket) console.log(data.toString("hex"));
		socket.asyncQueue.push(async () => {
			socket.write(data);
		});
	}
}

async function createTCPBufferSocketServerTransport(port) {
	return new Promise((resolve, reject) => {
		const server = net.createServer(socket => {
			if (server.socket) {
				// drop other connections
				socket.end();
				return;
			}

			socket = bufferSocket.enhanceSocket(socket);
			server.socket = socket;

			console.log("TCPBufferSocketServerTransport connected", socket.localAddress, socket.localPort, "<-->", socket.remoteAddress, socket.remotePort);

			socket
				.on("close", () => {
					server.close();

					console.log("TCPBufferSocketServerTransport closed", socket.localAddress, socket.localPort, "<-->", socket.remoteAddress, socket.remotePort);
				});

			return resolve(socket);
		});

		server.listen(port, LOCALHOST);
	});
}

async function createTCPBufferSocketClientTransport(host, port) {
	return new Promise((resolve, reject) => {
		const socket = bufferSocket.enhanceSocket(net.connect(port, host));
		socket
			.on("connect", () => {
				console.log("TCPBufferSocketClientTransport connected", socket.localAddress, socket.localPort, "<-->", socket.remoteAddress, socket.remotePort);

				return resolve(socket);
			})
			.on("close", () => {
				console.log("TCPBufferSocketClientTransport closed", socket.localAddress, socket.localPort, "<-->", socket.remoteAddress, socket.remotePort);
			});
	});
}

async function createWebSocketServerTransport(port) {
	return new Promise((resolve, reject) => {
		const webSocketServer = new ws.WebSocketServer({ host: LOCALHOST, port });

		webSocketServer
			.on("connection", webSocket => {
				if (webSocketServer.webSocket) {
					// drop other connections
					webSocket.end();
					return;
				}

				webSocketServer.webSocket = webSocket;

				console.log("WebSocketServerTransport connected", webSocket._socket.localAddress, webSocket._socket.localPort, "<-->", webSocket._socket.remoteAddress, webSocket._socket.remotePort);

				webSocket
					.on("close", () => {
						webSocketServer.close();

						console.log("WebSocketServerTransport closed", webSocket._socket.localAddress, webSocket._socket.localPort, "<-->", webSocket._socket.remoteAddress, webSocket._socket.remotePort);
					});

				webSocket.sendBuffer = buffer => webSocket.send(buffer);
				webSocket.end = () => webSocket.close();
				webSocket.on("message", message => webSocket.emit("buffer", message));

				return resolve(webSocket);
			});
	});
}

async function createWebSocketClientTransport(host, port) {
	return new Promise((resolve, reject) => {
		const webSocket = new ws.WebSocket(`ws://${host}:${port}`);

		webSocket
			.on("open", () => {
				console.log("WebSocketClientTransport connected", webSocket._socket.localAddress, webSocket._socket.localPort, "<-->", webSocket._socket.remoteAddress, webSocket._socket.remotePort);

				webSocket.sendBuffer = buffer => webSocket.send(buffer);
				webSocket.end = () => webSocket.close();
				webSocket.on("message", message => webSocket.emit("buffer", message));

				return resolve(webSocket);
			})
			.on("close", () => {
				console.log("WebSocketClientTransport closed", webSocket._socket.localAddress, webSocket._socket.localPort, "<-->", webSocket._socket.remoteAddress, webSocket._socket.remotePort);
			});
	});
}

async function run() {
	const args = mri(process.argv.slice(2), { alias: { i: "input", o: "output" } });
	console.log("cli args", args);

	const transportPort = 8013;
	const socks5InputConnectionPort = 8012;

	const [serverTransport, clientTransport] = await Promise.all([
		createTCPBufferSocketServerTransport(transportPort),
		createTCPBufferSocketClientTransport(LOCALHOST, transportPort)
	]);

	// const [serverTransport, clientTransport] = await Promise.all([
	// 	createWebSocketServerTransport(transportPort),
	// 	createWebSocketClientTransport(LOCALHOST, transportPort)
	// ]);

	async function createOutputNode() {
		const outputNode = new Node();
		outputNode.outputConnection = new InternetOutputConnection(outputNode);
		outputNode.transport = serverTransport;

		await outputNode.start();

		return outputNode;
	}

	async function createInputNode() {
		const inputNode = new Node();
		inputNode.inputConnection = new Socks5InputConnection(inputNode, { port: socks5InputConnectionPort });
		inputNode.transport = clientTransport;

		await inputNode.start();

		return inputNode;
	}

	const [outputNode, inputNode] = await Promise.all([
		createOutputNode(),
		createInputNode()
	]);

	async function curl(args) {
		return new Promise((resolve, reject) => {
			const child = childProcess.exec(`curl ${args}`);
			child.stdout
				.on("data", data => {
					console.log(data.toString());
				});

			child
				.on("error", error => {
					return reject(error);
				})
				.on("close", () => {
					return resolve();
				});
		});
	}

	await curl("https://jdam.am/api/ip");
	// await curl("-x socks5://127.0.0.1:8012 http://jdam.am:8260");
	// await curl("-x socks5://127.0.0.1:8012 https://jdam.am/api/ip");

	const urls = [
		// "http://jdam.am:8260",
		"https://jdam.am/api/ip",
		"https://api.ipify.org/?format=text",
		"http://jsonip.com/",
		"https://checkip.amazonaws.com/",
		"https://icanhazip.com/"
	];

	const test = async () => {
		console.log("Testing proxy multiplexing...");
		const start = performance.now();

		const requests = urls.map(async url => {
			try {
				const proxy = `socks5://127.0.0.1:${socks5InputConnectionPort}`;
				console.log(`${url} [${proxy}]`);

				const result = await fetch(url, { agent: new SocksProxyAgent(proxy) });
				const text = await result.text();

				console.log(url, text.split("\n")[0], (performance.now() - start) / 1000, "s");
			} catch (error) {
				console.log(error.message);
			}
		});

		await Promise.all(requests);

		console.log("Total time:", (performance.now() - start) / 1000, "s");
	};

	await test();

	await inputNode.stop();
	await outputNode.stop();

	serverTransport.end();
	clientTransport.end();
}

run();
