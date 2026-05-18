# 发现与决策
<!--
  作用：作为你的分析会话知识库，记录数据来源、假设与结果。
  原因：上下文窗口有限，这个文件是你进行分析工作时的“外部记忆”。
  时机：每当有任何发现后都要更新，尤其是在运行查询或查看图表之后。
-->

## 数据来源
<!--
  作用：记录你连接过的每一个数据源，包括 schema 细节和质量备注。
  原因：清楚数据来自哪里以及它有哪些局限，对结果可复现性至关重要。
  示例：
    | user_events | PostgreSQL prod replica | 2.3M rows | user_id, event_type, ts | 0.2% null user_id |
    | revenue.csv | Finance team export | 45K rows | account_id, mrr, churn_date | Complete, no nulls |
-->
| 来源 | 位置 | 规模 | 关键字段 | 质量备注 |
|------|------|------|----------|----------|
|      |      |      |          |          |

## 假设记录
<!--
  作用：记录你测试过的每个假设、使用的方法以及结果。
  原因：结构化跟踪可以避免 p-hacking，并让你的推理过程可审计。
  示例：
    | H1: Churn > 50% for low-activity users | Chi-squared test | Confirmed (p=0.003) | High |
    | H2: Feature X correlates with retention | Pearson correlation | Rejected (r=0.08) | High |
-->
| 假设 | 测试方法 | 结果 | 置信度 |
|------|----------|------|--------|
|      |          |      |        |

## 查询结果
<!--
  作用：记录你运行过的关键查询以及它们揭示了什么。
  原因：查询结果是短暂的，如果你不把结果写下来，一旦上下文重置它们就会丢失。
  时机：每次有重要查询后都立即更新，不要等待。
  示例：
    ### 按活跃分段统计流失率
    查询：SELECT activity_bucket, COUNT(*), AVG(churned) FROM user_segments GROUP BY 1
    结果：低活跃：62% 流失，中活跃：28%，高活跃：8%
    解读：活跃度与流失率之间存在很强的反向关系
-->
<!-- 为每个重要查询记录查询语句、结果摘要和解读 -->

## 统计发现
<!--
  作用：记录正式统计检验结果及所有相关指标。
  原因：把 p 值、效应量和置信区间记下来，能让结果具备可复现性。
  示例：
    | Chi-squared (churn ~ activity) | p=0.003 | Cramer's V=0.31 | Reject null: activity segments differ significantly in churn |
    | Pearson (feature_x ~ retention) | p=0.42 | r=0.08 | Fail to reject: no meaningful correlation |
-->
| 检验 | p 值 | 效应量 | 结论 |
|------|------|--------|------|
|      |      |        |      |

## 技术决策
<!--
  作用：记录分析方法选择及其原因。
  示例：
    | Use log transform on revenue | Right-skewed distribution, normalizes for parametric tests |
-->
| 决策 | 原因 |
|------|------|
|      |      |

## 遇到的问题
| 问题 | 处理方式 |
|------|----------|
|      |          |

## 资源
<!-- URL、文件路径、文档链接 -->
-

## 可视化/浏览器发现
<!--
  关键：在查看图表、仪表盘或浏览器结果后立刻更新。
  多模态内容不会保留在上下文中，必须立即转写为文本。
-->
-

---
*每进行 2 次查看/浏览器/搜索操作后就更新此文件*
*这样可以防止视觉信息丢失*
