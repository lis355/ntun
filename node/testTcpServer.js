import net from "node:net";

const port = Number(process.argv[2]);

net.createServer()
	.on("connection", socket => {
		socket.write(socket.remoteAddress);
		socket.end();
	})
	.listen(port, "127.0.0.1", () => {
		console.log(`testTcpServer started on port ${port}`);
	});
