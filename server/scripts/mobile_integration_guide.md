# 📱 **NFC支付系统 - 手机端集成与测试指南**

本文档为手机端（iOS/Android）开发人员提供详细的集成指南，以实现与后端NFC角色冲突管理系统的无缝对接。

---

## 📚 **目录**

1.  **[#核心概念](#核心概念)**
2.  **[#API接口定义](#api接口定义)**
3.  **[#数据模型](#数据模型)**
4.  **[#MQTT集成](#mqtt集成)**
5.  **[#集成步骤（Flutter示例）](#集成步骤flutter示例)**
6.  **[#集成步骤（Android/Kotlin原生示例）](#集成步骤androidkotlin原生示例)**
7.  **[#关键测试场景](#关键测试场景)**
8.  **[#故障排查](#故障排查)**

---

## 核心概念

-   **角色 (Role)**: 用户在NFC交互中的身份，分为`transmitter`（发卡端）和`receiver`（收卡端）。
-   **角色冲突 (Role Conflict)**: 同一用户在多个设备上尝试扮演同一角色。
-   **强制挤下线 (Force Kick)**: 后登录的设备强制使先登录的设备下线，以解决角色冲突。
-   **MQTT Token**: 一个专用的、有生命周期的JWT，用于MQTT连接认证。

## API接口定义

### **服务器地址**

-   **Base URL**: `http://43.165.186.134:8888`

### **接口详情**

#### 1. **用户登录**

-   **Endpoint**: `POST /base/login`
-   **功能**: 获取API访问的`x-token`。
-   **请求体**:
    ```json
    {
        "username": "your_username",
        "password": "your_password"
    }
    ```

#### 2. **生成MQTT Token（处理角色分配）**

-   **Endpoint**: `POST /role/generateMQTTToken`
-   **Headers**: `x-token: YOUR_AUTH_TOKEN`
-   **功能**: 为设备申请一个角色。这是最核心的接口，后端会处理所有冲突检测和挤下线逻辑。
-   **请求体**:
    ```json
    {
        "role": "transmitter", // "transmitter" 或 "receiver"
        "force_kick_existing": false, // 如果检测到冲突，是否强制挤下线
        "device_info": {
            "device_model": "iPhone 14 Pro",
            "app_version": "1.2.3"
        }
    }
    ```
-   **成功响应**:
    ```json
    {
        "code": 0,
        "data": {
            "client_id": "admin-transmitter-1672502400",
            "token": "ey...", // 用于MQTT连接的JWT
            "role": "transmitter"
        },
        "msg": "生成成功"
    }
    ```

## 数据模型

### **MQTTTokenResponse**

```dart
// Dart/Flutter
class MQTTTokenResponse {
  final String clientId;
  final String token;
  final String role;

  MQTTTokenResponse({required this.clientId, required this.token, required this.role});

  factory MQTTTokenResponse.fromJson(Map<String, dynamic> json) {
    return MQTTTokenResponse(
      clientId: json['client_id'],
      token: json['token'],
      role: json['role'],
    );
  }
}
```

### **挤下线通知 (RoleRevokedNotification)**

-   **主题**: `client/{your_client_id}/control/role_revoked_notification`
-   **消息体**:
    ```json
    {
        "reason": "role_revoked_by_peer",
        "kicker_client_id": "admin-transmitter-1672502405", // 挤下你的设备的ID
        "timestamp_utc": "2023-01-01T00:00:05Z"
    }
    ```

## MQTT集成

### **连接参数**

-   **Host**: `49.235.40.39`
-   **Port**: `1883` (TCP), `8083` (WebSocket)
-   **ClientID**: 使用从 `/role/generateMQTTToken` 获取的 `client_id`。
-   **Username**: 与 `ClientID` **相同**。
-   **Password**: 使用从 `/role/generateMQTTToken` 获取的 `token`。

### **核心订阅主题**

1.  **接收挤下线通知**
    -   `client/{your_client_id}/control/role_revoked_notification`
2.  **接收对端状态同步**
    -   `client/{your_client_id}/sync/peer_state_update`
3.  **接收交易指令**
    -   `client/{your_client_id}/transaction/apdu_command`

## 集成步骤（Flutter示例）

### **1. 依赖**

```yaml
# pubspec.yaml
dependencies:
  mqtt_client: ^10.0.0
  dio: ^5.0.0
```

### **2. 服务封装 (`nfc_role_service.dart`)**

```dart
import 'package:dio/dio.dart';
import 'package:mqtt_client/mqtt_server_client.dart';
// ... import your models and other packages

class NfcRoleService {
  final Dio _dio;
  MqttServerClient? _mqttClient;

  NfcRoleService() : _dio = Dio(BaseOptions(baseUrl: "http://43.165.186.134:8888"));

  // 步骤1: 登录并设置Auth Token
  Future<void> login(String username, String password) async {
    final response = await _dio.post('/base/login', data: {'username': username, 'password': password});
    final token = response.data['data']['token'];
    _dio.options.headers['x-token'] = token;
  }

  // 步骤2: 申请角色并获取MQTT Token
  Future<MQTTTokenResponse> requestRole(String role, {bool forceKick = false}) async {
    final response = await _dio.post('/role/generateMQTTToken', data: {
      'role': role,
      'force_kick_existing': forceKick,
      'device_info': {'device_model': 'Flutter Test Device'}
    });
    return MQTTTokenResponse.fromJson(response.data['data']);
  }

  // 步骤3: 连接MQTT
  Future<bool> connectMqtt(MQTTTokenResponse tokenInfo) async {
    _mqttClient = MqttServerClient(EMQX_HOST, tokenInfo.clientId);
    _mqttClient!.port = 1883;
    _mqttClient!.logging(on: true);

    final connMessage = MqttConnectMessage()
        .withClientIdentifier(tokenInfo.clientId)
        .withWillTopic('willtopic') // LWT
        .withWillMessage('My Will message')
        .startClean()
        .withWillQos(MqttQos.atLeastOnce);

    _mqttClient!.connectionMessage = connMessage;
    _mqttClient!.clientIdentifier = tokenInfo.clientId;
    _mqttClient!.userName = tokenInfo.clientId; // Username is the ClientID
    _mqttClient!.password = tokenInfo.token;    // Password is the JWT

    try {
      await _mqttClient!.connect();
    } catch (e) {
      print('MQTT 连接异常: $e');
      _mqttClient!.disconnect();
      return false;
    }
    
    // 订阅关键主题
    final subTopic = 'client/${tokenInfo.clientId}/control/role_revoked_notification';
    _mqttClient!.subscribe(subTopic, MqttQos.atLeastOnce);

    // 监听消息
    _mqttClient!.updates!.listen((List<MqttReceivedMessage<MqttMessage?>>? c) {
      final recMess = c![0].payload as MqttPublishMessage;
      final pt = MqttPublishPayload.bytesToStringAsString(recMess.payload.message);

      print('收到消息: topic is <${c[0].topic}>, payload is <-- $pt -->');
      
      // 处理挤下线通知
      if (c[0].topic == subTopic) {
        handleRoleRevoked(pt);
      }
    });
    
    return true;
  }
  
  // 步骤4: 处理挤下线
  void handleRoleRevoked(String payload) {
    print("‼️ 被挤下线了！ Payload: $payload");
    // 在这里实现UI提示，并断开连接
    _mqttClient?.disconnect();
    // UI层面应提示用户，例如弹窗
    // showKickedOutDialog();
  }
  
  void disconnect() {
    _mqttClient?.disconnect();
  }
}
```

## 集成步骤（Android/Kotlin原生示例）

### **1. 依赖**

```groovy
// build.gradle (app)
dependencies {
    implementation 'org.eclipse.paho:org.eclipse.paho.android.service:1.1.1'
    implementation 'com.squareup.retrofit2:retrofit:2.9.0'
    implementation 'com.squareup.retrofit2:converter-gson:2.9.0'
}
```

### **2. 服务封装 (`NfcRoleManager.kt`)**

```kotlin
import org.eclipse.paho.android.service.MqttAndroidClient
import org.eclipse.paho.client.mqttv3.*

class NfcRoleManager(private val context: Context) {
    private val apiService: ApiService = // ... (Retrofit instance)
    private var mqttClient: MqttAndroidClient? = null

    // 步骤1 & 2: 申请角色
    suspend fun requestRoleAndConnect(role: String, forceKick: Boolean): Boolean {
        val tokenInfo = apiService.generateMqttToken(
            GenerateTokenRequest(role, forceKick, mapOf("device_model" to Build.MODEL))
        ).data ?: return false
        
        return connectMqtt(tokenInfo)
    }

    // 步骤3: 连接MQTT
    private fun connectMqtt(tokenInfo: MqttTokenResponse): Boolean {
        val clientId = tokenInfo.clientId
        mqttClient = MqttAndroidClient(context, "tcp://$EMQX_HOST:$EMQX_PORT", clientId)
        
        val options = MqttConnectOptions().apply {
            userName = clientId
            password = tokenInfo.token.toCharArray()
            isCleanSession = true
        }

        try {
            mqttClient?.connect(options, null, object : IMqttActionListener {
                override fun onSuccess(asyncActionToken: IMqttToken?) {
                    subscribeToTopics(clientId)
                }
                override fun onFailure(asyncActionToken: IMqttToken?, exception: Throwable?) {
                    // Handle failure
                }
            })
        } catch (e: MqttException) {
            e.printStackTrace()
            return false
        }
        return true
    }

    // 步骤4: 订阅与消息处理
    private fun subscribeToTopics(clientId: String) {
        mqttClient?.setCallback(object : MqttCallbackExtended {
            override fun messageArrived(topic: String, message: MqttMessage) {
                if (topic.contains("role_revoked_notification")) {
                    // 被挤下线
                    handleRoleRevoked()
                }
            }
            // ... other callback methods
        })
        mqttClient?.subscribe("client/$clientId/control/#", 1)
    }

    private fun handleRoleRevoked() {
        // 在UI线程中显示弹窗
        // 断开连接
        mqttClient?.disconnect()
    }
}
```

## 关键测试场景

1.  **正常流程**
    -   设备A申请`transmitter`角色 -> 成功。
    -   设备A连接MQTT -> 成功。
    -   设备B申请`receiver`角色 -> 成功。
    -   设备B连接MQTT -> 成功。

2.  **冲突与挤下线**
    -   设备A已是`transmitter`。
    -   设备B申请`transmitter`角色，`force_kick_existing: true` -> 成功获取新Token。
    -   设备A的MQTT客户端收到`role_revoked_notification`消息，并被服务器断开连接。
    -   设备B使用新Token成功连接MQTT。

## 故障排查

-   **连接失败**: 检查Token是否过期、ClientID/Username/Password是否正确。
-   **收不到消息**: 确认主题订阅是否正确，检查EMQX控制台的客户端订阅列表。
-   **API请求失败**: 检查`x-token`是否正确携带，请求体格式是否正确。 