import { config as dotenv } from "dotenv-flow";
import mri from "mri";
import wrtc from "wrtc";

import WebRTCPeer from "../browser/src/common/WebRTCPeer.js";

dotenv();

const args = mri(process.argv.slice(2));

const isDevelopment = Boolean(process.env.VSCODE_INJECTION &&
	process.env.VSCODE_INSPECTOR_OPTIONS);

function now() {
	return new Date().toISOString();
}

let iceServers;
if (isDevelopment) {
	iceServers = process.env.DEVELOP_WEB_RTC_SERVERS;
} else {
	if (args._.length < 1) throw new Error("Please provide ice servers configuration");

	iceServers = args._[0];
}

iceServers = JSON.parse(iceServers);

const webRTCPeerOptions = {
	iceServers,
	cancelGatheringCondition: peer => {
		return peer.iceCandidates.filter(iceCandidate => iceCandidate.type === "relay").length > 0;
	}
};

function createOfferPeer() {
	const offerPeer = new WebRTCPeer(wrtc.RTCPeerConnection, webRTCPeerOptions)
		.on("log", (...objs) => {
			const event = objs[0];
			if (event.startsWith("iceGathering")) return;

			if (event === "sendMessage" ||
				event === "handleMessage") {
				objs[1] = WebRTCPeer.arrayBufferToBuffer(objs[1]).toString();
			}

			console.log("OFFER LOG", ...objs);
		})
		.on("connected", () => {
			offerPeer.pingInterval = setInterval(() => {

				const messageToSend = "offer ping " + (offerPeer.counter = (offerPeer.counter || 0) + 1).toString() + " " + now();

				offerPeer.sendMessage(WebRTCPeer.bufferToArrayBuffer(Buffer.from(messageToSend)));
			}, 1000);
		})
		.on("disconnected", () => {
			offerPeer.pingInterval = clearInterval(offerPeer.pingInterval);
		})
		.on("message", message => {
			// console.log("offer handle message: ", WebRTCPeer.arrayBufferToBuffer(message).toString());
		});

	return offerPeer;
}

function createAnswerPeer() {
	const answerPeer = new WebRTCPeer(wrtc.RTCPeerConnection, webRTCPeerOptions)
		.on("log", (...objs) => {
			const event = objs[0];
			if (event.startsWith("iceGathering")) return;

			if (event === "sendMessage" ||
				event === "handleMessage") {
				objs[1] = WebRTCPeer.arrayBufferToBuffer(objs[1]).toString();
			}

			console.log("ANSWER LOG", ...objs);
		})
		.on("connected", () => {
		})
		.on("disconnected", () => {
		})
		.on("message", message => {
			// console.log("answer handle message: ", WebRTCPeer.arrayBufferToBuffer(message).toString());

			const messageToSend = "answer pong " + (answerPeer.counter = (answerPeer.counter || 0) + 1).toString() + " " + now();

			answerPeer.sendMessage(WebRTCPeer.bufferToArrayBuffer(Buffer.from(messageToSend)));
		});

	return answerPeer;
}

// const SIMPLE_SIGNAL_SERVER_PORT_URL = "http://localhost:8260";
const SIMPLE_SIGNAL_SERVER_PORT_URL = "http://jdam.am:8260";

async function main() {
	if (process.argv[2] === "offer") {
		console.log("mode offer");

		const offerPeer = createOfferPeer();

		const sdpOfferBase64 = await offerPeer.createOffer();

		await fetch(SIMPLE_SIGNAL_SERVER_PORT_URL + "/offer", {
			method: "POST",
			body: sdpOfferBase64
		});

		console.log("offer created");

		const waitForAnswer = async () => {
			const response = await fetch(SIMPLE_SIGNAL_SERVER_PORT_URL + "/answer", {
				method: "GET"
			});

			if (response.status === 200) {
				const sdpAnswerBase64 = await response.text();
				await offerPeer.setAnswer(sdpAnswerBase64);

				console.log("answer settled");
			} else {
				setTimeout(waitForAnswer, 1000);
			}
		};

		waitForAnswer();
	} else if (process.argv[2] === "answer") {
		console.log("mode answer");

		const answerPeer = createAnswerPeer();

		const waitForOffer = async () => {
			const response = await fetch(SIMPLE_SIGNAL_SERVER_PORT_URL + "/offer", {
				method: "GET"
			});

			if (response.status === 200) {
				const sdpOfferBase64 = await response.text();
				const sdpAnswerBase64 = await answerPeer.createAnswer(sdpOfferBase64);

				await fetch(SIMPLE_SIGNAL_SERVER_PORT_URL + "/answer", {
					method: "POST",
					body: sdpAnswerBase64
				});

				console.log("answer created");
			} else {
				setTimeout(waitForOffer, 1000);
			}
		};

		waitForOffer();
	} else {
		// console.log("mode simple test without signal server");

		// const offerPeer = createOfferPeer();
		// const answerPeer = createAnswerPeer();

		// const sdpOfferBase64 = await offerPeer.createOffer();
		// const sdpAnswerBase64 = await answerPeer.createAnswer(sdpOfferBase64);
		// await offerPeer.setAnswer(sdpAnswerBase64);

		console.log("mode simple test via signal server");

		const offerPeer = createOfferPeer();
		const answerPeer = createAnswerPeer();

		const sdpOffer = await offerPeer.createOffer();

		await fetch(SIMPLE_SIGNAL_SERVER_PORT_URL + "/offer", {
			method: "POST",
			body: sdpOffer
		});

		let response = await fetch(SIMPLE_SIGNAL_SERVER_PORT_URL + "/offer", {
			method: "GET"
		});

		let sdp = await response.text();
		await fetch(SIMPLE_SIGNAL_SERVER_PORT_URL + "/answer", {
			method: "POST",
			body: await answerPeer.createAnswer(sdp)
		});

		response = await fetch(SIMPLE_SIGNAL_SERVER_PORT_URL + "/answer", {
			method: "GET"
		});

		sdp = await response.text();
		await offerPeer.setAnswer(sdp);
	}
}

main();
