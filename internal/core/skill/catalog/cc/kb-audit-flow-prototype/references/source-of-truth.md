# 真值来源

使用这份参考资料来判断原型应表达什么、应包含什么，以及应如何验证。

## 核心项目文件

主来源目录：

- `F:\codex\artifacts\kb-audit`

该目录中的重要文档：

- 业务流程总览 markdown 文件
- 站点菜单手册 markdown 文件
- 产品模块理解 markdown 文件
- 审批中心设计 markdown 文件
- 角色与权限边界 markdown 文件

请以业务流程总览文档作为节点顺序的主要来源。

## 规划与原型支持文档

在把分析结果转化为原型结构时，也要使用这些项目文档：

- Axure 页面清单 markdown 文件
- Axure 18 页线框稿 markdown 文件
- Axure 构建顺序与母版组件 markdown 文件

## 现有原型文件

当前阶段性原型目录：

- `F:\codex\artifacts\kb-audit\html-demo-prototype`

其中主要文件：

- `index.html`
- `styles.css`
- `app.js`
- usage-notes markdown file

除非用户明确要求第二个变体，否则就在这个目录中原地更新。

## 链路扩展顺序

推荐顺序：

1. 首页与导航
2. 采购与审批
3. 劳务
4. 营收侧
5. 机械

## 校验清单

每次有意义的更新后：

1. JS 必须通过语法校验。
2. 菜单入口、节点标签和实际页面必须保持一致。
3. 演示出的链路文本必须仍然与文档中的业务链路一致。
4. 在适用时，审批的强绑定规则必须仍然可见。
5. 回复中必须清楚区分：
   - fully interactive parts
   - context or support pages
   - future or lighter-depth nodes
