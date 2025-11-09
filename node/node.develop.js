import childProcess from "node:child_process";

import { config as dotenv } from "dotenv-flow";
import { SocksProxyAgent } from "socks-proxy-agent";
import fetch from "node-fetch";

import ntun from "./ntun.js";

dotenv();

async function run() {
	const transportPort = 8081;
	const transportHost = "127.0.0.1";

	const socks5InputConnectionPort = 8080;

	const serverTransport = new ntun.transports.TCPBufferSocketServerTransport(transportPort);
	// serverTransport
	// 	.on("connected", () => {
	// 		console.log("serverTransport", "connected");
	// 	})
	// 	.on("closed", () => {
	// 		console.log("serverTransport", "closed");
	// 	});

	// serverTransport.start();

	const clientTransport = new ntun.transports.TCPBufferSocketClientTransport(transportHost, transportPort);
	// clientTransport
	// 	.on("connected", () => {
	// 		console.log("clientTransport", "connected");
	// 	})
	// 	.on("closed", () => {
	// 		console.log("clientTransport", "closed");
	// 	});

	clientTransport.start();

	await Promise.all([
		// new Promise(resolve => serverTransport.once("connected", resolve)),
		new Promise(resolve => clientTransport.once("connected", resolve))
	]);

	async function createOutputNode() {
		const outputNode = new ntun.Node();
		outputNode.outputConnection = new ntun.outputConnections.InternetOutputConnection(outputNode);
		outputNode.transport = serverTransport;

		await outputNode.start();

		return outputNode;
	}

	async function createInputNode() {
		const inputNode = new ntun.Node();
		inputNode.inputConnection = new ntun.inputConnections.Socks5InputConnection(inputNode, { port: socks5InputConnectionPort });
		inputNode.transport = clientTransport;

		await inputNode.start();

		return inputNode;
	}

	const [outputNode, inputNode] = await Promise.all([
		// createOutputNode(),
		createInputNode()
	]);

	async function exec(str) {
		return new Promise((resolve, reject) => {
			const child = childProcess.exec(str);
			child.stdout
				.on("data", data => {
					data.toString().split("\n").filter(Boolean).forEach(line => {
						console.log("[" + str + "]", line.toString().trim());
					});
				});

			child.stderr
				.on("data", data => {
					data.toString().split("\n").filter(Boolean).forEach(line => {
						console.error("[" + str + "]", line.toString().trim());
					});
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

	await exec("curl -s https://jdam.am/api/ip");

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

	await exec(`curl -s -x socks5://127.0.0.1:${socks5InputConnectionPort} https://jdam.am/api/ip`);

	await inputNode.stop();
	await outputNode.stop();

	serverTransport.stop();
	clientTransport.stop();
}

run();
