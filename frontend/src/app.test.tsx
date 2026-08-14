import "@testing-library/jest-dom/vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { App } from "./app";

afterEach(cleanup);

describe("ChiHeng desktop shell", () => {
  it("renders all seven navigation destinations", () => {
    render(<App />);
    expect(screen.getByRole("navigation", { name: "主导航" })).toBeInTheDocument();
    for (const label of [
      "总览",
      "股票行情",
      "基金诊断",
      "外汇商品",
      "持仓管理",
      "定投计划",
      "投资知识",
    ]) {
      expect(screen.getByRole("link", { name: new RegExp(label) })).toBeInTheDocument();
    }
  });

  it("identifies browser-only preview content", () => {
    render(<App />);
    expect(screen.getByText("浏览器预览数据")).toBeInTheDocument();
    expect(screen.getByText(/不代表实时行情/)).toBeInTheDocument();
  });
});
