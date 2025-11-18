# tea/dara WebSocket 功能分析

## 概述

`tea/dara` 包提供了完整的 WebSocket 客户端实现，支持标准 WebSocket 协议以及两种应用层协议（AWAP 和 General）。

---

## 核心架构

### 1. 接口设计

#### WebSocketClient 接口
```go
type WebSocketClient interface {
    Connect(ctx context.Context) (map[string]interface{}, error)
    Disconnect(ctx context.Context) error
    Reconnect(ctx context.Context) (map[string]interface{}, error)
    IsConnected() bool
    SendText(ctx context.Context, text string) error
    SendBinary(ctx context.Context, data []byte) error
    GetSessionInfo() *WebSocketSessionInfo
    Close() error
}
```

#### WebSocketHandler 接口
```go
type WebSocketHandler interface {
    AfterConnectionEstablished(session *WebSocketSessionInfo) error
    HandleRawMessage(session *WebSocketSessionInfo, message *WebSocketMessage) error
    HandleError(session *WebSocketSessionInfo, err error) error
    AfterConnectionClosed(session *WebSocketSessionInfo, code int, reason string) error
    SupportsPartialMessages() bool
}
```

---

## 功能清单

### ✅ 1. 连接管理

#### 1.1 建立连接
- **功能**: `Connect(ctx context.Context)`
- **特性**:
  - 支持 `ws://` 和 `wss://` 协议
  - 自定义握手头部
  - 连接超时控制
  - 握手超时控制
  - TLS/SSL 支持（wss）
  - 返回连接结果和响应信息
- **实现位置**: `websocket.go:117-195`

#### 1.2 断开连接
- **功能**: `Disconnect(ctx context.Context)`
- **特性**:
  - 发送关闭帧（Close Frame）
  - 优雅关闭（等待所有 goroutine 完成）
  - 调用关闭回调
- **实现位置**: `websocket.go:197-233`

#### 1.3 重连机制
- **功能**: `Reconnect(ctx context.Context)`
- **特性**:
  - 自动重连开关
  - 最大重连次数限制
  - 重连间隔配置
  - 重连成功后重置计数
- **实现位置**: `websocket.go:235-263`

#### 1.4 连接状态
- **功能**: `IsConnected() bool`
- **特性**:
  - 原子操作保证线程安全
  - 状态值：0=disconnected, 1=connecting, 2=connected, 3=disconnecting
- **实现位置**: `websocket.go:265-267`

---

### ✅ 2. 消息发送

#### 2.1 发送文本消息
- **功能**: `SendText(ctx context.Context, text string) error`
- **特性**:
  - 连接状态检查
  - 写入超时控制
  - 错误处理
- **实现位置**: `websocket.go:269-279`

#### 2.2 发送二进制消息
- **功能**: `SendBinary(ctx context.Context, data []byte) error`
- **特性**:
  - 连接状态检查
  - 写入超时控制
  - 错误处理
- **实现位置**: `websocket.go:281-291`

---

### ✅ 3. 消息接收

#### 3.1 原始消息处理
- **功能**: `HandleRawMessage()`
- **特性**:
  - 支持文本和二进制消息
  - 消息类型转换
  - 时间戳记录
  - 错误处理
- **实现位置**: `websocket.go:310-365`

#### 3.2 消息类型
- **支持类型**:
  - `WebSocketMessageTypeText` - 文本消息
  - `WebSocketMessageTypeBinary` - 二进制消息
  - `WebSocketMessageTypePing` - Ping 消息
  - `WebSocketMessageTypePong` - Pong 消息
  - `WebSocketMessageTypeClose` - 关闭消息
- **实现位置**: `websocket.go:17-25`

#### 3.3 部分消息支持
- **功能**: `SupportsPartialMessages() bool`
- **特性**:
  - 可配置是否支持分片消息
  - 通过 Handler 接口控制
- **实现位置**: `websocket.go:68`

---

### ✅ 4. 心跳机制

#### 4.1 Ping/Pong 机制
- **功能**: `startPingPong()`
- **特性**:
  - 可配置 Ping 间隔
  - Pong 超时检测
  - 超时后自动重连
  - 独立的 goroutine 运行
- **实现位置**: `websocket.go:367-417`

#### 4.2 心跳配置
- **配置项**:
  - `PingInterval` - Ping 间隔（毫秒）
  - `PongTimeout` - Pong 超时（毫秒）
- **实现位置**: `websocket.go:54-55`

---

### ✅ 5. 超时控制

#### 5.1 连接超时
- **配置**: `ConnectTimeout`
- **用途**: 控制连接建立的最大等待时间
- **实现位置**: `websocket.go:50, 149`

#### 5.2 读取超时
- **配置**: `ReadTimeout`
- **用途**: 控制读取消息的最大等待时间
- **实现位置**: `websocket.go:51, 330-332`

#### 5.3 写入超时
- **配置**: `WriteTimeout`
- **用途**: 控制写入消息的最大等待时间
- **实现位置**: `websocket.go:52, 274-276, 287-289`

#### 5.4 握手超时
- **配置**: `HandshakeTimeout`
- **用途**: 控制 WebSocket 握手的最大等待时间
- **实现位置**: `websocket.go:53, 127`

---

### ✅ 6. 自动重连

#### 6.1 重连配置
- **配置项**:
  - `EnableReconnect` - 是否启用自动重连
  - `ReconnectInterval` - 重连间隔
  - `MaxReconnectTimes` - 最大重连次数
- **实现位置**: `websocket.go:57-59`

#### 6.2 重连触发
- **触发场景**:
  - 读取消息时连接断开
  - Ping/Pong 超时
- **实现位置**: `websocket.go:342-345, 398-400`

---

### ✅ 7. 会话管理

#### 7.1 会话信息
- **结构**: `WebSocketSessionInfo`
- **字段**:
  - `SessionID` - 会话 ID
  - `ConnectedAt` - 连接时间
  - `RemoteAddr` - 远程地址
  - `LocalAddr` - 本地地址
  - `Attributes` - 自定义属性
- **实现位置**: `websocket.go:39-45`

#### 7.2 会话获取
- **功能**: `GetSessionInfo() *WebSocketSessionInfo`
- **用途**: 获取当前会话信息
- **实现位置**: `websocket.go:293-295`

---

### ✅ 8. 错误处理

#### 8.1 错误回调
- **功能**: `HandleError()`
- **特性**:
  - 连接错误处理
  - 消息处理错误
  - Panic 恢复机制
- **实现位置**: `websocket.go:66, 311-318, 338-339`

#### 8.2 Panic 恢复
- **功能**: `recover()` 机制
- **特性**:
  - 防止 goroutine 崩溃
  - 将 panic 转换为 error
  - 通知错误处理器
- **实现位置**: `websocket.go:311-318`

---

### ✅ 9. 生命周期回调

#### 9.1 连接建立回调
- **功能**: `AfterConnectionEstablished()`
- **触发时机**: 连接成功建立后
- **实现位置**: `websocket.go:64, 184-186`

#### 9.2 连接关闭回调
- **功能**: `AfterConnectionClosed()`
- **触发时机**: 连接关闭后
- **参数**: 关闭码和原因
- **实现位置**: `websocket.go:67, 222-224`

---

### ✅ 10. 并发安全

#### 10.1 状态管理
- **特性**:
  - 使用 `atomic` 操作管理连接状态
  - 使用 `sync.Mutex` 保护重连操作
  - 使用 `sync.WaitGroup` 管理 goroutine 生命周期
- **实现位置**: `websocket.go:87-95`

#### 10.2 Goroutine 管理
- **特性**:
  - 消息读取独立 goroutine
  - 心跳独立 goroutine
  - 优雅关闭（等待所有 goroutine 完成）
- **实现位置**: `websocket.go:301-308, 367-405, 229-230`

---

### ✅ 11. AWAP 协议支持

#### 11.1 AWAP 消息结构
- **结构**: `AwapMessage`
- **字段**:
  - `Type` - 消息类型（request/response/event）
  - `ID` - 消息 ID
  - `Seq` - 序列号
  - `Headers` - 头部信息
  - `Payload` - 负载数据
  - `Format` - 格式（text/binary）
  - `Status` - 状态码
  - `Error` - 错误信息
  - `Data` - 响应数据
- **实现位置**: `websocket_awap.go:24-34`

#### 11.2 AWAP Handler
- **接口**: `AwapWebSocketHandler`
- **方法**:
  - `HandleAwapMessage()` - 处理 AWAP 消息
  - `HandleAwapIncomingMessage()` - 处理接收的 AWAP 消息
- **实现位置**: `websocket_awap.go:41-49`

#### 11.3 AWAP 消息构建
- **函数**:
  - `BuildAwapRequest()` - 构建请求消息
  - `BuildAwapResponse()` - 构建响应消息
  - `BuildAwapEvent()` - 构建事件消息
- **实现位置**: `websocket_awap.go:134-147`

#### 11.4 AWAP 消息解析
- **函数**: `ParseAwapMessage()`
- **特性**: 从 WebSocket 消息解析 AWAP 格式
- **实现位置**: `websocket_awap.go:111-122`

#### 11.5 AWAP 工具方法
- **方法**:
  - `ToJSON()` - 转换为 JSON
  - `WithHeader()` - 添加头部
  - `WithFormat()` - 设置格式
- **实现位置**: `websocket_awap.go:149-164`

---

### ✅ 12. General 协议支持

#### 12.1 General 消息结构
- **结构**: `GeneralMessage`
- **字段**:
  - `Headers` - 头部信息
  - `Body` - 消息体
- **实现位置**: `websocket_general.go:9-12`

#### 12.2 General Handler
- **接口**: `GeneralWebSocketHandler`
- **方法**:
  - `HandleGeneralTextMessage()` - 处理文本消息
  - `HandleGeneralBinaryMessage()` - 处理二进制消息
  - `HandleGeneralIncomingMessage()` - 处理接收的消息
- **实现位置**: `websocket_general.go:23-34`

#### 12.3 General 消息构建
- **函数**:
  - `BuildGeneralMessage()` - 构建通用消息
  - `BuildGeneralTextMessage()` - 构建文本消息
  - `BuildGeneralJSONMessage()` - 构建 JSON 消息
- **实现位置**: `websocket_general.go:138-158`

#### 12.4 General 消息解析
- **函数**: `ParseGeneralMessage()`
- **特性**: 
  - 支持 JSON 格式
  - 非 JSON 时作为纯文本处理
- **实现位置**: `websocket_general.go:117-136`

#### 12.5 General 工具方法
- **方法**:
  - `ToJSON()` - 转换为 JSON
  - `WithHeader()` - 添加头部
  - `GetHeader()` - 获取头部
- **实现位置**: `websocket_general.go:160-177`

---

### ✅ 13. 配置管理

#### 13.1 WebSocketConfig
- **配置项**:
  - `URL` - WebSocket URL
  - `Headers` - 自定义头部
  - `ConnectTimeout` - 连接超时
  - `ReadTimeout` - 读取超时
  - `WriteTimeout` - 写入超时
  - `HandshakeTimeout` - 握手超时
  - `PingInterval` - Ping 间隔
  - `PongTimeout` - Pong 超时
  - `MaxMessageSize` - 最大消息大小
  - `EnableReconnect` - 启用重连
  - `ReconnectInterval` - 重连间隔
  - `MaxReconnectTimes` - 最大重连次数
- **实现位置**: `websocket.go:47-60`

---

### ✅ 14. 抽象 Handler

#### 14.1 AbstractAwapWebSocketHandler
- **功能**: AWAP 协议的抽象实现
- **特性**:
  - 提供默认实现
  - 可选择性重写方法
  - 支持部分消息配置
- **实现位置**: `websocket_awap.go:51-108`

#### 14.2 AbstractGeneralWebSocketHandler
- **功能**: General 协议的抽象实现
- **特性**:
  - 提供默认实现
  - 可选择性重写方法
  - 支持部分消息配置
- **实现位置**: `websocket_general.go:36-115`

---

## 功能统计

### 已实现功能（14 大类，50+ 项）

| 类别 | 功能数 | 状态 |
|------|--------|------|
| 连接管理 | 4 | ✅ 完整 |
| 消息发送 | 2 | ✅ 完整 |
| 消息接收 | 3 | ✅ 完整 |
| 心跳机制 | 2 | ✅ 完整 |
| 超时控制 | 4 | ✅ 完整 |
| 自动重连 | 2 | ✅ 完整 |
| 会话管理 | 2 | ✅ 完整 |
| 错误处理 | 2 | ✅ 完整 |
| 生命周期回调 | 2 | ✅ 完整 |
| 并发安全 | 2 | ✅ 完整 |
| AWAP 协议 | 5 | ✅ 完整 |
| General 协议 | 5 | ✅ 完整 |
| 配置管理 | 1 | ✅ 完整 |
| 抽象 Handler | 2 | ✅ 完整 |

---

## 设计特点

### 1. 接口驱动
- 清晰的接口定义
- 易于扩展和测试
- 支持多种协议

### 2. 并发安全
- 使用原子操作管理状态
- 使用互斥锁保护关键操作
- 使用 WaitGroup 管理 goroutine

### 3. 错误处理
- 完善的错误回调机制
- Panic 恢复机制
- 优雅的错误传播

### 4. 可配置性
- 丰富的配置选项
- 灵活的超时控制
- 可配置的重连策略

### 5. 协议支持
- 标准 WebSocket
- AWAP 协议
- General 协议

---

## 使用示例

### 基础 WebSocket
```go
handler := &MyHandler{}
config := &dara.WebSocketConfig{
    URL: "ws://example.com/ws",
    // ... 其他配置
}
client, _ := dara.NewDefaultWebSocketClient(config, handler)
client.Connect(ctx)
```

### AWAP 协议
```go
handler := &MyAwapHandler{
    AbstractAwapWebSocketHandler: dara.AbstractAwapWebSocketHandler{},
}
// 使用 AWAP handler
```

### General 协议
```go
handler := &MyGeneralHandler{
    AbstractGeneralWebSocketHandler: dara.AbstractGeneralWebSocketHandler{},
}
// 使用 General handler
```

---

## 总结

`tea/dara` 的 WebSocket 实现提供了：

✅ **完整的核心功能** - 连接管理、消息收发、心跳、重连等
✅ **协议支持** - 标准 WebSocket、AWAP、General
✅ **并发安全** - 线程安全的状态管理和 goroutine 管理
✅ **错误处理** - 完善的错误处理和 Panic 恢复
✅ **可配置性** - 丰富的配置选项
✅ **易于使用** - 清晰的接口和抽象实现

这是一个功能完整、设计良好的 WebSocket 客户端实现。

