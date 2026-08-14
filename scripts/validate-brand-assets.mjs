// 校验品牌资产：验证 ICO/ICNS 结构与文本资产 emoji 扫描。
// 用法：node scripts/validate-brand-assets.mjs
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, extname } from "node:path";

const root = process.cwd();

function read(path) {
  return readFileSync(path);
}

function validateIco() {
  const data = read(join(root, "build/windows/icon.ico"));
  const header = data.readUInt16LE(0);
  if (header !== 0) throw new Error("ICO 头无效");
  const count = data.readUInt16LE(4);
  if (count !== 7) throw new Error(`ICO 条目数 = ${count}, want 7`);
  const sizes = [];
  for (let index = 0; index < count; index++) {
    const offset = 6 + index * 16;
    const width = data.readUInt8(offset);
    const height = data.readUInt8(offset + 1);
    const size = width === 0 ? 256 : width;
    if (size !== (height === 0 ? 256 : height)) {
      throw new Error(`ICO 条目 ${index} 宽高不一致`);
    }
    sizes.push(size);
  }
  const want = [16, 24, 32, 48, 64, 128, 256];
  if (JSON.stringify(sizes) !== JSON.stringify(want)) {
    throw new Error(`ICO sizes = ${sizes.join(",")}, want ${want.join(",")}`);
  }
  return sizes;
}

function validateIcns() {
  const data = read(join(root, "build/darwin/icon.icns"));
  if (data.toString("ascii", 0, 4) !== "icns") throw new Error("ICNS 头无效");
  if (data.readUInt32BE(4) !== data.length) throw new Error("ICNS 长度不匹配");
  const kinds = [];
  let offset = 8;
  while (offset < data.length) {
    const kind = data.toString("ascii", offset, offset + 4);
    const length = data.readUInt32BE(offset + 4);
    if (length <= 8) throw new Error(`ICNS 条目 ${kind} 长度无效`);
    kinds.push(kind);
    offset += length;
  }
  if (offset !== data.length) throw new Error("ICNS 结构不完整");
  const want = ["icp4", "icp5", "icp6", "ic07", "ic08", "ic09", "ic10"];
  if (JSON.stringify(kinds) !== JSON.stringify(want)) {
    throw new Error(`ICNS entries = ${kinds.join(",")}, want ${want.join(",")}`);
  }
  return kinds;
}

function isEmoji(codepoint) {
  return (
    (codepoint >= 0x1f000 && codepoint <= 0x1faff) ||
    (codepoint >= 0x2600 && codepoint <= 0x27bf) ||
    (codepoint >= 0x2300 && codepoint <= 0x23ff) ||
    codepoint === 0xfe0f
  );
}

function validateTextAssets() {
  const roots = [
    join(root, "DESIGN.md"),
    join(root, "assets/brand"),
    join(root, "build"),
    join(root, "frontend/public/brand"),
  ];
  const binary = new Set([".png", ".ico", ".icns"]);
  // 构建产物目录不参与品牌文本扫描
  const skipDirs = new Set([join(root, "build/bin"), join(root, "frontend/dist")]);
  const walk = (path) => {
    if (skipDirs.has(path)) return;
    const stat = statSync(path);
    if (stat.isFile()) {
      if (binary.has(extname(path).toLowerCase())) return;
      const text = readFileSync(path, "utf8");
      for (const char of text) {
        const codepoint = char.codePointAt(0);
        if (isEmoji(codepoint)) {
          throw new Error(`emoji in ${path}`);
        }
      }
      return;
    }
    for (const entry of readdirSync(path)) {
      walk(join(path, entry));
    }
  };
  for (const base of roots) {
    walk(base);
  }
}

function main() {
  const icoSizes = validateIco();
  const icnsKinds = validateIcns();
  validateTextAssets();
  console.log(`ICO sizes: ${icoSizes.join(", ")}`);
  console.log(`ICNS entries: ${icnsKinds.join(", ")}`);
  console.log("Text asset emoji scan: clean");
}

main();
