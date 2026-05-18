---
name: architecture-blueprint-generator
description: 用于生成全面的项目架构蓝图文档。它会分析代码库，识别技术栈与架构模式，产出详细的架构说明、可视化图表、实现模式总结以及可扩展的蓝图，用于维持架构一致性并指导后续开发。
---

# 综合项目架构蓝图生成器

## 配置变量
${PROJECT_TYPE="Auto-detect|.NET|Java|React|Angular|Python|Node.js|Flutter|Other"} <!-- 主要技术 -->
${ARCHITECTURE_PATTERN="Auto-detect|Clean Architecture|Microservices|Layered|MVVM|MVC|Hexagonal|Event-Driven|Serverless|Monolithic|Other"} <!-- 主要架构模式 -->
${DIAGRAM_TYPE="C4|UML|Flow|Component|None"} <!-- 架构图类型 -->
${DETAIL_LEVEL="High-level|Detailed|Comprehensive|Implementation-Ready"} <!-- 需要包含的细节层级 -->
${INCLUDES_CODE_EXAMPLES=true|false} <!-- 是否包含示例代码来说明模式 -->
${INCLUDES_IMPLEMENTATION_PATTERNS=true|false} <!-- 是否包含详细实现模式 -->
${INCLUDES_DECISION_RECORDS=true|false} <!-- 是否包含架构决策记录 -->
${FOCUS_ON_EXTENSIBILITY=true|false} <!-- 是否强调扩展点与扩展模式 -->

## 生成提示词

"创建一份完整的 `Project_Architecture_Blueprint.md` 文档，深入分析代码库中的架构模式，使其成为维持架构一致性的权威参考。请按以下方式进行：

### 1. 架构识别与分析
- ${PROJECT_TYPE == "Auto-detect" ? "分析项目结构，并通过以下内容识别所使用的全部技术栈与框架：
  - 项目文件与配置文件
  - 依赖包与 import 语句
  - 框架特有的模式与约定
  - 构建与部署配置" : "聚焦于 ${PROJECT_TYPE} 的特定模式与最佳实践"}
  
- ${ARCHITECTURE_PATTERN == "Auto-detect" ? "通过分析以下内容判断架构模式：
  - 目录组织与命名空间划分
  - 依赖流向与组件边界
  - 接口隔离与抽象使用模式
  - 组件间的通信机制" : "说明 ${ARCHITECTURE_PATTERN} 架构是如何落地实现的"}

### 2. 架构总览
- 用清晰、简洁的方式说明整体架构思路
- 记录从架构选择中体现出的指导原则
- 识别架构边界以及这些边界是如何被约束的
- 标注是否存在混合式架构，或对标准架构模式的改造

### 3. 架构可视化
${DIAGRAM_TYPE != "None" ? `创建 ${DIAGRAM_TYPE} 图，并覆盖多个抽象层级：
- 高层架构总览图，展示主要子系统
- 组件交互图，展示关系与依赖
- 数据流图，展示信息如何在系统中流动
- 确保图表反映的是真实实现，而不是理论上的理想结构` : "基于真实代码依赖关系，用文字清晰描述组件关系，包括：
- 子系统组织方式与边界
- 依赖方向与组件交互
- 数据流与处理顺序"}

### 4. 核心架构组件
针对代码库中发现的每一个架构组件，说明：

- **Purpose and Responsibility**：
  - 在架构中的主要作用
  - 所承载的业务域或技术关注点
  - 边界与范围限制

- **Internal Structure**：
  - 组件内部类/模块的组织方式
  - 关键抽象及其实现
  - 所采用的设计模式

- **Interaction Patterns**：
  - 该组件如何与其他组件通信
  - 对外暴露和依赖的接口
  - 依赖注入模式
  - 事件发布/订阅机制

- **Evolution Patterns**：
  - 该组件如何扩展
  - 可变点与插件机制
  - 配置与定制方式

### 5. 架构分层与依赖关系
- 映射代码库中实际存在的层次结构
- 记录层与层之间的依赖规则
- 识别支撑分层隔离的抽象机制
- 标记任何循环依赖或分层违例
- 记录用于维持分层隔离的依赖注入模式

### 6. 数据架构
- 记录领域模型的结构与组织方式
- 映射实体关系与聚合模式
- 识别数据访问模式（如 repository、data mapper 等）
- 记录数据转换与映射方式
- 标注缓存策略及其实现
- 记录数据校验模式

### 7. 横切关注点的实现
为以下横切关注点记录实现模式：

- **Authentication & Authorization**：
  - 安全模型的实现方式
  - 权限控制模式
  - 身份管理策略
  - 安全边界模式

- **Error Handling & Resilience**：
  - 异常处理模式
  - 重试与熔断机制
  - 回退与优雅降级策略
  - 错误上报与监控方式

- **Logging & Monitoring**：
  - 埋点模式
  - 可观测性实现
  - 诊断信息流向
  - 性能监控方式

- **Validation**：
  - 输入校验策略
  - 业务规则校验实现
  - 校验职责分布
  - 错误反馈模式

- **Configuration Management**：
  - 配置来源模式
  - 多环境配置策略
  - 密钥管理方式
  - Feature Flag 实现

### 8. 服务通信模式
- 记录服务边界定义方式
- 识别通信协议与格式
- 映射同步/异步通信模式
- 记录 API 版本管理策略
- 识别服务发现机制
- 标注服务通信中的弹性策略

### 9. 技术栈特定架构模式
${PROJECT_TYPE == "Auto-detect" ? "针对每一种识别出的技术栈，记录其特定架构模式：" : `记录 ${PROJECT_TYPE} 特有的架构模式：`}

${(PROJECT_TYPE == ".NET" || PROJECT_TYPE == "Auto-detect") ? 
"#### .NET 架构模式（如果识别到）
- Host 与应用模型的实现方式
- 中间件管线组织
- 框架服务集成模式
- ORM 与数据访问模式
- API 实现模式（controllers、minimal APIs 等）
- 依赖注入容器配置" : ""}

${(PROJECT_TYPE == "Java" || PROJECT_TYPE == "Auto-detect") ? 
"#### Java 架构模式（如果识别到）
- 应用容器与启动流程
- 依赖注入框架使用方式（Spring、CDI 等）
- AOP 实现模式
- 事务边界管理
- ORM 配置与使用方式
- 服务实现模式" : ""}

${(PROJECT_TYPE == "React" || PROJECT_TYPE == "Auto-detect") ? 
"#### React 架构模式（如果识别到）
- 组件组合与复用策略
- 状态管理架构
- 副作用处理模式
- 路由与导航方式
- 数据拉取与缓存模式
- 渲染优化策略" : ""}

${(PROJECT_TYPE == "Angular" || PROJECT_TYPE == "Auto-detect") ? 
"#### Angular 架构模式（如果识别到）
- 模块组织策略
- 组件层级设计
- 服务与依赖注入模式
- 状态管理方式
- 响应式编程模式
- 路由守卫实现" : ""}

${(PROJECT_TYPE == "Python" || PROJECT_TYPE == "Auto-detect") ? 
"#### Python 架构模式（如果识别到）
- 模块组织方式
- 依赖管理策略
- 面向对象与函数式实现模式
- 框架集成模式
- 异步编程方式" : ""}

### 10. 实现模式
${INCLUDES_IMPLEMENTATION_PATTERNS ? 
"记录关键架构组件的具体实现模式：

- **Interface Design Patterns**：
  - 接口隔离方式
  - 抽象层次选择
  - 通用接口与专用接口模式
  - 默认实现模式

- **Service Implementation Patterns**：
  - 服务生命周期管理
  - 服务组合方式
  - 操作实现模板
  - 服务内错误处理方式

- **Repository Implementation Patterns**：
  - 查询模式实现
  - 事务管理
  - 并发处理
  - 批量操作模式

- **Controller/API Implementation Patterns**：
  - 请求处理模式
  - 响应格式化方式
  - 参数校验
  - API 版本实现方式

- **Domain Model Implementation**：
  - 实体实现模式
  - 值对象模式
  - 领域事件实现
  - 业务规则约束方式" : "说明详细实现模式在代码库中可能存在差异。"}

### 11. 测试架构
- 记录与架构相匹配的测试策略
- 识别测试边界模式（单元、集成、系统）
- 映射测试替身与 mock 使用方式
- 记录测试数据策略
- 标注测试工具与框架的集成方式

### 12. 部署架构
- 根据配置记录部署拓扑
- 识别环境特定的架构适配
- 映射运行期依赖解析模式
- 记录多环境下的配置管理方式
- 识别容器化与编排方案
- 标注云服务集成模式

### 13. 扩展与演进模式
${FOCUS_ON_EXTENSIBILITY ? 
"提供关于如何扩展架构的详细指导：

- **Feature Addition Patterns**：
  - 在不破坏架构完整性的前提下新增功能
  - 不同类型新组件应该放在哪里
  - 新依赖的引入原则
  - 配置扩展模式

- **Modification Patterns**：
  - 如何安全修改现有组件
  - 保持向后兼容的策略
  - 废弃模式
  - 迁移方式

- **Integration Patterns**：
  - 如何接入新的外部系统
  - Adapter 实现模式
  - Anti-corruption layer 模式
  - Service facade 实现" : "记录架构中的关键扩展点。"}

${INCLUDES_CODE_EXAMPLES ? 
"### 14. 架构模式示例
提取具有代表性的代码片段，用于说明关键架构模式：

- **Layer Separation Examples**：
  - 接口定义与实现分离
  - 跨层通信模式
  - 依赖注入示例

- **Component Communication Examples**：
  - 服务调用模式
  - 事件发布与处理
  - 消息传递实现

- **Extension Point Examples**：
  - 插件注册与发现
  - 扩展接口实现
  - 配置驱动的扩展模式

每个示例都应包含足够上下文来说明模式本身，但保持简洁，只聚焦架构概念。" : ""}

${INCLUDES_DECISION_RECORDS ? 
"### 15. 架构决策记录
记录代码库中体现出的关键架构决策：

- **Architectural Style Decisions**：
  - 为什么选择当前架构模式
  - 可能考虑过的替代方案（基于代码演进推断）
  - 影响决策的约束条件

- **Technology Selection Decisions**：
  - 关键技术选型及其架构影响
  - 选择框架的理由
  - 自研与现成组件之间的取舍

- **Implementation Approach Decisions**：
  - 所采用的具体实现模式
  - 对标准模式的定制
  - 性能与可维护性之间的权衡

对每项决策都标明：
- 决策发生时的上下文
- 做决策时考虑的因素
- 带来的正负结果
- 对未来灵活性或限制的影响" : ""}

### ${INCLUDES_DECISION_RECORDS ? "16" : INCLUDES_CODE_EXAMPLES ? "15" : "14"}. 架构治理
- 记录架构一致性是如何被维持的
- 识别自动化架构合规检查
- 标注代码库中体现出的架构评审流程
- 记录架构文档维护方式

### ${INCLUDES_DECISION_RECORDS ? "17" : INCLUDES_CODE_EXAMPLES ? "16" : "15"}. 新开发蓝图
为新功能开发提供清晰的架构指南：

- **Development Workflow**：
  - 不同类型功能的起点
  - 组件创建顺序
  - 与现有架构的集成步骤
  - 各架构层的测试方式

- **Implementation Templates**：
  - 关键架构组件的基础类/接口模板
  - 新组件的标准文件组织方式
  - 依赖声明模式
  - 文档要求

- **Common Pitfalls**：
  - 需要避免的架构违例
  - 常见架构错误
  - 性能注意事项
  - 测试盲区

请附上该蓝图生成的时间，以及如何在架构持续演进时保持其更新的建议。"
