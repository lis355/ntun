import { config as dotenv } from "dotenv-flow";

import { setLogLevel, LOG_LEVELS } from "../../utils/log.js";
import exec from "../exec.js";
import ntun from "../../ntun.js";
import urlTests from "../urlTests.js";
import VkTransport from "../../transport/vk-calls/VkTransport.js";
import waits from "../waits.js";

dotenv();

setLogLevel(LOG_LEVELS.INFO);

async function run() {
	const joinId = VkTransport.getJoinId(process.env.DEVELOP_VK_JOIN_ID_OR_LINK);
	const socks5InputConnectionPort = 8080;

	const serverNode = new ntun.Node();
	serverNode.connection = new ntun.outputConnections.DirectOutputConnection(serverNode);
	serverNode.transport = new VkTransport.VkCallSignalServerTransport(joinId);

	const clientNode = new ntun.Node();
	clientNode.connection = new ntun.inputConnections.Socks5InputConnection(clientNode, { port: socks5InputConnectionPort });
	clientNode.transport = new VkTransport.VkCallSignalServerTransport(joinId);

	await Promise.all([
		new Promise(async resolve => {
			serverNode.start();
			clientNode.start();

			clientNode.transport.start();

			await new Promise(resolve => setTimeout(resolve, 3000));
			serverNode.transport.start();

			return resolve();
		}),
		waits.waitForStarted(serverNode),
		waits.waitForStarted(clientNode),
		waits.waitForConnected(serverNode.transport),
		waits.waitForConnected(clientNode.transport)
	]);

	// await urlTests(socks5InputConnectionPort);

	await exec(`curl -s -x socks5://127.0.0.1:${socks5InputConnectionPort} http://jdam.am:8302`);

	clientNode.transport.stop();
	await waits.waitForStopped(clientNode.transport);

	// await new Promise(resolve => setTimeout(resolve, 30000));

	clientNode.transport.start();

	await Promise.all([
		waits.waitForConnected(serverNode.transport),
		waits.waitForConnected(clientNode.transport)
	]);

	await new Promise(resolve => setTimeout(resolve, 3000));

	await exec(`curl -s -x socks5://127.0.0.1:${socks5InputConnectionPort} http://jdam.am:8302`);

	await Promise.all([
		new Promise(async resolve => {
			clientNode.transport.stop();
			serverNode.transport.stop();

			return resolve();
		}),
		waits.waitForStopped(serverNode.transport),
		waits.waitForStopped(clientNode.transport)
	]);

	await Promise.all([
		new Promise(async resolve => {
			serverNode.stop();
			clientNode.stop();

			return resolve();
		}),
		waits.waitForStopped(serverNode),
		waits.waitForStopped(clientNode)
	]);
}

run();
