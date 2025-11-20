const MAXIMUM_BUFFER_SIZE = 256 * 1024 ** 2; // 256 mB
const MAXIMUM_CHUNK_SIZE = 32 * 1024; // 32 kB

export default class BufferSocket {
	static STATE_READ_LENGTH = 0;
	static STATE_READ_BUFFER = 1;

	static enhanceSocket(socket, options) {
		const bufferSocket = new BufferSocket(options);
		bufferSocket.enhanceSocket(socket);

		return socket;
	}

	constructor(options) {
		this.options = options || {};
		this.maximumChunkSize = this.options.maximumChunkSize = this.options.maximumChunkSize || MAXIMUM_CHUNK_SIZE;

		this.sizeToRead = 4;
		this.state = BufferSocket.STATE_READ_LENGTH;
		this.chunks = [];
		this.chunksTotalSize = 0;

		this.processData = this.processData.bind(this);
	}

	enhanceSocket(socket) {
		this.socket = socket;

		this.socket.writeBuffer = this.writeBuffer.bind(this);

		this.socket.on("data", this.handleOnData.bind(this));
	}

	writeBuffer(buffer) {
		if (buffer.length > MAXIMUM_BUFFER_SIZE) throw new Error("Buffer too large");

		const lengthBuffer = Buffer.allocUnsafe(4);
		lengthBuffer.writeUInt32BE(buffer.length, 0);
		this.socket.write(lengthBuffer);

		if (buffer.length > this.maximumChunkSize) {
			for (let i = 0; i < buffer.length; i += this.maximumChunkSize) {
				this.socket.write(buffer.subarray(i, i + this.maximumChunkSize));
			}
		} else {
			this.socket.write(buffer);
		}
	}

	handleOnData(chunk) {
		this.chunks.push(chunk);
		this.chunksTotalSize += chunk.length;

		this.processData();
	}

	processData() {
		while (this.chunksTotalSize >= this.sizeToRead) {
			let chunksToReadAmount = 0;
			let chunksToReadSize = 0;
			while (chunksToReadSize < this.sizeToRead) chunksToReadSize += this.chunks[chunksToReadAmount++].length;

			if (chunksToReadAmount > 1) this.chunks.unshift(Buffer.concat(this.chunks.splice(0, chunksToReadAmount)));

			let chunk = this.chunks[0];
			let nextSizeToRead;
			switch (this.state) {
				case BufferSocket.STATE_READ_LENGTH:
					nextSizeToRead = chunk.readUInt32BE(0);
					this.state = BufferSocket.STATE_READ_BUFFER;
					break;

				case BufferSocket.STATE_READ_BUFFER:
					this.pushBuffer(chunk.length > this.sizeToRead ? chunk.subarray(0, this.sizeToRead) : chunk);

					nextSizeToRead = 4;
					this.state = BufferSocket.STATE_READ_LENGTH;
					break;
			}

			if (chunk.length > this.sizeToRead) {
				this.chunks[0] = chunk.subarray(this.sizeToRead);
			} else {
				this.chunks.shift();
			}

			this.chunksTotalSize -= this.sizeToRead;

			this.sizeToRead = nextSizeToRead;
		}
	}

	pushBuffer(buffer) {
		this.socket.emit("buffer", buffer);
	}
}
