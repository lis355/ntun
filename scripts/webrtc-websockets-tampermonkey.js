// ==UserScript==
// @name         webrtc-websockets-tampermonkey
// @namespace    http://tampermonkey.net/
// @version      2025-10-31
// @description  try to take over the world!
// @author       You
// @match        https://telemost.yandex.ru/*
// @match        https://vk.com/calls*
// @grant        none
// ==/UserScript==


// Monkeypatching WebRTC с ручным управлением DataChannel
(function () {
	'use strict';

	const OriginalWebSocket = window.WebSocket;

	window.WebSocket = function (url, protocols) {
		console.log('WebSocket connecting to:', url);

		const ws = new OriginalWebSocket(url, protocols);

		ws.addEventListener('open', (event) => {
			console.log('WebSocket connected');
		});

		ws.addEventListener('message', (event) => {
			console.log('WebSocket received:', event.data);
		});

		ws.addEventListener('error', (event) => {
			console.log('WebSocket error');
		});

		ws.addEventListener('close', (event) => {
			console.log('WebSocket closed:', event.code, event.reason);
		});

		const originalSend = ws.send.bind(ws);
		ws.send = function (data) {
			console.log('📤 WebSocket sending:', data);
			originalSend(data);
		};

		return ws;
	};

	window.WebSocket.prototype = OriginalWebSocket.prototype;
	window.WebSocket.CONNECTING = OriginalWebSocket.CONNECTING;
	window.WebSocket.OPEN = OriginalWebSocket.OPEN;
	window.WebSocket.CLOSING = OriginalWebSocket.CLOSING;
	window.WebSocket.CLOSED = OriginalWebSocket.CLOSED;

	const originalRTCPeerConnection = window.RTCPeerConnection;

	window.RTCPeerConnection = function (config) {
		console.log('ICE Servers:', JSON.stringify(config?.iceServers));

		const pc = new originalRTCPeerConnection(config);

		// const originalCreateOffer = pc.createOffer.bind(pc);
		// pc.createOffer = async function (options) {
		// 	const offer = await originalCreateOffer(options);
		// 	// console.log('SDP Offer:', offer.sdp);
		// 	return offer;
		// };

		return pc;
	};

	window.RTCPeerConnection.prototype = originalRTCPeerConnection.prototype;
})();