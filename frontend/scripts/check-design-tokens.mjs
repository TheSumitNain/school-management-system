import { readdirSync, readFileSync } from "node:fs";
import { extname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const sourceRoot = fileURLToPath(new URL("../src/", import.meta.url));
const violations = [];
const colorProperties = /(?:^|[;{])\s*(?:color|background(?:-color)?|border(?:-[a-z-]+)?-color|outline-color|fill|stroke|box-shadow|text-shadow)\s*:\s*([^;}]+)/gim;
const rawColor = /#[\da-f]{3,8}\b|\b(?:rgb|rgba|hsl|hsla)\s*\(|\b(?:white|black|red|blue|green|purple|orange|yellow|gray|grey|silver|maroon|navy|teal)\b/i;

function visit(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const fullPath = join(directory, entry.name);
    if (entry.isDirectory()) {
      visit(fullPath);
      continue;
    }
    if (extname(entry.name) !== ".css" || entry.name === "tokens.css") continue;

    const source = readFileSync(fullPath, "utf8");
    for (const match of source.matchAll(colorProperties)) {
      if (rawColor.test(match[1])) {
        violations.push({
          file: relative(sourceRoot, fullPath),
          declaration: match[1].trim(),
        });
      }
    }
  }
}

visit(sourceRoot);

if (violations.length > 0) {
  console.error("Design-token check failed. Move literal colors into styles/tokens.css:");
  for (const violation of violations) {
    console.error(`- ${violation.file}: ${violation.declaration}`);
  }
  process.exitCode = 1;
} else {
  console.log("Design-token check passed: component styles use centralized color tokens.");
}
