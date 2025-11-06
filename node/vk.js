import crypto from "node:crypto";

import { config as dotenv } from "dotenv-flow";
import * as ws from "ws";
import mri from "mri";

dotenv();

const args = mri(process.argv.slice(2));

const isDevelopment = Boolean(process.env.VSCODE_INJECTION &&
	process.env.VSCODE_INSPECTOR_OPTIONS);

async function getVkWebRTCServersByJoinIdOrLink(joinIdOrLink) {
	if (!joinIdOrLink) throw new Error("Join id or link is required");

	let joinId = joinIdOrLink;

	try {
		const url = new URL(joinIdOrLink);
		if (url.href.startsWith("https://vk.com/call/join/")) joinId = url.pathname.split("/").at(-1);
	} catch (_) {
	}

	const applicationKey = "CGMMEJLGDIHBABABA";
	const clientSecret = "QbYic1K3lEV5kTGiqlq2";
	const clientId = "6287487";
	const appId = "6287487";

	const deviceId = crypto.randomUUID();
	const username = "Anonym " + deviceId.slice(0, 4);

	async function postJson(url, params) {
		const response = await fetch(url, {
			method: "POST",
			body: new URLSearchParams(params)
		});

		const json = await response.json();
		// console.log("POST", url);
		// console.log(JSON.stringify(params, null, 2));
		// console.log(JSON.stringify(json, null, 2));

		return json;
	}

	const responses = {};

	responses["https://login.vk.com/?act=get_anonym_token__1"] = await postJson("https://login.vk.com/?act=get_anonym_token", {
		"client_secret": clientSecret,
		"client_id": clientId,
		"app_id": appId,
		"version": "1",

		"scopes": "audio_anonymous,video_anonymous,photos_anonymous,profile_anonymous",
		"isApiOauthAnonymEnabled": "false"
	});

	responses["https://api.vk.com/method/calls.getAnonymousAccessTokenPayload"] = await postJson("https://api.vk.com/method/calls.getAnonymousAccessTokenPayload?v=5.265&client_id=6287487", {
		"access_token": responses["https://login.vk.com/?act=get_anonym_token__1"]["data"]["access_token"]
	});

	responses["https://login.vk.com/?act=get_anonym_token__2"] = await postJson("https://login.vk.com/?act=get_anonym_token", {
		"client_secret": clientSecret,
		"client_id": clientId,
		"app_id": appId,
		"version": "1",

		"token_type": "messages",
		"payload": responses["https://api.vk.com/method/calls.getAnonymousAccessTokenPayload"]["response"]["payload"]
	});

	responses["https://api.vk.com/method/calls.getAnonymousToken"] = await postJson("https://api.vk.com/method/calls.getAnonymousToken?v=5.265&client_id=6287487", {
		"vk_join_link": "https://vk.com/call/join/" + joinId,
		"name": username,
		"access_token": responses["https://login.vk.com/?act=get_anonym_token__2"]["data"]["access_token"]
	});

	responses["https://calls.okcdn.ru/fb.do__auth.anonymLogin"] = await postJson("https://calls.okcdn.ru/fb.do", {
		"method": "auth.anonymLogin",
		"format": "JSON",
		"application_key": applicationKey,
		"session_data": JSON.stringify({
			"version": 2,
			"device_id": deviceId,
			"client_version": 1.1,
			"client_type": "SDK_JS"
		})
	});

	responses["https://calls.okcdn.ru/fb.do__vchat.joinConversationByLink"] = await postJson("https://calls.okcdn.ru/fb.do", {
		"method": "vchat.joinConversationByLink",
		"format": "JSON",
		"application_key": applicationKey,
		"session_key": responses["https://calls.okcdn.ru/fb.do__auth.anonymLogin"]["session_key"],
		"joinLink": joinId,
		"isVideo": false,
		"protocolVersion": 5,
		"anonymToken": responses["https://api.vk.com/method/calls.getAnonymousToken"]["response"]["token"]
	});

	return new Promise((resolve, reject) => {
		const webSocketUrl = responses["https://calls.okcdn.ru/fb.do__vchat.joinConversationByLink"]["endpoint"] + "&platform=WEB&appVersion=1.1&version=5&device=browser&capabilities=2F7F&clientType=VK&tgt=join";
		const webSocket = new ws.WebSocket(webSocketUrl);
		webSocket
			.on("open", () => {
			})
			.on("close", () => {
				return resolve();
			})
			.on("error", error => {
				webSocket.close();

				return reject(error);
			})
			.on("message", message => {
				try {
					const json = JSON.parse(message);
					webSocket.close();

					return resolve([json.conversationParams.turn]);
				} catch (error) {
					return reject(error);
				}

			});
	});
}

let joinIdOrLink;
if (isDevelopment) {
	joinIdOrLink = process.env.DEVELOP_VK_JOIN_ID_OR_LINK;
} else {
	if (args._.length < 1) throw new Error("Please provide a join id or link");

	joinIdOrLink = args._[0];
}

const servers = await getVkWebRTCServersByJoinIdOrLink(joinIdOrLink);
console.log(JSON.stringify(servers));
