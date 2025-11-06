import { Component } from "react";

import WebRTCPeer from "./common/WebRTCPeer.js";

function now() {
	return new Date().toISOString();
}

async function writeClipboardText(text) {
	await navigator.clipboard.writeText(text);
}

async function readClipboardText() {
	try {
		return await navigator.clipboard.readText();
	} catch (error) {
		console.error(error);

		return "";
	}
}

class App extends Component {
	constructor(props) {
		super(props);

		this.state = {
			logs: [],

			clientType: "",
			webRTCConnected: false,
			webSocketConnected: false,

			inputOptionLocalWebSocket: this.optionLocalWebSocket,
			inputOptionIceServers: this.optionIceServers,

			inputSdpMessage: "",
		};

		this.peerConnection = null;
	}

	componentDidMount() {
	}

	componentWillUnmount() {
		this.pingInterval = clearInterval(this.pingInterval);
	}

	get optionIceServers() {
		return localStorage.getItem("iceServers") || "";
	}

	set optionIceServers(value) {
		localStorage.setItem("iceServers", value || "");
	}

	get optionLocalWebSocket() {
		return localStorage.getItem("localWebSocket") || "ws://localhost:8050";
	}

	set optionLocalWebSocket(value) {
		localStorage.setItem("localWebSocket", value || "");
	}

	log(...objs) {
		this.setState(prevState => ({
			logs: [...prevState.logs, [now(), ...objs.map(String)].join(" ")].slice(-20),
		}));
	}

	createPeer() {
		this.peer = new WebRTCPeer(RTCPeerConnection, {
			iceServers: JSON.parse(this.optionIceServers)
		});

		this.peer.initialize();

		this.peer.on("log", (...objs) => {
			if (objs[0] === "sendMessage" ||
				objs[0] === "handleMessage") {
				objs[1] = new TextDecoder().decode(objs[1]);
			}
			this.log(...objs);
		});

		this.peer
			.on("connected", () => {
				this.setState({ webRTCConnected: true });

				if (this.state.clientType === "offer") {
					this.pingInterval = setInterval(() => {
						this.peer.sendMessage(new TextEncoder().encode("offer ping " + now()).buffer);
					}, 1000);
				}
			})
			.on("disconnected", () => {
				this.setState({ webRTCConnected: false });

				this.pingInterval = clearInterval(this.pingInterval);
			})
			.on("message", message => {
				if (this.state.clientType === "answer") {
					this.peer.sendMessage(new TextEncoder().encode("answer pong " + now()).buffer);
				}

				if (this.state.webRTCConnected &&
					this.state.webSocketConnected) {
					this.ws.send(message);
				}
			});

		this.log("peer created");
	}

	destroyPeer() {
		this.peer.destroy();
		this.peer = null;

		this.log("peer destroyed");

		this.ws.close();
		this.ws = null;

		this.log("ws closed");
	}

	createWs() {

		this.ws = new WebSocket(this.optionLocalWebSocket);

		this.ws.onopen = () => {
			this.setState({ webSocketConnected: true });

			this.log("ws connected to server");
		};

		this.ws.onmessage = event => {
			this.log("ws received", event.data);

			if (this.state.webRTCConnected &&
				this.state.webSocketConnected) {
				this.peer.sendMessage(event.data);
			}
		};

		this.ws.onerror = error => {
			this.log("ws error", error);
		};

		this.ws.onclose = () => {
			this.setState({ webSocketConnected: false });

			this.log("ws closed");
		};

		this.log("ws opened");
	}

	render() {
		return (
			<div className="app">
				<div>
					<h2>WebRTC Ping-Pong [{this.state.clientType}]</h2>

					<p>Local WebSocket</p>
					<textarea
						name="optionLocalWebSocket"
						value={this.state.inputOptionLocalWebSocket}
						onChange={event => {
							this.optionLocalWebSocket = event.target.value;
							this.setState({ inputOptionLocalWebSocket: this.optionLocalWebSocket });
						}}
						rows="1"
						disabled={this.peerConnection}
					/>
					<button onClick={async () => {
						this.optionLocalWebSocket = await readClipboardText();
						this.setState({ inputOptionLocalWebSocket: this.optionLocalWebSocket });
					}}>Paste</button>

					<p>ICE Servers</p>
					<textarea
						name="optionIceServers"
						value={this.state.inputOptionIceServers}
						onChange={event => {
							this.optionIceServers = event.target.value;
							this.setState({ inputOptionIceServers: this.optionIceServers });
						}}
						rows="12"
					/>
					<button onClick={async () => {
						this.optionIceServers = await readClipboardText();
						this.setState({ inputOptionIceServers: this.optionIceServers });
					}}>Paste</button>

					<p><button onClick={() => this.createPeer()} disabled={this.peer || !this.optionLocalWebSocket || !this.optionIceServers}>Start WebRTC</button></p>
					<p><button onClick={() => this.destroyPeer()} disabled={!this.peer}>Stop WebRTC</button></p>

					<textarea
						name="inputSdpMessage"
						value={this.state.inputSdpMessage}
						onChange={event => this.setState({ inputSdpMessage: event.target.value })}
						rows="6"
					/>
					<button disabled={!this.state.inputSdpMessage} onClick={async () => await writeClipboardText(this.state.inputSdpMessage)}>Copy</button>
					<button disabled={!this.peer} onClick={async () => {
						this.setState({ inputSdpMessage: await readClipboardText() });
					}}>Paste</button>

					<p>
						<button onClick={() => {
							this.setState({ clientType: "offer" }, async () => {
								const sdpOfferBase64 = await this.peer.createOffer();
								// await writeClipboardText(sdpOfferBase64);
								this.setState({ inputSdpMessage: sdpOfferBase64 });
								this.log("sdp offer created and copied to clipboard");
							});
						}} disabled={!this.peer}>Create Offer</button>
						<button onClick={() => {
							this.setState({ clientType: "answer" }, async () => {
								const sdpOfferBase64 = this.state.inputSdpMessage;
								this.setState({ inputSdpMessage: "" });
								const sdpAnswerBase64 = await this.peer.createAnswer(sdpOfferBase64);
								// await writeClipboardText(sdpAnswerBase64);
								this.setState({ inputSdpMessage: sdpAnswerBase64 });
								this.log("sdp answer created and copied to clipboard");
							});
						}} disabled={!this.peer}>Create Answer</button>
						<button onClick={async () => {
							const sdpAnswerBase64 = this.state.inputSdpMessage;
							this.setState({ inputSdpMessage: "" });
							await this.peer.setAnswer(sdpAnswerBase64);
							this.log("sdp answer set");
						}} disabled={!this.peer}>Set Answer</button>
					</p>

					<p>Log</p>
					<div>
						{this.state.logs.map((entry, index) => (
							<div key={index}>{entry}</div>
						))}
					</div>
				</div >
			</div>
		);
	}
}

export default App;
