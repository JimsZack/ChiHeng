// useRuntimeNotice：在浏览器预览模式下返回提示文本；Wails 模式返回 null。
export const useRuntimeNotice = (wailsMode: boolean): string | null => {
  if (wailsMode) {
    return null;
  }
  return "当前为浏览器预览模式，数据为演示内容；安装桌面版后自动连接本地账本。";
};
