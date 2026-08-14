---
name: gen-test
description: 为 ChiHeng 项目生成与现有风格一致的单元测试。Go 后端用 Given/When/Then 风格，前端用 Vitest + Testing Library。当用户要求"写测试/补测试"时使用。
disable_model_invocation: true
---

# 生成测试

按被测文件所在目录选择对应测试风格。

## Go 后端（internal/）

**文件位置**：与源文件同包，命名 `xxx_test.go`（如 `money.go` → `money_test.go`）。

**命名约定**：`Test_<函数名>_when_<场景>`，如 `Test_DecimalRoundTrip_when_PreciseValue`。

**结构约定**：Given / When / Then 注释分段：

```go
func Test_<Func>_when_<Scenario>(t *testing.T) {
	t.Parallel()

	// Given
	input := "123.456789"

	// When
	parsed, err := ParseDecimal(input)

	// Then
	if err != nil || FormatDecimal(parsed) != input {
		t.Fatalf("round trip = %q, %v", FormatDecimal(parsed), err)
	}
}
```

要点：
- 所有用例 `t.Parallel()`
- 错误场景用 `if err == nil { t.Fatal("expected ...") }`
- 金额断言用 `ParseDecimal` / `FormatDecimal` 而非浮点比较
- 运行 `go test ./internal/...` 验证

## 前端（frontend/src/）

**文件位置**：与被测组件同目录，命名 `xxx.test.tsx`。

**风格**：Vitest + `@testing-library/react` + jest-dom：

```tsx
import "@testing-library/jest-dom/vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { Component } from "./component";

afterEach(cleanup);

describe("描述", () => {
  it("行为描述", () => {
    render(<Component />);
    expect(screen.getByRole("button", { name: "按钮名" })).toBeInTheDocument();
  });
});
```

要点：
- `afterEach(cleanup)` 必须有
- 查询优先 `getByRole`，次选 `getByText` / `getByLabelText`
- 中文字面量（导航、按钮文案）用正则 `new RegExp(label)` 匹配
- 运行 `cd frontend && pnpm test` 验证
