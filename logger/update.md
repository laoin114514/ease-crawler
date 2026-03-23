# v0.2.1
这版主要优化了：

- 引入了强类型日志级别 `Level`（`Debug/Info/Warn/Error/Fatal`），避免魔法数字和字符串比较错误。
- 级别过滤逻辑更直观：只打印大于等于当前阈值的日志。
- `SetLevelOfOutput` 现在会校验输入，返回 `bool` 表示是否设置成功（无效级别不会污染当前配置）。
- `Print/Println` 类行为（你现在是 `Info/ln` 等）改为基于 `fmt.Sprint/Sprintln`，行为更贴近标准库语义。
- 给 `SetPrefix/GetPrefix` 和 level 相关读写都加了互斥保护，避免并发读写时潜在数据竞争。
- 去掉了结构体里未使用的 `out` 字段，简化状态。
- `Fatal` 退出改成走可注入 `exitFunc`（默认还是 `os.Exit`），后续测试更友好。
- 保留了全局 API（`Infof/Errorf/Debugf...`）调用方式，尽量不破坏现有代码。
