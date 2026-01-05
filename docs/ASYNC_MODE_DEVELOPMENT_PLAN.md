# SSH MCP Server - 异步交互模式开发计划

## 📋 项目概述

**目标：** 将 SSH MCP Server 从"同步等待"模式改造为"异步挂起+主动查询"模式，并实施三层保活机制。

**测试环境：**
- 主机：192.168.3.7
- 用户：cigar
- 密码：liuxuejia.123

**停止条件：** 所有功能实现、测试通过、文档完整、可交付

---

## 🎯 核心设计理念

### **异步交互模式**
```
旧模式（同步）：
ssh_shell() → 等待输出 → 返回结果 ❌ 可能卡住

新模式（异步）：
ssh_shell() → 立即返回 ✅
ssh_write_input() → 立即返回 ✅
ssh_read_output(strategy="latest_lines", limit=20) → 灵活读取 ✅
```

### **三层保活机制**
```
层 1：TCP Keepalive (30s)     → 防止路由器/NAT超时
层 2：SSH Keepalive (30s)     → 协议层探活
层 3：应用层心跳 (60s)        → ANSI控制码，无可见效果
```

---

## 📅 开发阶段划分

### **阶段 1：基础设施 ✅ (已完成)**
- [x] 环形缓冲区实现
- [x] TCP Keepalive
- [x] SSH Keepalive
- [x] 应用层心跳
- [x] 后台输出读取 goroutine
- [x] 优雅关闭机制

### **阶段 2：测试当前实现 (当前阶段)**
- [ ] 单元测试（保活机制、缓冲区）
- [ ] 功能测试（连接、保活、输出读取）
- [ ] 长时间稳定性测试

### **阶段 3：异步模式实现**
- [ ] 修改 `ssh_shell` 为立即返回
- [ ] 重构 `ssh_read_output` 支持多种策略
- [ ] 移除 `non_blocking` 参数
- [ ] 增强 `ssh_shell_status` 返回详细信息

### **阶段 4：集成测试**
- [ ] 完整交互流程测试
- [ ] 多种场景测试
- [ ] 边界条件测试

### **阶段 5：文档和交付**
- [ ] API 文档更新
- [ ] 使用示例
- [ ] README 更新
- [ ] 发布说明

---

## 🔍 阶段 2：当前功能测试计划

### **测试目标**
验证已完成的基础设施是否正常工作

### **2.1 单元测试**

#### **测试 1：环形缓冲区测试**
**文件：** `pkg/sshmcp/circular_buffer_test.go`

**测试用例：**
```go
func TestCircularBuffer_BasicOperations(t *testing.T)
func TestCircularBuffer_Overflow(t *testing.T)
func TestCircularBuffer_ReadLatestLines(t *testing.T)
func TestCircularBuffer_ReadLatestBytes(t *testing.T)
func TestCircularBuffer_ReadAllUnread(t *testing.T)
func TestCircularBuffer_HeartbeatFiltering(t *testing.T)
func TestCircularBuffer_ConcurrentAccess(t *testing.T)
```

#### **测试 2：保活机制测试**
**文件：** `pkg/sshmcp/keepalive_test.go`

**测试用例：**
```go
func TestTCPKeepAlive_Enabled(t *testing.T)
func TestSSHKeepAlive_SendsKeepalive(t *testing.T)
func TestApplicationHeartbeat_SendsANSICodes(t *testing.T)
func TestKeepaliveFailureDetection(t *testing.T)
```

### **2.2 功能测试**

#### **测试 3：连接和保活功能测试**
**文件：** `pkg/sshmcp/integration_keepalive_test.go`

**测试场景：**
1. 创建 SSH 连接
2. 启动 shell（raw 和 cooked 模式）
3. 等待 2 分钟，验证连接未断开
4. 检查 keepalive 计数
5. 验证缓冲区无心跳数据

**测试命令：**
```bash
go test -v ./pkg/sshmcp -run TestConnectionKeepalive \
  -ssh-host=192.168.3.7 \
  -ssh-user=cigar \
  -ssh-pass=liuxuejia.123
```

#### **测试 4：后台输出读取测试**
**文件：** `pkg/sshmcp/integration_output_test.go`

**测试场景：**
1. 启动 shell
2. 发送 `top` 命令
3. 等待 5 秒
4. 从缓冲区读取最新 20 行
5. 验证输出内容有效

**测试命令：**
```bash
go test -v ./pkg/sshmcp -run TestBackgroundOutputReading \
  -ssh-host=192.168.3.7 \
  -ssh-user=cigar \
  -ssh-pass=liuxuejia.123
```

#### **测试 5：长时间稳定性测试**
**文件：** `pkg/sshmcp/integration_stability_test.go`

**测试场景：**
- 运行 10 分钟
- 每 30 秒检查一次状态
- 验证连接持续活跃
- 验证缓冲区正常工作

---

## 🚀 阶段 3：异步模式实现计划

### **3.1 修改 `ssh_shell` 工具**

**文件：** `pkg/mcp/handlers.go`

**改动：**
```go
func (s *Server) handleSSHShell(...) {
    // 创建 shell（已经启动后台 goroutine）
    shellSession, err := session.CreateShellWithConfig(...)

    // ⚡ 立即返回，不等待输出
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            &mcp.TextContent{
                Text: fmt.Sprintf(`✅ Shell 会话已启动（后台运行）

会话信息：
- 会话 ID: %s
- 模式: %s
- 终端: %dx%d
- 缓冲区大小: %d 行
- 保活机制: 已启用（TCP + SSH + 应用层）

后续操作：
1. 发送命令：
   ssh_write_input(session_id="%s", input="your command")

2. 读取输出（多种策略）：
   - ssh_read_output(session_id="%s", strategy="latest_lines", limit=20)
   - ssh_read_output(session_id="%s", strategy="all_unread")
   - ssh_read_output(session_id="%s", strategy="latest_bytes", limit=4096)

3. 查看状态：
   ssh_shell_status(session_id="%s")

4. 发送特殊字符：
   ssh_write_input(session_id="%s", special_char="ctrl+c")
`,
                    sessionID, mode, cols, rows, bufferSize,
                    sessionID, sessionID, sessionID, sessionID, sessionID),
            },
        },
    }, nil, nil
}
```

### **3.2 重构 `ssh_read_output` 工具**

**文件：** `pkg/mcp/schemas.go` + `pkg/mcp/handlers.go`

**新的 Schema：**
```go
func sshReadOutputSchema() map[string]any {
    return getCommonJSONSchema(map[string]any{
        "session_id": map[string]any{
            "type": "string",
            "description": "会话 ID 或别名",
        },
        "strategy": map[string]any{
            "type": "string",
            "description": `读取策略：
- "latest_lines"：读取最新 N 行（默认）
- "all_unread"：读取所有未读数据
- "latest_bytes"：读取最新 N 字节

推荐使用 "latest_lines" + limit=20-50`,
            "enum": []string{"latest_lines", "all_unread", "latest_bytes"},
            "default": "latest_lines",
        },
        "limit": map[string]any{
            "type": "integer",
            "description": "读取限制（配合 strategy 使用）。
- latest_lines: 读取多少行（默认 20）
- latest_bytes: 读取多少字节（默认 4096）",
            "default": 20,
        },
    }, []string{"session_id"})
}
```

**新的 Handler：**
```go
func (s *Server) handleSSHReadOutput(...) {
    strategy := args["strategy"].(string)
    limit := int(args["limit"].(float64))

    var output string
    switch strategy {
    case "latest_lines":
        lines := shellSession.OutputBuffer.ReadLatestLines(limit)
        output = strings.Join(lines, "\n")
    case "all_unread":
        lines := shellSession.OutputBuffer.ReadAllUnread()
        output = strings.Join(lines, "\n")
    case "latest_bytes":
        output = shellSession.OutputBuffer.ReadLatestBytes(limit)
    }

    // 返回输出 + 统计信息
    return fmt.Sprintf(`📄 输出读取结果

读取策略: %s
读取行数: %d
未读剩余: %d 行
缓冲区使用: %d/%d (%.1f%%)

--- 输出内容 ---
%s
--- 输出结束 ---

提示：
- 如需查看更多输出，增加 limit 参数
- 如需查看所有未读输出，使用 strategy="all_unread"
- 查看详细状态：ssh_shell_status(session_id="%s")`,
        strategy, lineCount, remainingCount,
        used, total, percentage,
        output, sessionID)
}
```

### **3.3 移除 `non_blocking` 参数**

**文件：** `pkg/mcp/schemas.go`

**改动：**
- 从 `ssh_read_output` schema 中移除 `non_blocking` 字段
- 更新描述，明确说明现在是异步模式

### **3.4 增强 `ssh_shell_status` 工具**

**文件：** `pkg/mcp/handlers.go`

**新增显示：**
```go
func (s *Server) handleSSHShellStatus(...) {
    status := shellSession.GetStatus()

    return fmt.Sprintf(`🔍 会话状态

基本信息：
- 会话 ID: %s
- 状态: %s (活动/空闲/关闭)
- 运行时长: %s
- 上次活动: %s 前

上下文：
- 当前目录: %s
- 终端模式: %s
- 终端大小: %dx%d

输出缓冲区：
- 缓冲行数: %d/%d (%.1f%%)
- 未读数据: %s

保活状态：
- 上次 keepalive: %s 前
- 连续失败: %d 次
- 状态: %s

建议：
%s`,
        sessionID, state, uptime, lastActivity,
        currentDir, mode, rows, cols,
        bufferUsed, bufferTotal, percentage,
        hasUnread,
        lastKeepalive, fails, keepaliveStatus,
        suggestions)
}
```

---

## 🧪 阶段 4：集成测试计划

### **测试 6：完整交互流程测试**
**文件：** `pkg/mcp/integration_async_flow_test.go`

**测试场景：**
```go
func TestAsyncFlow_CompleteInteraction(t *testing.T) {
    // 1. 连接
    // 2. 启动 shell（raw 模式）→ 验证立即返回
    // 3. 发送 top 命令
    // 4. 等待 2 秒
    // 5. 读取最新 20 行
    // 6. 发送 q 命令退出
    // 7. 验证状态
}
```

### **测试 7：多种读取策略测试**
**文件：** `pkg/mcp/integration_read_strategies_test.go`

**测试场景：**
```go
func TestReadStrategies_All(t *testing.T) {
    // 1. 启动 shell
    // 2. 发送 for i in {1..100}; do echo "Line $i"; sleep 0.1; done
    // 3. 等待 5 秒
    // 4. 测试 latest_lines
    // 5. 测试 all_unread
    // 6. 测试 latest_bytes
}
```

### **测试 8：边界条件测试**
**文件：** `pkg/mcp/integration_edge_cases_test.go`

**测试场景：**
- 读取时缓冲区为空
- 读取超过缓冲区大小的数据
- 读取负数 limit
- 多个并发读取
- 会话关闭后读取

---

## 📚 阶段 5：文档和交付

### **5.1 API 文档更新**

**文件：** `docs/async-mode-api.md`

**内容：**
- 异步模式设计说明
- 所有工具的完整 API 文档
- 参数说明
- 返回值说明
- 错误处理

### **5.2 使用示例**

**文件：** `examples/async-mode-examples.md`

**示例场景：**
1. 运行 top 并监控输出
2. 运行长时间命令并定期检查
3. 运行交互式程序（vim、python）
4. 批量操作并保持状态

### **5.3 README 更新**

**文件：** `README.md` + `README_CN.md`

**新增章节：**
- 异步交互模式介绍
- 保活机制说明
- 迁移指南（从旧版本）
- 性能对比
- 故障排查

### **5.4 发布说明**

**文件：** `CHANGELOG.md`

**内容：**
- 新功能列表
- 破坏性变更
- 升级指南
- 已知问题

---

## ✅ 验收标准

### **功能完整性**
- [ ] 所有阶段 3 功能实现完成
- [ ] 所有测试用例通过
- [ ] 文档完整

### **代码质量**
- [ ] 单元测试覆盖率 > 80%
- [ ] 无编译警告
- [ ] 代码格式化
- [ ] 注释完整

### **稳定性**
- [ ] 长时间运行测试通过（1小时+）
- [ ] 内存泄漏测试通过
- [ ] 并发测试通过

### **可交付性**
- [ ] 文档完整清晰
- [ ] 示例可运行
- [ ] 测试可复现
- [ ] 用户可自助使用

---

## 📝 TODO 列表

### **当前 TODO（阶段 2：测试）**
- [ ] 创建单元测试文件
- [ ] 实现环形缓冲区测试
- [ ] 实现保活机制测试
- [ ] 实现功能测试（连接、保活、输出）
- [ ] 实现长时间稳定性测试
- [ ] 运行所有测试并修复问题
- [ ] 达到测试通过率 100%

### **后续 TODO（阶段 3-5）**
- [ ] 修改 ssh_shell 为立即返回
- [ ] 重构 ssh_read_output
- [ ] 移除 non_blocking 参数
- [ ] 增强 ssh_shell_status
- [ ] 实现集成测试
- [ ] 更新文档
- [ ] 最终验收

---

## 🎯 交付物清单

1. ✅ **源代码**：完整实现所有功能
2. ✅ **单元测试**：覆盖率 > 80%
3. ✅ **集成测试**：所有场景通过
4. ✅ **API 文档**：完整的 API 说明
5. ✅ **使用示例**：可运行的示例
6. ✅ **README**：用户指南
7. ✅ **CHANGELOG**：变更说明
8. ✅ **可执行文件**：编译后的二进制

---

## 📞 联系和反馈

**开发者：** Claude Code + Human
**测试环境：** cigar@192.168.3.7
**完成时间：** 预计 2-3 小时（全程自动化）

**备注：** 本文档将作为整个开发过程的指导文档，所有工作将严格按照此计划执行。
