# Redis Key 命名规范

**版本： 1.0**
**日期： 2025年6月3日**

## 1. 概述

本文档旨在为安全卡片中继系统项目定义一套统一的Redis Key命名规范。规范的命名有助于提高系统的可维护性、可读性，并减少潜在的键冲突和误用。所有Redis Key的设计应遵循本规范。

## 2. 基本原则

1.  **可读性:** Key名应清晰表达其存储内容的含义。
2.  **层次性:** 使用冒号 `:` 作为分隔符，构建具有层次结构的Key名，方便管理和模式匹配。
3.  **简洁性:** 在保证可读性的前提下，尽量保持Key名简洁。
4.  **一致性:** 同类数据或功能的Key名应保持结构和用词的一致性。
5.  **避免特殊字符:** Key名中避免使用空格、换行符等可能引起问题的特殊字符，推荐使用小写字母、数字、冒号 `:` 和下划线 `_` (下划线用于单词内部连接，冒号用于分段)。

## 3. 命名结构

推荐采用以下通用结构：

`{scope}:{entity_group}:{entity_id}:{attribute_or_sub_entity}`

或者更具体的：

`{prefix}:{project_module}:{business_function}:{unique_identifier}:{details}`

-   **`scope` / `prefix` (作用域/前缀):** (必需) 用于标识Key的顶层分类，便于快速识别Key的用途。详见下文推荐前缀。
-   **`entity_group` / `project_module` (实体组/项目模块):** (可选) Key所属的模块或主要实体类型，例如 `user`, `device`, `auth`, `session`。
-   **`entity_id` / `unique_identifier` (实体ID/唯一标识符):** (必需) 具体的实体唯一标识，例如 `userID`, `clientID`, `transactionID`, `jti`。
-   **`attribute_or_sub_entity` / `details` (属性或子实体/详细信息):** (可选) 描述Key的具体属性或子实体的细节。例如 `profile`, `status`, `list`, `hash`。

## 4. 推荐前缀 (Scope)

根据开发手册建议，并结合项目实际，推荐使用以下前缀：

-   **`cfg` (Configuration):** 存储系统配置信息。
    -   示例: `cfg:system:settings`, `cfg:feature_flags:user_registration`
-   **`sts` (Status):** 存储实体或组件的实时状态信息。
    -   示例: `sts:client:{clientID}:current_screen`, `sts:user:{userID}:nfc_status`
-   **`sess` (Session):** 存储会话相关信息，包括用户登录会话、角色会话、交易会话。
    -   示例: `sess:user_role:{userID}:transmitter_client_id`
    -   示例: `sess:transaction:{transactionID}:details`
-   **`jwt` (JSON Web Token):** 专门用于JWT管理，如活跃列表。
    -   示例: `jwt:active:{userID}:{jti}`
-   **`lk` (Lock):** 用于分布式锁。
    -   示例: `lk:resource:update_user_balance:{userID}`
-   **`cache` (Cache):** 通用缓存，例如缓存数据库查询结果。
    -   示例: `cache:db:user_profile:{userID}`
-   **`tmp` (Temporary):** 临时数据，具有较短生命周期。
    -   示例: `tmp:password_reset_token:{token}`
-   **`cnt` (Counter):** 计数器。
    -   示例: `cnt:api_request:login_attempts:ip:{ip_address}`
-   **`q` (Queue):** 如果使用Redis作为简单消息队列的元数据。
    -   示例: `q:task:pending:user_notification`

## 5. 具体示例 (结合开发手册)

### 5.1. 用户认证与设备 (参考手册 2.1)
-   **活跃JWT记录:**
    -   Key: `jwt:active:{userID}:{jti}`
    -   Type: STRING
    -   Value: `clientID` 或 `true`
    -   TTL: JWT有效期
    -   *说明: 用于主动管理活跃的JWT，替代或补充黑名单机制。*

### 5.2. 角色与客户端状态 (参考手册 3.2)
-   **用户角色:**
    -   Key: `sess:user_roles:{userID}`
    -   Type: HASH
    -   Fields: `transmitter_client_id`, `transmitter_set_at_utc`, `receiver_client_id`, `receiver_set_at_utc`
-   **客户端实时状态:**
    -   Key: `sts:client_state:{clientID}`
    -   Type: HASH
    -   Fields: `user_id`, `role`, `device_model`, `nfc_status_transmitter`, `hce_status_receiver`, etc.

### 5.3. 交易会话 (参考手册 6.1)
-   **交易会话详情:**
    -   Key: `sess:transaction_session:{transactionID}`
    -   Type: HASH
    -   Fields: `transmitter_client_id`, `receiver_client_id`, `status`, `created_at_utc`, etc.

## 6. Key命名审查

在定义新的Redis Key时，应进行审查，确保其符合本规范。团队成员应共同维护和更新此规范文档。

## 7. 注意事项

-   当`entity_id`本身已包含足够信息表明实体类型时，`entity_group` 可以适当省略以保持简洁。例如，如果`clientID`全局唯一且能清晰表明是客户端，`sts:client_state:{clientID}` 中可不再重复 `client`。但通常显式指明有助于理解。
-   对于集合类型的Key（如List, Set, Sorted Set），可以在Key名末尾添加类型提示，如 `:list`, `:set`, `:zset`，但这并非强制，只要Key本身能清晰表达。
-   时间戳相关的Key或Field，建议统一使用ISO 8601格式字符串，并在命名中明确（如 `_at_utc`）。

--- 