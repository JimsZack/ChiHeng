// 运行时检测：判断当前是否运行在 Wails 桌面环境中。
// Wails v2 会在 globalThis.go.main.App 注入绑定对象；浏览器预览时不存在。
import type { PreviewData } from "../shared/types";
import { previewData } from "./preview-data";

export type RuntimeMode = "wails" | "preview";

const hasWailsBindings = (): boolean => {
  const app = (globalThis as { go?: { main?: { App?: unknown } } }).go?.main?.App;
  return app !== undefined && app !== null && typeof app === "object";
};

export const getRuntimeMode = (): RuntimeMode => (hasWailsBindings() ? "wails" : "preview");

export const getPreviewData = (): PreviewData => previewData;
