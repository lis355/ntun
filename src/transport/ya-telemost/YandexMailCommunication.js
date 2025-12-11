import streamConsumers from "node:stream/consumers";

import { simpleParser } from "mailparser";
import Imap from "imap";
import nodemailer from "nodemailer";

import { createLog, ifLog, LOG_LEVELS } from "../../utils/log.js";
import symmetricBufferCipher from "../../utils/symmetricBufferCipher.js";

const log = createLog("[YandexMailCommunication]");

export default class YandexMailCommunication extends EventEmitter {
	constructor(options) {
		super();

		this.options = options || {};
		if (typeof this.options.user !== "string") throw new Error("Invalid user");
		if (typeof this.options.password !== "string") throw new Error("Invalid password");
		if (typeof this.options.opponent !== "string") throw new Error("Invalid opponent");

		this.handleImapOnReady = this.handleImapOnReady.bind(this);
		this.handleImapOnEnd = this.handleImapOnEnd.bind(this);
	}

	start() {
		this.imap = new Imap({
			user: this.options.user,
			password: this.options.password,
			host: "imap.yandex.ru",
			port: 993,
			tls: true
		})
			.on("ready", this.handleImapOnReady)
			.on("error", error => {
				if (ifLog(LOG_LEVELS.INFO)) log("imap error", error.message);

				this.abort(error);
			})
			.on("end", this.handleImapOnEnd);

		this.imap.connect();

		this.smtp = nodemailer.createTransport({
			host: "smtp.yandex.ru",
			port: 465,
			secure: true,
			auth: {
				user: this.options.user,
				pass: this.options.password
			}
		});
	}

	stop() {
		this.imap
			.off("ready", this.handleImapOnReady)
			.off("end", this.handleImapOnEnd);

		this.imap.end();

		this.imap = null;

		this.smtp.close();

		this.smtp = null;
	}

	abort(error) {
		this.emit("error", error);

		this.stop();
	}

	getEcryptKey() {
		// current date without time
		return symmetricBufferCipher.createChipherKeyFromString(new Date().toISOString().split("T")[0]);
	}

	encryptMessage(data) {
		return symmetricBufferCipher.encrypt(Buffer.from(JSON.stringify({ ...data, date: new Date().toISOString() })), this.getEcryptKey()).toString("hex");
	}

	decryptedMessage(encryptedMessage) {
		try {
			return JSON.parse(symmetricBufferCipher.decrypt(Buffer.from(encryptedMessage, "hex"), this.getEcryptKey()).toString());
		} catch (error) {
			if (ifLog(LOG_LEVELS.INFO)) log("error in decrypt message", error.message);

			this.abort(error);
		}
	}

	async sendMessage(text) {
		try {
			const info = await this.smtp.sendMail({
				from: this.options.user,
				to: this.options.opponent,
				text
			});

			if (ifLog(LOG_LEVELS.DETAILED)) log("smtp message sent", info.messageId);
		} catch (error) {
			if (ifLog(LOG_LEVELS.INFO)) log("error in send message", error.message);

			this.abort(error);
		}
	}

	handleImapOnReady() {
		this.emit("started");

		this.imap.openBox("INBOX", false, (error, box) => {
			if (error) throw error;

			this.imap.on("mail", newMessagesAmount => {
				this.processUnseenMessages();
			});

			this.processUnseenMessages();
		});
	}

	handleImapOnEnd() {
		this.emit("stopped");
	}

	processUnseenMessages() {
		this.imap.search(["UNSEEN"], (error, results) => {
			if (error) return this.abort(error);

			if (results.length === 0) return;

			const unseenFetchResult = this.imap.fetch(results[results.length - 1], { bodies: "" });
			unseenFetchResult
				.on("error", error => {
					this.abort(error);
				})
				.on("message", (message, seqno) => {
					message.on("body", async (stream, info) => {
						const body = await streamConsumers.text(stream);

						simpleParser(body, async (error, mail) => {
							if (error) return this.abort(error);

							const sender = mail.from.value[0].address;
							if (sender === this.options.opponent) {
								const text = mail.text.trim();

								this.imap.seq.addFlags(seqno, "\\Deleted");

								const message = this.decryptedMessage(text);
								if (message) this.emit("message", message);
							}
						});
					});
				});
		});
	}
}
