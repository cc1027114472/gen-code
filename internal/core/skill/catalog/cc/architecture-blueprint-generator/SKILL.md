---
name: architecture-blueprint-generator
description: '用于生成全面的项目架构蓝图文档。它会分析代码库，识别技术栈与架构模式，生成详细的架构文档、可视化图表、实现模式说明，以及用于维持架构一致性并指导后续开发的可扩展蓝图。'
---

# 全面项目架构蓝图生成器

## 配置变量
${PROJECT_TYPE="Auto-detect|.NET|Java|React|Angular|Python|Node.js|Flutter|Other"} <!-- 主要技术栈 -->
${ARCHITECTURE_PATTERN="Auto-detect|Clean Architecture|Microservices|Layered|MVVM|MVC|Hexagonal|Event-Driven|Serverless|Monolithic|Other"} <!-- 主要架构模式 -->
${DIAGRAM_TYPE="C4|UML|Flow|Component|None"} <!-- 架构图类型 -->
${DETAIL_LEVEL="High-level|Detailed|Comprehensive|Implementation-Ready"} <!-- 需要包含的细节层级 -->
${INCLUDES_CODE_EXAMPLES=true|false} <!-- 是否包含用于说明模式的示例代码 -->
${INCLUDES_IMPLEMENTATION_PATTERNS=true|false} <!-- 是否包含详细实现模式 -->
${INCLUDES_DECISION_RECORDS=true|false} <!-- 是否包含架构决策记录 -->
${FOCUS_ON_EXTENSIBILITY=true|false} <!-- 是否强调扩展点和扩展模式 -->

## 生成提示词

下面是一份用于生成架构蓝图的完整提示词模板：

```text
创建一份完整的 `Project_Architecture_Blueprint.md` 文档，对代码库中的架构模式进行深入分析，并将其作为维护架构一致性的权威参考。请使用以下方法：

### 1. 架构检测与分析
- ${PROJECT_TYPE == "Auto-detect" ? "通过检查以下内容来分析项目结构，识别当前使用的全部技术栈和框架：
  - 项目文件和配置文件
  - 包依赖和 import 语句
  - 框架特定模式与约定
  - 构建与部署配置" : "重点关注 ${PROJECT_TYPE} 特有的模式与实践"}
  
- ${ARCHITECTURE_PATTERN == "Auto-detect" ? "通过分析以下内容来判断所采用的架构模式：
  - 文件夹组织与命名空间
  - 依赖流向与组件边界
  - 接口隔离与抽象模式
  - 组件之间的通信机制" : "说明 ${ARCHITECTURE_PATTERN} 架构是如何被实现的"}

### 2. 架构总览
- 对整体架构方法给出清晰、简洁的说明
- 记录架构选择中体现出的指导原则
- 识别架构边界以及这些边界是如何被约束的
- 标注任何混合型架构模式，或标准模式的改造方式

### 3. 架构可视化
${DIAGRAM_TYPE != "None" ? `创建 ${DIAGRAM_TYPE} 图，并覆盖多个抽象层级：
- 展示主要子系统的高层架构总览图
- 展示关系与依赖的组件交互图
- 展示信息在系统中流动方式的数据流图
- 确保图表反映的是实际实现，而不是理论模式` : "基于真实代码依赖关系，用清晰的文字说明组件关系，内容包括：
- 子系统组织方式及边界
- 依赖方向与组件交互
- 数据流与处理流程"}

### 4. 核心架构组件
对于在代码库中发现的每一个架构组件：

- **Purpose and Responsibility**：
  - 在整体架构中的主要作用
  - 所覆盖的业务域或技术关注点
  - 边界与范围限制

- **Internal Structure**：
  - 组件内部类/模块的组织方式
  - 关键抽象及其实现
  - 所使用的设计模式

- **Interaction Patterns**：
  - 组件如何与其他组件通信
  - 暴露和消费的接口
  - 依赖注入模式
  - 事件发布/订阅机制

- **Evolution Patterns**：
  - 该组件可以如何被扩展
  - 变体点和插件机制
  - 配置和定制方式

### 5. 架构层次与依赖
- 映射代码库中实际实现的分层结构
- 记录各层之间的依赖规则
- 识别支撑分层隔离的抽象机制
- 标注任何循环依赖或分层违规
- 记录为维持隔离所采用的依赖注入模式

### 6. 数据架构
- 记录领域模型结构及其组织方式
- 映射实体关系和聚合模式
- 识别数据访问模式（repository、data mapper 等）
- 记录数据转换与映射方式
- 标注缓存策略及其实现
- 记录数据校验模式

### 7. 横切关注点的实现
记录横切关注点的实现模式：

- **Authentication & Authorization**：
  - 安全模型的实现方式
  - 权限控制模式
  - 身份管理方式
  - 安全边界模式

- **Error Handling & Resilience**：
  - 异常处理模式
  - 重试与熔断实现
  - 回退与优雅降级策略
  - 错误上报与监控方式

- **Logging & Monitoring**：
  - 埋点模式
  - 可观测性实现
  - 诊断信息流
  - 性能监控方式

- **Validation**：
  - 输入校验策略
  - 业务规则校验实现
  - 校验职责分布
  - 错误报告模式

- **Configuration Management**：
  - 配置来源模式
  - 环境特定配置策略
  - 密钥管理方式
  - Feature Flag 实现

### 8. 服务通信模式
- 记录服务边界定义
- 识别通信协议与数据格式
- 映射同步与异步通信模式
- 记录 API 版本管理策略
- 识别服务发现机制
- 标注服务通信中的韧性模式

### 9. 技术特定的架构模式
${PROJECT_TYPE == "Auto-detect" ? "对每一个被检测到的技术栈，记录其特定架构模式：" : `记录 ${PROJECT_TYPE} 特定的架构模式：`}

${(PROJECT_TYPE == ".NET" || PROJECT_TYPE == "Auto-detect") ? 
"#### .NET 架构模式（如有）
- Host 与应用模型实现
- 中间件管线组织方式
- 框架服务集成模式
- ORM 与数据访问方式
- API 实现模式（controllers、minimal APIs 等）
- 依赖注入容器配置" : ""}

${(PROJECT_TYPE == "Java" || PROJECT_TYPE == "Auto-detect") ? 
"#### Java 架构模式（如有）
- 应用容器与启动流程
- 依赖注入框架使用方式（Spring、CDI 等）
- AOP 实现模式
- 事务边界管理
- ORM 配置与使用模式
- 服务实现模式" : ""}

${(PROJECT_TYPE == "React" || PROJECT_TYPE == "Auto-detect") ? 
"#### React 架构模式（如有）
- 组件组合与复用策略
- 状态管理架构
- 副作用处理模式
- 路由与导航方式
- 数据获取与缓存模式
- 渲染优化策略" : ""}

${(PROJECT_TYPE == "Angular" || PROJECT_TYPE == "Auto-detect") ? 
"#### Angular 架构模式（如有）
- 模块组织策略
- 组件层级设计
- 服务与依赖注入模式
- 状态管理方式
- 响应式编程模式
- 路由守卫实现" : ""}

${(PROJECT_TYPE == "Python" || PROJECT_TYPE == "Auto-detect") ? 
"#### Python 架构模式（如有）
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
  - 抽象层级决策
  - 泛型与专用接口模式
  - 默认实现模式

- **Service Implementation Patterns**：
  - 服务生命周期管理
  - 服务组合模式
  - 操作实现模板
  - 服务内部的错误处理

- **Repository Implementation Patterns**：
  - 查询模式实现
  - 事务管理
  - 并发处理
  - 批量操作模式

- **Controller/API Implementation Patterns**：
  - 请求处理模式
  - 响应格式化方式
  - 参数校验
  - API 版本实现

- **Domain Model Implementation**：
  - 实体实现模式
  - 值对象模式
  - 领域事件实现
  - 业务规则约束方式" : "说明详细的实现模式在代码库中可能存在差异。"}

### 11. 测试架构
- 记录与该架构匹配的测试策略
- 识别测试边界模式（单元、集成、系统）
- 映射测试替身与 mock 方法
- 记录测试数据策略
- 标注测试工具和框架的集成方式

### 12. 部署架构
- 基于配置记录部署拓扑
- 识别环境特定的架构适配方式
- 映射运行时依赖解析模式
- 记录跨环境的配置管理方式
- 识别容器化与编排方法
- 标注云服务集成模式

### 13. 扩展与演进模式
${FOCUS_ON_EXTENSIBILITY ? 
"为扩展架构提供详细指导：

- **Feature Addition Patterns**：
  - 如何在保持架构完整性的前提下新增功能
  - 不同类型新组件应放在哪里
  - 新增依赖的指导原则
  - 配置扩展模式

- **Modification Patterns**：
  - 如何安全地修改现有组件
  - 维持向后兼容的策略
  - 废弃模式
  - 迁移方式

- **Integration Patterns**：
  - 如何集成新的外部系统
  - 适配器实现模式
  - 防腐层模式
  - 服务门面实现" : "记录该架构中的关键扩展点。"}

${INCLUDES_CODE_EXAMPLES ? 
"### 14. 架构模式示例
提取代表性代码示例，用于说明关键架构模式：

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

对每个示例保留足够上下文，以清楚展示模式，但保持示例简洁并聚焦架构概念。" : ""}

${INCLUDES_DECISION_RECORDS ? 
"### 15. 架构决策记录
记录代码库中体现出的关键架构决策：

- **Architectural Style Decisions**：
  - 为什么选择当前架构模式
  - 结合代码演进推断曾考虑过哪些替代方案
  - 影响该决策的约束条件

- **Technology Selection Decisions**：
  - 关键技术选择及其架构影响
  - 框架选择理由
  - 自研与现成组件之间的取舍

- **Implementation Approach Decisions**：
  - 具体实现模式的选择
  - 对标准模式的改造
  - 性能与可维护性之间的权衡

对每一项决策，记录：
- 导致该决策产生的背景
- 决策时考虑的因素
- 带来的后果（正面与负面）
- 引入的未来灵活性或限制" : ""}

### ${INCLUDES_DECISION_RECORDS ? "16" : INCLUDES_CODE_EXAMPLES ? "15" : "14"}. 架构治理
- 记录如何维持架构一致性
- 识别用于架构合规的自动化检查
- 标注代码库中体现出的架构评审流程
- 记录架构文档维护方式

### ${INCLUDES_DECISION_RECORDS ? "17" : INCLUDES_CODE_EXAMPLES ? "16" : "15"}. 面向新开发的蓝图
创建一份清晰的架构指南，用于实现新功能：

- **Development Workflow**：
  - 不同类型功能的起点
  - 组件创建顺序
  - 与现有架构的集成步骤
  - 按架构层划分的测试方式

- **Implementation Templates**：
  - 关键架构组件的基础类/接口模板
  - 新组件的标准文件组织方式
  - 依赖声明模式
  - 文档要求

- **Common Pitfalls**：
  - 需要避免的架构违规
  - 常见架构错误
  - 性能注意事项
  - 测试盲点

包含该蓝图的生成时间信息，以及在架构演进过程中如何保持其持续更新的建议。
```
