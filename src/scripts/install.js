import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { projectDirectory, windowsBatFilePath } from "./appInfo.js";

switch (os.platform()) {
	case "win32": {
		fs.writeFileSync(windowsBatFilePath, `@echo off
node "${path.join(projectDirectory, "src", "ntun.cli.js")}" %*
`);

		console.log(`${windowsBatFilePath} created`);

		break;
	}
	default:
		console.log(`${os.platform()} is not supported`);
}
