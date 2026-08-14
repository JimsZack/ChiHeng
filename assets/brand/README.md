# ChiHeng Brand Assets

The originals in this directory are the only editable masters. Do not trace exported PNG/ICO/ICNS files.

| Asset | Master dimensions | Purpose |
|---|---:|---|
| `chiheng-app-icon.svg` | 1024 square viewBox | application icon master |
| `chiheng-mark.svg` | 256 square viewBox | in-product identity mark |
| `chiheng-tray-template.svg` | 32 square viewBox | black monochrome tray/template master |
| `chiheng-tray-template-inverse.svg` | 32 square viewBox | white monochrome tray master |
| `chiheng-splash.svg` | 1200×720 viewBox | launch splash master |
| `chiheng-ui-icons.svg` | 24 square symbols | static/bootstrap status icons only |

Keep application-icon padding, corner radius, gradient, and mark geometry unchanged. The tray assets must stay single color with transparent background. UI code uses Phosphor regular icons according to `DESIGN.md`; the sprite exists only for surfaces where the React icon library cannot load.

Generated destinations:

- `build/appicon.png`: Wails 1024×1024 source.
- `build/windows/icon.ico`: 16, 24, 32, 48, 64, 128, and 256 px PNG entries.
- `build/darwin/icon.icns`: 16, 32, 64, 128, 256, 512, and 1024 px entries.
- `build/linux/icons/hicolor/*/apps/com.chiheng.app.png`: freedesktop raster sizes.
- `build/tray/`: black and inverse-white 16, 24, and 32 px PNGs plus SVG masters.
- `frontend/public/brand/`: browser-ready mark, app icon, and splash assets.

Validation:

```sh
file build/appicon.png build/windows/icon.ico build/darwin/icon.icns build/tray/*.png
find build/linux/icons/hicolor -type f -name '*.png' -exec file {} +
node scripts/validate-brand-assets.mjs
```

Small-size visual checks cover 16, 24, 32, and 128 px on light and dark grounds, with application icons also viewed through a grayscale filter. Re-run these checks after any master change.
