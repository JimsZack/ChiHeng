# ChiHeng build and release

## Supported build hosts

Wails desktop packages are built natively on each target operating system. Use
Go 1.25 or newer, Node.js 22, pnpm 10, and the Wails v2 CLI. Run
`./scripts/build-all.sh` before producing an installer.

| Target | Host prerequisites | Package command |
|---|---|---|
| Windows 11 x64 | WebView2 runtime, MSVC Build Tools | `wails build -clean -platform windows/amd64` |
| macOS 14+ universal | Xcode Command Line Tools | `wails build -clean -platform darwin/universal` |
| Ubuntu 24.04 x64 | `build-essential`, `libgtk-3-dev`, `libwebkit2gtk-4.1-dev`, `libsoup-3.0-dev` | `wails build -clean -tags webkit2_41 -platform linux/amd64` |

> 实测：Ubuntu 24.04 (noble) 仓库只提供 webkit2gtk-4.1，不提供 4.0。
> Wails v2.14 通过 `-tags webkit2_41` 使用 webkit2gtk-4.1 + libsoup-3.0；
> 不带该标签时默认按 webkit2gtk-4.0 查找 pkg-config，在 noble 上会失败。

Cross-compiling GUI packages is not a supported release route. CI must run the
quality gate, Go race tests, frontend checks, and a native Wails build on all
three hosts. Release artifacts must be signed after the native build, then have
their SHA-256 checksums published beside them.

## Local build

```bash
# 一次性安装（Ubuntu 24.04）
sudo apt-get install -y build-essential libgtk-3-dev libwebkit2gtk-4.1-dev libsoup-3.0-dev pkg-config

# 全量质量门禁（emoji/密钥扫描 + Go 测试 + 前端 check + Wails 打包）
./scripts/build-all.sh

# 单独验证
go build ./...
cd frontend && pnpm check
wails build -clean -tags webkit2_41
```

产物输出至 `build/bin/`。

## Release checklist

1. Confirm `git status` contains only reviewed release changes.
2. Run `./scripts/build-all.sh` on every target host.
3. Smoke-test launch, portfolio CSV round trip, and clean shutdown.
4. Sign Windows binaries, notarize the macOS bundle, and sign Linux repository metadata.
5. Generate checksums and attach the release and privacy notes to the release.

External release credentials never belong in repository files. CI obtains
signing identities, notarization credentials, and package-registry tokens from
the platform secret store.
