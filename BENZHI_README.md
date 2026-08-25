基于 Go 实现的 ForgeVac 项目，一款真空热处理炉安全联控服务，协调升温、渗碳、淬火、设备联锁与可恢复工艺证据。

服务通过 JSON API 管理炉次、设备状态、联锁决策和异常记录。运行数据使用本地事件日志、原子快照与漏率轨迹文件持久化。

使用固定 Go 工具链和 vendor 离线构建：

```text
go build -mod=vendor ./...
go run -mod=vendor ./cmd/forgevac -addr 127.0.0.1:21227 -data ./data
```

健康检查位于 `/healthz`，业务入口位于 `/api/operations`、`/api/equipment`、`/api/interlocks` 和 `/api/incidents`。
