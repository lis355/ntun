import path from "node:path";
import url from "node:url";

import info from "../../package.json" with { type: "json" };

export const name = info.name;
export const version = info.version;
export const projectDirectory = path.resolve(path.dirname(url.fileURLToPath(import.meta.url)), "..", "..");
export const windowsBatFilePath = `C:/windows/${name}.bat`;
