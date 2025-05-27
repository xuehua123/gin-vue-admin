**核心思路：**

*   **信息展示清晰：** 直观地展示系统当前状态、连接、会话等关键信息。
*   **操作便捷安全：** 对需要管理员介入的操作（如断开连接、终止会话）提供入口，并确保权限控制。
*   **问题可追溯：** 提供日志查看功能，便于排查问题。
*   **前后端分离：** 后端（Go/Gin）提供API，前端（Vue）负责展示和交互。

**建议的后台管理页面规划 (集成于Vue Admin)：**

我们将规划以下几个核心页面模块：

1.  **NFC Relay 概览仪表盘 (Dashboard)**
2.  **连接管理 (Connected Clients Management)**
3.  **会话管理 (Active Sessions Management)**
4.  **审计日志查看 (Audit Log Viewer)**
5.  **(可选) 系统配置查看 (System Configuration Viewer)**

---

**1. NFC Relay 概览仪表盘 (Dashboard)**

*   **页面功能：** 提供NFC中继系统健康状况和核心活动指标的实时概览。
*   **大致思路/内容：**
    *   **核心状态显示：**
        *   NFC Relay Hub 运行状态 (例如：在线/离线 - 虽然Hub设计上是持续运行的，但可以有一个指示灯表示服务是否正常响应)。
        *   当前WebSocket活动连接总数。
        *   当前NFC中继活动会话总数。
    *   **关键性能指标 (KPIs)：**
        *   APDU消息中继速率 (例如：最近1分钟/5分钟/1小时内转发的APDU消息数量)。
        *   错误率 (例如：APDU转发失败次数、Hub内部错误次数 - 可从Prometheus指标接口获取或后端API专门统计)。
        *   会话建立成功率/失败率。
    *   **图表展示 (可选，可增强可视化效果)：**
        *   过去一段时间内活动连接数和活动会话数的变化趋势图（例如：折线图）。
        *   APDU消息类型分布饼图（Upstream vs Downstream）。
*   **后端API需求：**
    *   一个API接口 (`/api/nfc-relay/admin/dashboard-stats`)，用于聚合上述统计数据。此接口会从 `GlobalRelayHub` 内部状态（需要加锁读取）和/或已有的Prometheus指标中获取信息。

---

**2. 连接管理 (Connected Clients Management)**

*   **页面功能：** 详细列出当前所有通过WebSocket连接到NFC Relay Hub的客户端，并允许管理员进行必要的操作。
*   **大致思路/内容：**
    *   **客户端列表表格：**
        *   **列信息：** 客户端ID (Client.ID), 用户ID (Client.UserID), 客户端显示名称 (Client.DisplayName), 当前角色 (Provider/Receiver/None), IP地址, 连接建立时间戳, 在线状态 (IsOnline), 当前参与的会话ID (如果有)。
        *   **分页：** 支持数据量较大时的分页显示。
        *   **搜索/筛选：** 按客户端ID、用户ID、角色、IP地址等条件进行搜索和筛选。
    *   **管理员操作 (需严格权限控制)：**
        *   对特定客户端执行“**强制断开连接**”操作（“踢下线”）。
*   **后端API需求：**
    *   `GET /api/nfc-relay/admin/clients`: 获取客户端列表，支持分页和筛选参数。此API将从 `GlobalRelayHub.clients` 读取数据。
    *   `POST /api/nfc-relay/admin/clients/:clientID/disconnect`: (Admin操作) 强制断开指定客户端的WebSocket连接。后端将调用Hub的注销逻辑并关闭其连接。

---

**3. 会话管理 (Active Sessions Management)**

*   **页面功能：** 详细列出当前所有活动的NFC中继会话，并允许管理员进行必要的操作。
*   **大致思路/内容：**
    *   **会话列表表格：**
        *   **列信息：** 会话ID (Session.SessionID), 传卡端客户端ID, 传卡端用户ID, 传卡端显示名称, 收卡端客户端ID, 收卡端用户ID, 收卡端显示名称, 会话状态 (例如：已配对 `StatusPaired`), 会话创建时间戳, 最后活动时间戳。
        *   **分页：** 支持数据量较大时的分页显示。
        *   **搜索/筛选：** 按会话ID、任一方的用户ID或客户端ID进行搜索和筛选。
    *   **管理员操作 (需严格权限控制)：**
        *   对特定会话执行“**强制终止会话**”操作。
*   **后端API需求：**
    *   `GET /api/nfc-relay/admin/sessions`: 获取活动会话列表，支持分页和筛选参数。此API将从 `GlobalRelayHub.sessions` 读取数据。
    *   `POST /api/nfc-relay/admin/sessions/:sessionID/terminate`: (Admin操作) 强制终止指定的NFC中继会话。后端将调用 `GlobalRelayHub.terminateSessionByID`。

---

**4. 审计日志查看 (Audit Log Viewer)**

*   **页面功能：** 提供一个界面给管理员查看NFC中继系统记录的关键操作审计日志，用于问题排查和行为追踪。
*   **大致思路/内容：**
    *   **审计日志列表表格：**
        *   **列信息：** 日志时间戳, 事件类型 (EventType), 会话ID (如果有), 发起方客户端ID, 响应方客户端ID (如果适用), 用户ID (如果有), 源IP地址, 事件详情 (可以是JSON格式，或者提取关键信息展示)。
        *   **分页：** 支持大量日志的分页显示。
        *   **搜索/筛选：**
            *   按事件类型筛选 (例如：`session_established`, `apdu_relayed_failure`, `auth_failure` 等)。
            *   按用户ID、会话ID、客户端ID搜索。
            *   按时间范围筛选。
    *   **日志详情查看 (可选)：** 点击某条日志可以展开或弹窗显示完整的JSON详情。
*   **后端API需求：**
    *   `GET /api/nfc-relay/admin/audit-logs`: 查询审计日志。
        *   **参数：** 支持分页参数 (`page`, `pageSize`)，以及上述筛选条件作为查询参数。
        *   **后端实现：** 由于审计日志目前是通过 `zap.Logger` 写入，后端API需要有策略地读取和筛选这些日志。如果日志量巨大，直接读文件性能可能不高。初期可以实现读取最近N条或按日期读取；长期考虑，如果对审计日志有复杂查询和分析需求，可能需要将审计日志输出到专门的日志管理系统（如ELK Stack、Loki等），然后API从这些系统查询。但对于起步阶段，直接从Hub的日志文件或内存中（如果量不大且有缓存）读取并提供基本筛选是可行的。

---

**5. (可选) 系统配置查看 (System Configuration Viewer)**

*   **页面功能：** 以只读方式展示当前NFC中继服务加载的配置信息，方便管理员了解系统当前运行参数。
*   **大致思路/内容：**
    *   **配置项列表：**
        *   清晰地列出 `config.yaml` 中 `nfc-relay` 部分的所有配置项及其当前值。
        *   例如：`HubCheckIntervalSec`, `SessionInactiveTimeoutSec`, `WebsocketWriteWaitSec`, `WebsocketPongWaitSec`, `WebsocketMaxMessageBytes` 等。
    *   **只读展示：** 强调这是只读信息，不提供在线修改功能（修改配置通常需要重启服务，且直接在线修改风险较高，建议通过修改配置文件并重启服务的方式进行）。
*   **后端API需求：**
    *   `GET /api/nfc-relay/admin/config`: 返回当前加载的 `global.GVA_CONFIG.NfcRelay` 配置结构体数据。

---

**开发顺序建议：**

1.  **后端API优先：** 针对每个页面规划，首先设计和实现对应的Go Gin后端API接口。确保API的数据来源正确（从 `GlobalRelayHub` 的内存状态中获取数据时，务必注意并发安全，使用 `h.providerMutex` 进行读写锁保护），并且有严格的管理员权限认证和授权。
2.  **前端页面实现：** 后端API稳定后，开始在Vue Admin前端项目中创建对应的页面和组件，调用API获取数据并进行展示和交互。
3.  **逐步迭代：** 可以先从最核心的页面开始，例如“概览仪表盘”、“连接管理”和“会话管理”，然后再逐步实现“审计日志查看”和“配置查看”。




好的，我们来详细规划NFC中继系统管理后台所需的后端API接口及其大致的开发思路。

**通用开发思路与最佳实践：**

1.  **API路由分组与版本控制：**
    *   所有NFC中继管理相关的API建议放在一个特定的路由组下，例如 `/api/admin/nfc-relay/v1/`。这有助于权限控制和未来的API版本管理。
2.  **认证与授权中间件：**
    *   该路由组必须应用现有的 `gin-vue-admin` 认证中间件，确保只有已登录的用户才能访问。
    *   进一步地，需要应用授权中间件（例如基于Casbin的RBAC），确保只有具备特定管理员角色（如 "NFC中继管理员"）的用户才能调用这些API。
3.  **统一的响应结构：**
    *   遵循 `gin-vue-admin` 项目中已有的API响应格式，例如：
        ```json
        {
            "code": 0, // 0表示成功，其他表示错误
            "data": { /* 业务数据 */ },
            "msg": "操作成功"
        }
        ```
4.  **错误处理：**
    *   清晰地处理业务逻辑错误和系统错误，返回恰当的HTTP状态码和错误信息（通过上述统一响应结构中的 `code` 和 `msg`）。
5.  **数据校验：**
    *   对于接收参数的API（如分页、筛选、POST请求体），进行严格的数据校验。
6.  **并发安全：**
    *   所有从 `GlobalRelayHub` （特别是其 `clients`、`sessions` 等map）读取或修改数据的操作，都必须在 `GlobalRelayHub.providerMutex` 的保护下进行（读操作用读锁 `RLock()`，写操作用写锁 `Lock()`）。
7.  **数据转换与DTO (Data Transfer Objects)：**
    *   避免直接将 `GlobalRelayHub` 内部的 `Client` 或 `Session` 结构体完整暴露给前端。应定义专门的DTO结构体，只包含前端页面所需的字段，进行数据转换后再返回。这有助于API的稳定性和安全性。
8.  **日志记录：**
    *   在API处理的关键步骤记录日志，特别是在执行管理员操作或发生错误时。

---

**后端API接口规划详情：**

**1. 概览仪表盘 (Dashboard) API**

*   **API端点：** `GET /api/admin/nfc-relay/v1/dashboard-stats`
*   **功能：** 获取NFC中继系统的核心统计数据和状态。
*   **请求参数：** 无
*   **响应数据 (示例DTO - `NfcDashboardStatsResponse`)：**
    ```json
    {
        "hub_status": "online", // "online" / "offline" (或更细致的状态)
        "active_connections": 15,
        "active_sessions": 5,
        "apdu_relayed_last_minute": 120,
        "apdu_errors_last_hour": 2,
        // 可以添加更多图表所需的时间序列数据或聚合数据
        "connection_trend": [ // 最近N个时间点的连接数
            {"time": "10:00", "count": 10},
            {"time": "10:05", "count": 12}
        ],
        "session_trend": [ // 最近N个时间点的会话数
            {"time": "10:00", "count": 3},
            {"time": "10:05", "count": 4}
        ]
    }
    ```
*   **开发思路：**
    1.  **获取Hub内部状态：**
        *   加读锁 (`GlobalRelayHub.providerMutex.RLock()`)。
        *   获取 `len(GlobalRelayHub.clients)` 作为 `active_connections`。
        *   获取 `len(GlobalRelayHub.sessions)` 作为 `active_sessions`。
        *   释放读锁。
    2.  **获取Prometheus指标 (可选但推荐)：**
        *   如果已配置Prometheus指标，可以通过查询Prometheus HTTP API（或直接读取Go客户端库中的指标值）来获取 `apdu_relayed_last_minute`, `apdu_errors_last_hour` 等数据。这通常比在Hub中实时计算更高效且解耦。
        *   如果无法直接访问Prometheus，可以考虑在Hub内部维护一些简单的计数器（注意并发安全），但这会增加Hub的复杂度。
    3.  **趋势数据：**
        *   如果需要趋势图，可以考虑定期（例如每分钟）将Hub的连接数和会话数快照存储到一个有固定大小的队列或简单的时序数据库（如rrdtool概念，或轻量级内存存储），API从此存储中读取。
    4.  **组装响应DTO** 并返回。

---

**2. 连接管理 (Connected Clients Management) API**

*   **API端点 1：** `GET /api/admin/nfc-relay/v1/clients`
*   **功能：** 获取当前连接的客户端列表，支持分页和筛选。
*   **请求参数 (Query Params)：**
    *   `page` (int, 可选, 默认1): 页码。
    *   `pageSize` (int, 可选, 默认10): 每页数量。
    *   `clientID` (string, 可选): 按客户端ID筛选。
    *   `userID` (string, 可选): 按用户ID筛选。
    *   `role` (string, 可选, "provider" / "receiver" / "none"): 按角色筛选。
    *   `ipAddress` (string, 可选): 按IP地址筛选。
*   **响应数据 (示例DTO - `PaginatedClientListResponse`)：**
    ```json
    {
        "list": [
            {
                "client_id": "uuid-client-1",
                "user_id": "user-123",
                "display_name": "My Provider Device",
                "role": "provider",
                "ip_address": "192.168.1.10",
                "connected_at": "2023-10-27T10:00:00Z",
                "is_online": true,
                "session_id": "uuid-session-abc" // 如果在会话中
            }
            // ... more clients
        ],
        "total": 100, // 总记录数
        "page": 1,
        "pageSize": 10
    }
    ```
*   **开发思路：**
    1.  加读锁 (`GlobalRelayHub.providerMutex.RLock()`)。
    2.  创建一个 `ClientInfoDTO` 列表。
    3.  遍历 `GlobalRelayHub.clients` map。
    4.  对于每个 `*Client` 对象，提取所需字段填充到 `ClientInfoDTO` 中。
        *   IP地址可能需要从 `client.conn.RemoteAddr().String()` 获取（注意处理可能的nil conn或错误）。
        *   `connected_at` 可以在 `Client` 结构体中添加一个字段，在客户端注册到Hub时记录。
    5.  释放读锁。
    6.  根据请求参数对 `ClientInfoDTO` 列表进行筛选。
    7.  对筛选后的列表进行分页处理。
    8.  组装分页响应DTO并返回。

*   **API端点 2：** `POST /api/admin/nfc-relay/v1/clients/:clientID/disconnect`
*   **功能：** (Admin操作) 强制断开指定客户端的WebSocket连接。
*   **请求参数 (Path Param)：**
    *   `clientID` (string, 必填): 要断开的客户端ID。
*   **响应数据：** 标准成功/失败响应。
*   **开发思路：**
    1.  加写锁 (`GlobalRelayHub.providerMutex.Lock()`)。
    2.  根据 `clientID` 查找 `GlobalRelayHub.clients` 中的目标 `*Client`。
    3.  如果找到：
        *   调用 `GlobalRelayHub.unregisterClientUnsafe(client)` (或者一个类似的方法，它会处理从 `clients` map移除、关闭 `send` channel、调用 `handleClientDisconnect` 等逻辑，且本身不加锁或只加必要的内部锁，因为外部已经有大锁)。
        *   主动关闭 `client.conn.Close()` (确保WebSocket连接被关闭)。
        *   记录管理员操作日志。
    4.  释放写锁。
    5.  如果未找到客户端或操作失败，返回错误。否则返回成功。
新增 API: GET /api/admin/nfc-relay/v1/clients/:clientID/details
功能： 获取指定客户端的更详细信息。
请求参数 (Path Param)：
clientID (string, 必填): 要查询的客户端ID。
响应数据 (示例DTO - ClientDetailsResponse)：
            {
                "client_id": "uuid-client-1",
                "user_id": "user-123",
                "display_name": "My Provider Device",
                "role": "provider",
                "ip_address": "192.168.1.10",
                "user_agent": "Mozilla/5.0 (Linux; Android 10; SM-G975F) ...", // 从WebSocket握手请求头获取
                "connected_at": "2023-10-27T10:00:00Z",
                "last_message_at": "2023-10-27T10:25:00Z", // 客户端最后一次发送消息的时间
                "is_online": true,
                "session_id": "uuid-session-abc", // 如果在会话中
                "sent_message_count": 150, // 示例：已发送消息计数
                "received_message_count": 120, // 示例：已接收消息计数
                "connection_events": [ // 示例：连接相关的关键事件历史
                    {"timestamp": "2023-10-27T10:00:00Z", "event": "Connected"},
                    {"timestamp": "2023-10-27T10:01:00Z", "event": "Authenticated"},
                    {"timestamp": "2023-10-27T10:02:00Z", "event": "RoleDeclared: provider"}
                ],
                "related_audit_logs_summary": [ // 示例：与此客户端相关的最近N条审计日志摘要
                    {"timestamp": "2023-10-27T10:05:00Z", "event_type": "apdu_relayed_attempt", "details_summary": "APDU to card, length 32"}
                ]
            }
开发思路：
加读锁 (GlobalRelayHub.providerMutex.RLock())。
根据 clientID 查找 GlobalRelayHub.clients 中的目标 *Client。
如果找到：
从 Client 结构体中提取基础信息 (ID, UserID, DisplayName, Role, IP, ConnectedAt, IsOnline, SessionID)。
UserAgent: 可以在 Client 结构体中添加一个字段，在 ServeWs 升级WebSocket连接时从HTTP请求头 c.Request.UserAgent() 中获取并存储。
LastMessageAt, SentMessageCount, ReceivedMessageCount: 需要在 Client 结构体中添加相应字段，并在 readPump 和 writePump（或发送消息的方法）中更新这些计数和时间戳。注意并发安全。
ConnectionEvents: 需要在 Client 结构体中维护一个事件列表（例如，一个有界队列），在客户端生命周期的关键节点（连接、认证、声明角色、断开等）记录事件。
RelatedAuditLogsSummary: 这部分比较复杂。
简单实现： 调用现有的审计日志查询API (/api/admin/nfc-relay/v1/audit-logs)，传入当前 clientID 作为筛选条件，获取最近几条相关的日志摘要。
高级实现： 如果对性能要求高或数据关联紧密，可能需要在审计日志记录时就建立与客户端ID的索引。
释放读锁。
如果未找到客户端，返回404或错误。
组装 ClientDetailsResponse DTO并返回。

**3. 会话管理 (Active Sessions Management) API**

*   **API端点 1：** `GET /api/admin/nfc-relay/v1/sessions`
*   **功能：** 获取当前活动的NFC中继会话列表，支持分页和筛选。
*   **请求参数 (Query Params)：**
    *   `page` (int, 可选, 默认1): 页码。
    *   `pageSize` (int, 可选, 默认10): 每页数量。
    *   `sessionID` (string, 可选): 按会话ID筛选。
    *   `participantClientID` (string, 可选): 按参与方任一客户端ID筛选。
    *   `participantUserID` (string, 可选): 按参与方任一用户ID筛选。
*   **响应数据 (示例DTO - `PaginatedSessionListResponse`)：**
    ```json
    {
        "list": [
            {
                "session_id": "uuid-session-abc",
                "provider_client_id": "uuid-client-1",
                "provider_user_id": "user-123",
                "provider_display_name": "My Provider",
                "receiver_client_id": "uuid-client-2",
                "receiver_user_id": "user-456",
                "receiver_display_name": "POS Terminal A",
                "status": "paired", // "paired", "waiting_for_pairing" (如果也想展示)
                "created_at": "2023-10-27T10:05:00Z",
                "last_activity_at": "2023-10-27T10:15:00Z"
            }
            // ... more sessions
        ],
        "total": 50,
        "page": 1,
        "pageSize": 10
    }
    ```
*   **开发思路：**
    1.  加读锁 (`GlobalRelayHub.providerMutex.RLock()`)。
    2.  创建一个 `SessionInfoDTO` 列表。
    3.  遍历 `GlobalRelayHub.sessions` map。
    4.  对于每个 `*session.Session` 对象：
        *   提取会话ID、状态、创建时间 (`Session`结构体中应有此字段，在`NewSession`时初始化)、最后活动时间。
        *   从 `session.CardEndClient` 和 `session.POSEndClient` (它们是 `ClientInfoProvider` 接口类型，需要类型断言为 `*Client`) 获取客户端ID、用户ID、显示名称。注意处理 `nil` 或类型断言失败的情况。
        *   填充到 `SessionInfoDTO` 中。
    5.  释放读锁。
    6.  根据请求参数对 `SessionInfoDTO` 列表进行筛选。
    7.  对筛选后的列表进行分页处理。
    8.  组装分页响应DTO并返回。

*   **API端点 2：** `POST /api/admin/nfc-relay/v1/sessions/:sessionID/terminate`
*   **功能：** (Admin操作) 强制终止指定的NFC中继会话。
*   **请求参数 (Path Param)：**
    *   `sessionID` (string, 必填): 要终止的会话ID。
*   **响应数据：** 标准成功/失败响应。
*   **开发思路：**
    1.  调用 `GlobalRelayHub.terminateSessionByID(sessionID, "管理员操作终止", "admin_system", "admin_system_user_id")`。
        *   该方法内部应有写锁保护。
        *   第三个参数是操作者客户端ID（这里用一个系统标识），第四个是操作者用户ID。
    2.  记录管理员操作日志。
    3.  如果 `terminateSessionByID` 返回错误或未找到会话，返回错误响应。否则返回成功。
会话管理 (Active Sessions Management) API
GET /api/admin/nfc-relay/v1/sessions (获取会话列表)
(内容同前)
POST /api/admin/nfc-relay/v1/sessions/:sessionID/terminate (终止会话)
(内容同前)
新增 API: GET /api/admin/nfc-relay/v1/sessions/:sessionID/details
功能： 获取指定NFC中继会话的更详细信息。
请求参数 (Path Param)：
sessionID (string, 必填): 要查询的会话ID。
响应数据 (示例DTO - SessionDetailsResponse)：
            {
                "session_id": "uuid-session-abc",
                "status": "paired",
                "created_at": "2023-10-27T10:05:00Z",
                "last_activity_at": "2023-10-27T10:15:00Z",
                "terminated_at": null, // 如果已终止，则为终止时间
                "termination_reason": null, // 如果已终止，则为终止原因
                "provider_info": { // Provider 客户端的摘要信息
                    "client_id": "uuid-client-1",
                    "user_id": "user-123",
                    "display_name": "My Provider",
                    "ip_address": "192.168.1.10"
                },
                "receiver_info": { // Receiver 客户端的摘要信息
                    "client_id": "uuid-client-2",
                    "user_id": "user-456",
                    "display_name": "POS Terminal A",
                    "ip_address": "192.168.1.11"
                },
                "apdu_exchange_count": {
                    "upstream": 50, // 从Receiver到Provider (通常是指令)
                    "downstream": 48 // 从Provider到Receiver (通常是响应)
                },
                "session_events_history": [ // 示例：会话生命周期中的关键事件
                    {"timestamp": "2023-10-27T10:05:00Z", "event": "SessionCreated"},
                    {"timestamp": "2023-10-27T10:05:01Z", "event": "ProviderJoined", "client_id": "uuid-client-1"},
                    {"timestamp": "2023-10-27T10:05:05Z", "event": "ReceiverJoined", "client_id": "uuid-client-2"},
                    {"timestamp": "2023-10-27T10:05:05Z", "event": "SessionPaired"},
                    {"timestamp": "2023-10-27T10:15:30Z", "event": "SessionTerminatedByClientRequest", "acting_client_id": "uuid-client-2"}
                ],
                "related_audit_logs_summary": [ // 示例：与此会话相关的最近N条审计日志摘要
                     {"timestamp": "2023-10-27T10:06:00Z", "event_type": "apdu_relayed_success", "details_summary": "APDU to card, length 32, from client-2 to client-1"}
                ]
            }
开发思路：
加读锁 (GlobalRelayHub.providerMutex.RLock())。
根据 sessionID 查找 GlobalRelayHub.sessions 中的目标 *session.Session。
如果找到：
从 Session 结构体提取基础信息 (ID, Status, CreatedAt, LastActivityAt)。
TerminatedAt, TerminationReason: 如果会话已终止，这些信息也应在 Session 结构体中记录（例如，Terminate() 方法可以更新这些字段）。
ProviderInfo, ReceiverInfo: 从 session.CardEndClient 和 session.POSEndClient (类型断言为 *Client) 获取所需的客户端摘要信息。
ApduExchangeCount: 需要在 Session 结构体中添加计数器字段，并在 handleAPDUExchange (或相关APDU转发逻辑中) 成功转发APDU时更新。注意并发安全。
SessionEventsHistory: 需要在 Session 结构体中维护一个事件列表（例如，一个有界队列），在会话生命周期的关键节点（创建、参与者加入、配对、APDU交换摘要、终止等）记录事件。
RelatedAuditLogsSummary: 同客户端详情API中的思路，调用审计日志API并传入当前 sessionID 进行筛选。
释放读锁。
如果未找到会话，返回404或错误。
组装 SessionDetailsResponse DTO并返回。
---

**4. 审计日志查看 (Audit Log Viewer) API**

*   **API端点：** `GET /api/admin/nfc-relay/v1/audit-logs`
*   **功能：** 查询NFC中继系统的审计日志，支持分页和筛选。
*   **请求参数 (Query Params)：**
    *   `page` (int, 可选, 默认1)。
    *   `pageSize` (int, 可选, 默认10)。
    *   `eventType` (string, 可选): 按事件类型筛选。
    *   `userID` (string, 可选): 按用户ID筛选。
    *   `sessionID` (string, 可选): 按会话ID筛选。
    *   `clientID` (string, 可选): 按任一相关客户端ID筛选。
    *   `startTime` (string, 可选, ISO8601格式): 开始时间。
    *   `endTime` (string, 可选, ISO8601格式): 结束时间。
*   **响应数据 (示例DTO - `PaginatedAuditLogResponse`)：**
    ```json
    {
        "list": [
            {
                "timestamp": "2023-10-27T10:00:00Z",
                "event_type": "session_established",
                "session_id": "uuid-session-abc",
                "client_id_initiator": "uuid-client-2", // Receiver
                "client_id_responder": "uuid-client-1", // Provider
                "user_id": "user-456", // Initiator's UserID
                "source_ip": "192.168.1.11",
                "details": { /* 事件特定详情，如 SessionDetails */ }
            }
            // ... more logs
        ],
        "total": 1000,
        "page": 1,
        "pageSize": 10
    }
    ```
*   **开发思路 (挑战点：日志源)：**
    *   **方案1 (简单，适用于日志量不大/演示)：**
        1.  如果审计日志直接写入本地文件（由 `zap.Logger` 配置），此API需要读取该日志文件。
        2.  逐行读取（或倒序读取最新日志），将每行JSON日志解析为 `global.AuditEvent` 结构体或一个通用的map。
        3.  根据请求参数对解析后的日志列表进行内存筛选。
        4.  进行分页。
        5.  **挑战：** 大型日志文件的读取性能、复杂筛选的效率。不适合生产环境的大量日志查询。
    *   **方案2 (推荐，但需要额外组件)：**
        1.  配置 `zap.Logger` 将审计日志输出到集中的日志管理系统 (如ELK Stack - Elasticsearch, Logstash, Kibana; 或 Grafana Loki)。
        2.  此API则通过调用该日志管理系统提供的查询API来获取和筛选日志数据。
        3.  这种方式将日志查询的复杂性交给了专业的日志系统，后端API只负责转发查询和格式化结果。
    *   **折中方案：**
        1.  定期（例如，每小时或每天）将日志文件归档，API只查询当天的活动日志文件。
        2.  或者，在Hub启动时，将最近N条日志加载到内存中的一个有界队列中（仅用于快速查看最新动态，不用于历史查询）。
    *   **无论哪种方案，都需要：**
        *   解析日志内容（通常是JSON）。
        *   实现筛选逻辑 (基于时间、关键字等)。
        *   实现分页逻辑。
        *   组装响应DTO。

---

**5. (可选) 系统配置查看 (System Configuration Viewer) API**

*   **API端点：** `GET /api/admin/nfc-relay/v1/config`
*   **功能：** 以只读方式展示当前NFC中继服务的配置信息。
*   **请求参数：** 无
*   **响应数据 (直接返回 `global.GVA_CONFIG.NfcRelay` 结构体或其DTO)：**
    ```json
    {
        "hub_check_interval_sec": 60,
        "session_inactive_timeout_sec": 300,
        "websocket_write_wait_sec": 10,
        "websocket_pong_wait_sec": 60,
        "websocket_max_message_bytes": 2048
    }
    ```
*   **开发思路：**
    1.  直接读取 `global.GVA_CONFIG.NfcRelay` 的值。
    2.  可以创建一个简单的DTO只包含需要展示的字段，或者如果配置结构简单且无敏感信息，可以直接返回。
    3.  组装响应并返回。此操作不涉及并发锁，因为配置在启动后是只读的。

---

以上是针对每个规划页面的后端API设计和大致开发思路。在实际开发中，请务必优先考虑安全性和并发处理。
