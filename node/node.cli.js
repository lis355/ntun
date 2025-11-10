import figlet from "figlet";
import parser from "yargs-parser";

import log from "./utils/log.js";
import ntun from "./ntun.js";

import info from "./package.json" with { type: "json" };

const argv = process.argv.slice(2);
const args = parser(argv, {
	alias: { input: "i", output: "o", transport: "t" },
	array: ["transport"]
});

function printLogo() {
	process.stdout.write(
		figlet.textSync(`${info.name} ${info.version}`, {
			font: "ANSI Shadow",
			whitespaceBreak: false
		}).trim()
	);
}

async function run() {
	printLogo();

	const node = new ntun.Node();

	if (!args.input && !args.output ||
		args.input && args.output) throw new Error("One of input or output must be specified");

	if (args.input) {
		if (!Number.isFinite(args.input) ||
			args.input < 0 ||
			args.input > 65535) throw new Error("Invalid input port");

		node.inputConnection = new ntun.inputConnections.Socks5InputConnection(node, { port: args.input });

		log("Input connection:", "Socks5InputConnection", "created");
	}

	if (args.output) {
		node.outputConnection = new ntun.outputConnections.InternetOutputConnection(node);

		log("Output connection:", "InternetOutputConnection", "created");
	}

	if (!args.transport ||
		args.transport.length === 0) throw new Error("Transport must be specified");

	switch (args.transport[0]) {
		case "tcp": {
			if (node.inputConnection) {
				try {
					let [host, port] = args.transport[1].split(":");
					port = Number(port);

					if (!Number.isFinite(port) ||
						port < 0 ||
						port > 65535) throw new Error("Invalid transport port");

					node.transport = new ntun.transports.TCPBufferSocketClientTransport(host, port);

					log("Transport:", "TCPBufferSocketClientTransport", "created");
				} catch (_) {
					throw new Error("Invalid transport URL");
				}
			} else if (node.outputConnection) {
				if (!Number.isFinite(args.transport[1]) ||
					args.transport[1] < 0 ||
					args.transport[1] > 65535) throw new Error("Invalid transport port");

				node.transport = new ntun.transports.TCPBufferSocketServerTransport(args.transport[1]);

				log("Transport:", "TCPBufferSocketServerTransport", "created");
			}

			break;
		}
		case "ws": {
			if (node.inputConnection) {
				let url;
				try {
					url = new URL(args.transport[1]);
				} catch (_) {
					throw new Error("Invalid transport URL");
				}

				node.transport = new ntun.transports.WebSocketBufferSocketClientTransport(url.hostname, url.port);

				log("Transport:", "WebSocketBufferSocketClientTransport", "created");
			} else if (node.outputConnection) {
				if (!Number.isFinite(args.transport[1]) ||
					args.transport[1] < 0 ||
					args.transport[1] > 65535) throw new Error("Invalid transport port");

				node.transport = new ntun.transports.WebSocketBufferSocketServerTransport(args.transport[1]);

				log("Transport:", "WebSocketBufferSocketServerTransport", "created");
			}

			break;
		}

		default:
			throw new Error("Invalid transport");
	}

	node.transport
		.on("connected", () => {
			log("Transport:", "connected");

			node.start();
		})
		.on("closed", () => {
			log("Transport:", "closed");

			node.stop();
		});

	node.transport.start();
}

run();
