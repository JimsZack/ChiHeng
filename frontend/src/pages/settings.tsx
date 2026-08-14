import { ShieldCheck } from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { PageHeader, PreviewNotice } from "../components/ui";
import {
  type AIProfile,
  getAIProfile,
  getPreferences,
  type Preferences,
  saveAIProfile,
  savePreferences,
} from "../services/api";
import { useAppStore } from "../stores/app";

export const SettingsPage = () => {
  const { wailsMode } = useAppStore();
  const [prefs, setPrefs] = useState<Preferences | null>(null);
  const [profile, setProfile] = useState<AIProfile | null>(null);
  const [apiKey, setApiKey] = useState("");
  const [model, setModel] = useState("deepseek-chat");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      const [p, a] = await Promise.all([getPreferences(), getAIProfile()]);
      setPrefs(p);
      setProfile(a);
      setModel(a.model);
    })();
  }, []);

  const onSavePreferences = async () => {
    if (!prefs) {
      return;
    }
    try {
      const saved = await savePreferences(prefs);
      setPrefs(saved);
      setMessage("偏好已保存");
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
      setMessage(null);
    }
  };

  const onSaveAI = async () => {
    try {
      const saved = await saveAIProfile({ provider: "deepseek", model, apiKey });
      setProfile(saved);
      setApiKey("");
      setMessage("AI 配置已保存，密钥仅存于本机");
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
      setMessage(null);
    }
  };

  return (
    <div className="page">
      <PageHeader title="设置" description="偏好、AI 服务与隐私边界" />
      {!wailsMode ? <PreviewNotice /> : null}
      {message ? (
        <div className="notice section" role="status">
          {message}
        </div>
      ) : null}
      {error ? (
        <div className="notice error section" role="alert">
          {error}
        </div>
      ) : null}
      <div className="grid two section">
        <section className="card">
          <h2>界面偏好</h2>
          {prefs ? (
            <div className="grid two section">
              <div className="field">
                <label htmlFor="theme">主题</label>
                <select
                  id="theme"
                  className="input"
                  value={prefs.theme}
                  onChange={(e) => setPrefs({ ...prefs, theme: e.target.value })}
                >
                  <option value="system">跟随系统</option>
                  <option value="light">浅色</option>
                  <option value="dark">深色</option>
                </select>
              </div>
              <div className="field">
                <label htmlFor="refresh">行情刷新间隔（秒）</label>
                <input
                  id="refresh"
                  className="input"
                  type="number"
                  min={10}
                  step={10}
                  value={prefs.refreshIntervalSeconds}
                  onChange={(e) =>
                    setPrefs({ ...prefs, refreshIntervalSeconds: Number(e.target.value) || 60 })
                  }
                />
              </div>
              <div className="field">
                <label htmlFor="locale">语言</label>
                <select
                  id="locale"
                  className="input"
                  value={prefs.locale}
                  onChange={(e) => setPrefs({ ...prefs, locale: e.target.value })}
                >
                  <option value="zh-CN">简体中文</option>
                  <option value="en-US">English</option>
                </select>
              </div>
              <div className="field">
                <label htmlFor="log-level">日志级别</label>
                <select
                  id="log-level"
                  className="input"
                  value={prefs.logLevel}
                  onChange={(e) => setPrefs({ ...prefs, logLevel: e.target.value })}
                >
                  <option value="debug">调试</option>
                  <option value="info">信息</option>
                  <option value="warn">警告</option>
                  <option value="error">错误</option>
                </select>
              </div>
            </div>
          ) : (
            <div className="state">正在加载偏好...</div>
          )}
          <button
            type="button"
            className="button primary section"
            onClick={() => void onSavePreferences()}
          >
            保存偏好
          </button>
        </section>
        <section className="card">
          <h2>AI 服务（DeepSeek）</h2>
          {profile ? (
            <div className="section">
              <div className={`badge ${profile.hasSecret ? "success" : "warning"}`}>
                {profile.hasSecret ? "已配置密钥" : "未配置密钥"}
              </div>
              {profile.unavailableReason ? (
                <div className="muted">{profile.unavailableReason}</div>
              ) : null}
              <div className="field section">
                <label htmlFor="ai-model">模型名称</label>
                <input
                  id="ai-model"
                  className="input"
                  value={model}
                  onChange={(e) => setModel(e.target.value)}
                />
              </div>
              <div className="field section">
                <label htmlFor="ai-key">API Key</label>
                <input
                  id="ai-key"
                  className="input"
                  type="password"
                  placeholder={
                    profile.hasSecret ? "已保存（留空则不修改）" : "输入 DeepSeek API Key"
                  }
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                />
              </div>
              <button type="button" className="button primary" onClick={() => void onSaveAI()}>
                保存 AI 配置
              </button>
              <div className="notice section">
                密钥仅保存在本机密钥存储中，不会显示或上传；AI 组合分析前会逐次请求同意。
              </div>
            </div>
          ) : (
            <div className="state">正在加载 AI 配置...</div>
          )}
        </section>
      </div>
      <section className="card section">
        <h2>
          <ShieldCheck size={18} /> 隐私与安全边界
        </h2>
        <div className="list">
          <div className="list-item">持仓数据全部保存在本地 SQLite 数据库，绝不云端上传</div>
          <div className="list-item">AI 诊断仅在明确同意后将脱敏指标发送至所选服务商</div>
          <div className="list-item">日志不包含 API Key 与持仓金额明细</div>
          <div className="list-item">数据自动备份，安全存放于本机</div>
          <div className="list-item">分析结果仅供参考，不构成投资建议</div>
        </div>
      </section>
    </div>
  );
};
