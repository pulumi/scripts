// usage: node override-package-version <path-to-package-json-file> [<package-name> <package-version>]
const fs = require("fs");
const os = require("os");
const manifest = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));

for (let i = 3; i < process.argv.length - 1; i += 2) {
    console.log("forcing " + process.argv[i] + " to version " + process.argv[i + 1] + " in " + process.argv[2]);

    if (manifest.dependencies !== undefined && manifest.dependencies[process.argv[i]] !== undefined) {
        manifest.dependencies[process.argv[i]] = process.argv[i + 1];
    }

    if (manifest.devDependencies !== undefined && manifest.devDependencies[process.argv[i]] !== undefined) {
        manifest.devDependencies[process.argv[i]] = process.argv[i + 1];
    }

    if (manifest.resolutions === undefined) {
        manifest.resolutions = {};
    }

    manifest.resolutions[process.argv[i]] = process.argv[i + 1];
}

fs.writeFileSync(process.argv[2], JSON.stringify(manifest, null, 4) + os.EOL);
