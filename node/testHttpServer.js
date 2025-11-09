import http from "node:http";

const port = Number(process.argv[2]);

http.createServer((req, res) => {
	res.writesHead(200, { "Content-Type": "text/plain" });
	res.end(req.socket.remoteAddress);
})
	.listen(port, () => {
		log(`testHttpServer started on http://localhost:${port}`);
	});
