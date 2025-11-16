import fs from "node:fs";
import os from "node:os";

import { windowsBatFilePath } from "./appInfo.js";

switch (os.platform()) {
	case "win32": {
		fs.rmSync(windowsBatFilePath);

		console.log(`${windowsBatFilePath} removed`);

		break;
	}
	default:
		console.log(`${os.platform()} is not supported`);
}
