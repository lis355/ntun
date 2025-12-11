import crypto from "node:crypto";

import hash from "./hash.js";

import info from "../../package.json" with { type: "json" };

function createChipherKeyFromString(str) {
	return crypto.createHash("sha256").update(str).digest();
}

const DEFAULT_CIPHER_KEY = createChipherKeyFromString(hash(info));

function encrypt(buffer, cipherKey = DEFAULT_CIPHER_KEY) {
	const iv = crypto.randomBytes(12);
	const cipher = crypto.createCipheriv("aes-256-gcm", cipherKey, iv);
	const encrypted = Buffer.concat([cipher.update(buffer), cipher.final()]);
	const authTag = cipher.getAuthTag();
	return Buffer.concat([iv, authTag, encrypted]);
}

function decrypt(encryptedBuffer, chipherKey = DEFAULT_CIPHER_KEY) {
	const iv = encryptedBuffer.subarray(0, 12);
	const authTag = encryptedBuffer.subarray(12, 28);
	const data = encryptedBuffer.subarray(28);
	const decipher = crypto.createDecipheriv("aes-256-gcm", chipherKey, iv);
	decipher.setAuthTag(authTag);
	return Buffer.concat([decipher.update(data), decipher.final()]);
}

export default {
	createChipherKeyFromString,
	encrypt,
	decrypt
};
