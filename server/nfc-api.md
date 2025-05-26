# NFC 中继服务 - WebSocket API 文档

## 1. 概述

本文档定义了 NFC 中继服务的 WebSocket API。客户端（如 Android App）可以通过此 API 与后端服务器进行通信，以实现NFC卡的远程读写能力。

**通信协议:** WebSocket (WS)
**消息格式:** JSON

## 2. 连接端点

客户端应连接到以下 WebSocket 端点:

`ws://<your_server_address>:<port>/nfcrelay/ws`

*   `<your_server_address>`: 您的服务器域名或 IP 地址。
*   `<port>`: 您的服务器运行的端口。
*   路径 `/nfcrelay/ws` 是在 `backend/router/nfc_relay.go` 中定义的。

## 3. 消息结构

所有消息都遵循一个基础结构，包含 `type` 字段来区分不同的消息类型。

**基础消息结构 (`protocol.GenericMessage`):**

```json
{
  "type": "message_type_string",
  "seq": 12345 // 可选的消息序列号，用于追踪和调试
}
```

## 4. 核心流程与消息详解

### Phase A: 客户端认证

客户端连接成功后，必须发送的第一条消息是认证消息。只有认证通过的客户端才能执行后续操作。

**4.1. 客户端发送认证请求 (`client_auth`)**

*   **消息类型:** `protocol.MessageTypeClientAuth` ("client_auth")
*   **方向:** Client -> Server
*   **结构 (`protocol.ClientAuthMessage`):**

    ```json
    {
      "type": "client_auth",
      "token": "YOUR_JWT_TOKEN_STRING"
    }
    ```
*   **字段说明:**
    *   `token`: 从用户登录系统获取的有效 JWT (JSON Web Token)。

**4.2. 服务器响应认证结果 (`server_auth_response`)**

*   **消息类型:** `protocol.MessageTypeServerAuthResponse` ("server_auth_response")
*   **方向:** Server -> Client
*   **结构 (`protocol.ServerAuthResponseMessage`):**

    *   认证成功:
        ```json
        {
          "type": "server_auth_response",
          "success": true,
          "userId": "USER_ID_FROM_TOKEN",
          "message": "Authentication successful"
        }
        ```
    *   认证失败:
        ```json
        {
          "type": "server_auth_response",
          "success": false,
          "message": "Authentication failed: Token has expired" // 或其他错误信息
        }
        ```
*   **字段说明:**
    *   `success`: `true` 表示认证成功，`false` 表示失败。
    *   `userId`: 认证成功时，返回从 JWT Token 中解析出的用户唯一标识符。
    *   `message`: 提供认证结果的描述信息。

---

### Phase B: 角色声明

认证成功后，客户端需要声明其角色（发卡方或收卡方）以及在线状态。

**4.3. 客户端声明角色和状态 (`declare_role`)**

*   **消息类型:** `protocol.MessageTypeDeclareRole` ("declare_role")
*   **方向:** Client -> Server
*   **结构 (`protocol.DeclareRoleMessage`):**

    ```json
    {
      "type": "declare_role",
      "role": "provider", // 或 "receiver", "none"
      "online": true,     // 对于 provider: true=上线服务, false=下线; 对于 receiver: 可忽略或用于表示主动寻找状态
      "providerName": "My NFC Card Reader" // 可选, 当 role="provider" 时，客户端可提供一个显示名称
    }
    ```
*   **字段说明:**
    *   `role` (`protocol.RoleType`):
        *   `protocol.RoleProvider` ("provider"): 声明为发卡方 (NFC读卡器端)。
        *   `protocol.RoleReceiver` ("receiver"): 声明为收卡方 (需要使用NFC卡片的应用端)。
        *   `protocol.RoleNone` ("none"): 清除角色或下线。
    *   `online`: 布尔值。
        *   对于 "provider": `true` 表示上线并提供服务，`false` 表示下线。
        *   对于 "receiver": 此字段当前主要由服务器内部参考，客户端可根据需要设置。
    *   `providerName`: 可选字符串。当 `role` 为 "provider" 时，客户端可以提供一个自定义的显示名称，方便收卡方识别。如果未提供，服务器可能会使用默认名称。

**4.4. 服务器响应角色声明结果 (`role_declared_response`)**

*   **消息类型:** `protocol.MessageTypeRoleDeclaredResponse` ("role_declared_response")
*   **方向:** Server -> Client
*   **结构 (`protocol.RoleDeclaredResponseMessage`):**

    ```json
    {
      "type": "role_declared_response",
      "success": true,
      "role": "provider", // 服务器确认的角色
      "online": true,    // 服务器确认的在线状态
      "message": "您已成功声明为发卡方并上线" // 或其他相关消息
    }
    ```
    *   如果失败:
        ```json
        {
          "type": "role_declared_response",
          "success": false,
          "message": "无效的角色声明"
        }
        ```
*   **字段说明:**
    *   `success`: `true` 表示角色声明成功，`false` 表示失败。
    *   `role`: 服务器确认并记录的客户端角色。
    *   `online`: 服务器确认并记录的客户端在线状态。
    *   `message`: 提供声明结果的描述信息。

---

### Phase C: 服务发现 (Receiver 客户端)

声明为 "receiver" 的客户端可以请求其账户下当前可用的 "provider" 列表。

**4.5. Receiver 请求可用发卡方列表 (`list_card_providers`)**

*   **消息类型:** `protocol.MessageTypeListCardProviders` ("list_card_providers")
*   **方向:** Client (Receiver) -> Server
*   **结构 (`protocol.ListCardProvidersMessage`):**

    ```json
    {
      "type": "list_card_providers"
      // 未来可以添加过滤条件，例如按特定用户ID (如果业务允许跨用户)
    }
    ```

**4.6. 服务器推送/响应发卡方列表 (`card_providers_list`)**

*   **消息类型:** `protocol.MessageTypeCardProvidersList` ("card_providers_list")
*   **方向:** Server -> Client (Receiver)
*   **触发时机:**
    1.  当 Receiver 发送 `list_card_providers` 请求时。
    2.  当 Receiver 订阅的 UserID 下的发卡方列表状态发生变化时（例如，新的 Provider 上线、Provider 下线、Provider 变忙/变空闲），服务器会主动推送更新后的列表给所有订阅了该 UserID 的 Receiver 客户端。
*   **结构 (`protocol.CardProvidersListMessage`):**

    ```json
    {
      "type": "card_providers_list",
      "providers": [
        {
          "providerId": "PROVIDER_CLIENT_UNIQUE_ID_1",
          "providerName": "Card Reader Alpha",
          "userId": "USER_ID_OF_PROVIDER_1",
          "isBusy": false
        },
        {
          "providerId": "PROVIDER_CLIENT_UNIQUE_ID_2",
          "providerName": "NFC Dongle Office",
          "userId": "USER_ID_OF_PROVIDER_2",
          "isBusy": true
        }
        // ... more providers
      ]
    }
    ```
*   **字段说明:**
    *   `providers`: 一个 `protocol.CardProviderInfo` 对象数组。
        *   `providerId`: 发卡方客户端的唯一ID。
        *   `providerName`: 发卡方客户端的显示名称。
        *   `userId`: 该发卡方所属用户的ID (Receiver 只能看到自己 UserID 下的 Provider)。
        *   `isBusy`: 布尔值，`true` 表示该发卡方当前正忙于一个会话，`false` 表示空闲。

---

### Phase D: 会话建立 (Receiver 选择 Provider)

Receiver 客户端从列表中选择一个空闲的 Provider，并请求与其建立一个NFC中继会话。

**4.7. Receiver 选择发卡方并请求连接 (`select_card_provider`)**

*   **消息类型:** `protocol.MessageTypeSelectCardProvider` ("select_card_provider")
*   **方向:** Client (Receiver) -> Server
*   **结构 (`protocol.SelectCardProviderMessage`):**

    ```json
    {
      "type": "select_card_provider",
      "providerId": "PROVIDER_CLIENT_UNIQUE_ID_TO_CONNECT"
    }
    ```
*   **字段说明:**
    *   `providerId`: Receiver 希望连接的目标 Provider 的唯一ID (从 `card_providers_list` 中获取)。

**4.8. 服务器通知会话已建立 (`session_established`)**

*   **消息类型:** `protocol.MessageTypeSessionEstablished` ("session_established")
*   **方向:** Server -> Client (通知 Receiver 和被选择的 Provider)
*   **触发时机:** 当服务器成功为 Receiver 和 Provider 建立会话后。
*   **结构 (`protocol.SessionEstablishedMessage`):**

    ```json
    {
      "type": "session_established",
      "sessionId": "UNIQUE_SESSION_ID_GENERATED_BY_SERVER",
      "peerId": "ID_OF_THE_OTHER_CLIENT_IN_SESSION",
      "peerRole": "provider" // 对端客户端的角色 (对Receiver来说是"provider", 对Provider来说是"receiver")
    }
    ```
*   **字段说明:**
    *   `sessionId`: 本次成功建立的会话的唯一ID。后续APDU交换等操作需携带此ID。
    *   `peerId`: 会话中对端客户端的唯一ID。
    *   `peerRole` (`protocol.RoleType`): 会话中对端客户端的角色。

**4.9. 服务器通知会话建立失败 (`session_failed`)**

*   **消息类型:** `protocol.MessageTypeSessionFailed` ("session_failed")
*   **方向:** Server -> Client (通常是发起请求的 Receiver)
*   **触发时机:** 当会话建立请求因故失败时（例如，目标Provider不存在、忙碌、非同一用户、请求者已在会话中等）。
*   **结构 (`protocol.SessionFailedMessage`):**

    ```json
    {
      "type": "session_failed",
      "targetProviderId": "ATTEMPTED_PROVIDER_ID", // 尝试连接的发卡方ID
      "reason": "选择的发卡方当前正忙。" // 失败原因描述
    }
    ```
*   **字段说明:**
    *   `targetProviderId`: 客户端尝试连接的发卡方的ID。
    *   `reason`: 会话建立失败的具体原因。

---

### Phase E: APDU 消息中继

会话建立后，Receiver 和 Provider 可以通过服务器中继 APDU (Application Protocol Data Unit) 命令和响应。

**4.10. Receiver 发送 APDU 给 Provider (`apdu_upstream`)**

*   **消息类型:** `protocol.MessageTypeAPDUUpstream` ("apdu_upstream")
*   **方向:** Client (Receiver) -> Server
*   **结构 (`protocol.APDUUpstreamMessage`):**

    ```json
    {
      "type": "apdu_upstream",
      "sessionId": "ACTIVE_SESSION_ID",
      "apdu": "BASE64_ENCODED_APDU_COMMAND_STRING"
    }
    ```
*   **字段说明:**
    *   `sessionId`: 当前活动的会话ID。
    *   `apdu`: Base64 编码的 APDU 命令字符串。

**4.11. 服务器转发 APDU 给 Provider (`apdu_to_card`)**

*   **消息类型:** `protocol.MessageTypeAPDUToCard` ("apdu_to_card")
*   **方向:** Server -> Client (Provider)
*   **结构 (`protocol.APDUToCardMessage`):**

    ```json
    {
      "type": "apdu_to_card",
      "sessionId": "ACTIVE_SESSION_ID",
      "apdu": "BASE64_ENCODED_APDU_COMMAND_STRING" // 与 apdu_upstream 中的 APDU 相同
    }
    ```
*   **字段说明:**
    *   `sessionId`: 当前活动的会话ID。
    *   `apdu`: 从 Receiver 端收到的 Base64 编码的 APDU 命令。Provider 端接收后应解码并发送给物理NFC卡。

**4.12. Provider 发送 APDU 响应给 Receiver (`apdu_downstream`)**

*   **消息类型:** `protocol.MessageTypeAPDUDownstream` ("apdu_downstream")
*   **方向:** Client (Provider) -> Server
*   **结构 (`protocol.APDUDownstreamMessage`):**

    ```json
    {
      "type": "apdu_downstream",
      "sessionId": "ACTIVE_SESSION_ID",
      "apdu": "BASE64_ENCODED_APDU_RESPONSE_STRING_FROM_CARD"
    }
    ```
*   **字段说明:**
    *   `sessionId`: 当前活动的会话ID。
    *   `apdu`: 从物理NFC卡获取到的 APDU 响应，经过 Base64 编码。

**4.13. 服务器转发 APDU 响应给 Receiver (`apdu_from_card`)**

*   **消息类型:** `protocol.MessageTypeAPDUFromCard` ("apdu_from_card")
*   **方向:** Server -> Client (Receiver)
*   **结构 (`protocol.ServerAPDUFromCardMessage`):**

    ```json
    {
      "type": "apdu_from_card",
      "sessionId": "ACTIVE_SESSION_ID",
      "apdu": "BASE64_ENCODED_APDU_RESPONSE_STRING_FROM_CARD" // 与 apdu_downstream 中的 APDU 相同
    }
    ```
*   **字段说明:**
    *   `sessionId`: 当前活动的会话ID。
    *   `apdu`: 从 Provider 端收到的 Base64 编码的 APDU 响应。Receiver 端接收后应解码并处理。

---

### Phase F: 会话结束

会话中的任一客户端都可以主动请求结束当前会话。客户端断开 WebSocket 连接也会导致其参与的会话被动结束。

**4.14. 客户端请求结束会话 (`end_session`)**

*   **消息类型:** `protocol.MessageTypeEndSession` ("end_session")
*   **方向:** Client (Receiver 或 Provider) -> Server
*   **结构 (`protocol.EndSessionMessage`):**

    ```json
    {
      "type": "end_session",
      "sessionId": "SESSION_ID_TO_TERMINATE"
    }
    ```
*   **字段说明:**
    *   `sessionId`: 要结束的会话的ID。客户端应确保这是自己当前参与的会话。

**4.15. 服务器确认会话已终止 (`session_terminated`)**

*   **消息类型:** `protocol.MessageTypeSessionTerminated` ("session_terminated")
*   **方向:** Server -> Client (发送 `end_session` 的客户端)
*   **结构 (`protocol.SessionTerminatedMessage`):**

    ```json
    {
      "type": "session_terminated",
      "sessionId": "TERMINATED_SESSION_ID",
      "reason": "您已成功结束会话。"
    }
    ```
*   **字段说明:**
    *   `sessionId`: 已终止的会话ID。
    *   `reason`: 终止原因。

**4.16. 服务器通知对端会话已断开 (`peer_disconnected`)**

*   **消息类型:** `protocol.MessageTypePeerDisconnected` ("peer_disconnected")
*   **方向:** Server -> Client (会话中的另一方)
*   **触发时机:** 当一个客户端断开连接或主动结束会话时，通知其会话对端。
*   **结构 (`protocol.PeerDisconnectedMessage`):**

    ```json
    {
      "type": "peer_disconnected",
      "sessionId": "TERMINATED_SESSION_ID",
      "reason": "会话已由通信伙伴主动结束。" // 或 "对端连接已断开"
    }
    ```
*   **字段说明:**
    *   `sessionId`: 已终止的会话ID。
    *   `reason`: 对端断开或会话结束的原因。

---

## 5. 通用错误消息 (`error`)

当发生任何未被特定消息类型处理的错误，或者服务器需要向客户端发送通用错误通知时，会使用此消息。

*   **消息类型:** `protocol.MessageTypeError` ("error")
*   **方向:** Server -> Client
*   **结构 (`protocol.ErrorMessage`):**

    ```json
    {
      "type": "error",
      "sessionId": "OPTIONAL_SESSION_ID_IF_RELATED_TO_A_SESSION",
      "code": 40001, // 错误码
      "message": "无效的消息格式" // 错误描述
    }
    ```
*   **字段说明:**
    *   `sessionId`: (可选) 如果错误与特定会话相关，则包含会话ID。
    *   `code`: 数字错误码，对应 `protocol/protocol.go` 中定义的 `ErrorCode...` 常量。
    *   `message`: 人类可读的错误描述。

**常见错误码 (`protocol.ErrorCode...`):**

*   `40001` (`ErrorCodeBadRequest`): 无效的请求格式或参数。
*   `40101` (`ErrorCodeAuthRequired`): 需要认证。
*   `40102` (`ErrorCodeAuthFailed`): 认证失败 (例如 Token 无效、过期、被拒)。
*   `40301` (`ErrorCodePermissionDenied`): 权限不足。
*   `40401` (`ErrorCodeNotFound`): 资源未找到。
*   `40402` (`ErrorCodeProviderNotFound`): 指定的发卡方未找到或不可用。
*   `40902` (`ErrorCodeSessionConflict`): 会话相关的冲突 (例如一方或双方状态已改变)。
*   `40903` (`ErrorCodeProviderBusy`): 发卡方正忙。
*   `40904` (`ErrorCodeReceiverBusy`): 收卡方正忙。
*   `40905` (`ErrorCodeSelectSelf`): 不能选择自己。
*   `40906` (`ErrorCodeProviderUnavailable`): 提供者状态变更，不可用。
*   `41501` (`ErrorCodeUnsupportedType`): 不支持的消息类型。
*   `50001` (`ErrorCodeInternalError`): 服务器内部错误。

---

## 6. 客户端状态更新 (可选/未来扩展)

`protocol.MessageTypeStatusUpdate` ("status_update_to_server") 和 `protocol.MessageTypePeerStatusUpdate` ("peer_status_update") 目前在 `hub.go` 中未被完整处理。如果未来需要客户端（特别是Provider端）向服务器报告更细致的状态（如 "NFC卡已连接", "NFC卡已移除", "NFC读卡错误"），并由服务器转发给对端，则需要启用和完善这些消息的处理逻辑。

---

## 7. 心跳机制

WebSocket 连接通过标准的 Ping/Pong 帧来维持心跳，由 `gorilla/websocket` 库在服务器端自动处理。客户端库通常也会自动响应 Pong 帧。这确保了连接的活性检测。

## 8. 注意事项

*   **并发:** 客户端应准备好异步接收来自服务器的消息，特别是 `card_providers_list` (列表更新推送) 和 `peer_disconnected`。
*   **错误处理:** 客户端应能健壮地处理各种错误消息和网络异常。
*   **APDU 数据:** 所有 APDU 数据在 JSON 消息中都应使用 Base64 编码。
*   **UserID 一致性:** Provider 和 Receiver 必须属于同一个 UserID 才能建立会话和查看对方。
