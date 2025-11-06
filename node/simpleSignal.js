import express from "express";

const app = express();
app.use(express.text());

let currentOffer = "";
let currentAnswer = "";

app.use((req, res, next) => {
	console.log(`[${new Date().toISOString()}]: ${req.method} ${req.url}`);

	return next();
});

app.use((req, res, next) => {
	res.header("Access-Control-Allow-Origin", "*");
	res.header("Access-Control-Allow-Methods", "GET, POST");
	res.header("Access-Control-Allow-Headers", "Content-Type");

	return next();
});

app.get("/offer", (req, res) => {
	if (currentOffer) {
		console.log("offer unsettled");

		const sdp = currentOffer;

		currentOffer = "";
		currentAnswer = "";

		return res.send(sdp);
	}

	return res.sendStatus(404);
});

app.post("/offer", (req, res) => {
	currentOffer = req.body;

	console.log("offer settled");

	return res.sendStatus(200);
});

app.get("/answer", (req, res) => {
	if (currentAnswer) {
		console.log("answer unsettled");

		const sdp = currentAnswer;

		currentOffer = "";
		currentAnswer = "";

		return res.send(sdp);
	}

	return res.sendStatus(404);
});

app.post("/answer", (req, res) => {
	currentAnswer = req.body;

	console.log("answer settled");

	return res.sendStatus(200);
});

app.options(/(.*)/, (req, res) => {
	return res.sendStatus(200);
});

const port = 8260;
app.listen(port, () => {
	console.log(`Simple signal server listening on port ${port}`);
});
