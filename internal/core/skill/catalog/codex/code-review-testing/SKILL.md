---
name: code-review-testing
description: Test authoring guidance
---

对于 agent 变更，优先选择集成测试而不是单元测试。集成测试位于 `core/suite` 下，并使用 `test_codex` 来搭建 codex 的测试实例。

会改变 agent 逻辑的功能必须添加集成测试：
- 提供需要被测试的主要逻辑变更和面向用户行为的列表。

如果需要单元测试，把它们放在专门的测试文件（`*_tests.rs`）中。
避免在主实现中加入仅供测试使用的函数。

检查是否已有现成的 helper，可以让测试更精简、更易读。
