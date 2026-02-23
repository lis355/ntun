import { SocksProxyAgent } from "socks-proxy-agent";
import chalk from "chalk";
import fetch from "node-fetch";

import { createLog } from "../utils/log.js";
import exec from "./exec.js";

const log = createLog("[url-tests]");

export default async function urlTests(socks5InputConnectionPort) {
	await exec(`curl -s ${process.env.DEVELOP_GET_PUBLIC_IP_HTTP_URL}`);
	await exec(`curl -s ${process.env.DEVELOP_GET_PUBLIC_IP_HTTPS_URL}`);

	const { stdoutString: externalIp } = await exec(`curl -s ${process.env.DEVELOP_GET_PUBLIC_IP_HTTP_URL}`);

	await exec(`curl -s -x socks5://127.0.0.1:${socks5InputConnectionPort} ${process.env.DEVELOP_GET_PUBLIC_IP_HTTP_URL}`);
	await exec(`curl -s -x socks5://127.0.0.1:${socks5InputConnectionPort} ${process.env.DEVELOP_GET_PUBLIC_IP_HTTPS_URL}`);

	const urls = [
		process.env.DEVELOP_GET_PUBLIC_IP_HTTP_URL,
		process.env.DEVELOP_GET_PUBLIC_IP_HTTPS_URL,
		"https://api.ipify.org/?format=text",
		"https://checkip.amazonaws.com/",
		"https://icanhazip.com/"
	];

	const test = async () => {
		const start = performance.now();
		const time = () => (performance.now() - start) / 1000;

		const requests = urls.map(async url => {
			try {
				const proxy = `socks5://127.0.0.1:${socks5InputConnectionPort}`;
				// log(`${url} [${proxy}]`);

				const result = await fetch(url, { agent: new SocksProxyAgent(proxy) });
				const text = (await result.text()).trim().split("\n")[0];

				if (externalIp !== text) throw new Error(`Bad ip response, expected: ${externalIp}, actual: ${text}`);

				log(url, chalk.magenta(text), time().toFixed(2), "s");
			} catch (error) {
				log(url, chalk.red(error.message));
			}
		});

		await Promise.all(requests);
	};

	await test();
}
