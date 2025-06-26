# EMQX API修复手动验证指南

## 修复内容总结

### 1. 问题确认
- **问题描述**: 角色冲突处理中，设备B强制获取transmitter角色后，设备A的MQTT连接没有被断开
- **根本原因**: EMQX API密码配置不一致
  - `config.yaml`中配置: `nfc_relay_admin_2024` 
  - 实际EMQX Dashboard密码: `xuehua123`

### 2. 修复操作
- **文件**: `server/config.yaml`
- **位置**: `mqtt.api.password`
- **修改**: 从 `nfc_relay_admin_2024` 改为 `xuehua123`

### 3. 代码逻辑确认
经过代码分析，确认角色冲突处理流程完整：

1. **API端点**: `/role/generateMQTTToken` (POST)
2. **服务方法**: `RoleConflictService.AssignRole`
3. **强制踢出**: 当`force_kick=true`时调用`handleForceKick`
4. **双重断开机制**:
   - JWT撤销: 从Redis删除`mqtt:active:{userID}:{jti}`
   - 物理断开: 通过EMQX API `/api/v5/clients/{clientID}` DELETE

### 4. 手动验证步骤

#### 步骤1: 验证EMQX API连接
```bash
curl -X POST "http://192.168.50.194:18083/api/v5/login" \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "xuehua123"}'
```
预期结果: 返回200状态码和token

#### 步骤2: 测试客户端断开API
```bash
# 使用步骤1获得的token
curl -X DELETE "http://192.168.50.194:18083/api/v5/clients/test-client-123" \
  -H "Authorization: Bearer {TOKEN}"
```
预期结果: 返回200/204状态码（客户端不存在时返回404也正常）

#### 步骤3: 获取当前连接客户端列表
```bash
curl -X GET "http://192.168.50.194:18083/api/v5/clients" \
  -H "Authorization: Bearer {TOKEN}"
```
预期结果: 返回当前连接的客户端列表

### 5. 完整角色冲突测试

#### 准备工作
1. 确保服务器运行在8888端口
2. 确保EMQX运行在18083端口（管理API）和8883端口（MQTT）

#### 测试流程
1. **设备A获取角色**:
   ```bash
   curl -X POST "http://192.168.50.194:8888/role/generateMQTTToken" \
     -H "x-token: {SERVER_TOKEN}" \
     -H "Content-Type: application/json" \
     -d '{
       "user_id": "test_user_001",
       "role": "transmitter",
       "device_info": {"device_model": "TestDevice_A"},
       "force_kick": false
     }'
   ```

2. **检查设备A连接状态**:
   使用EMQX API检查返回的client_id是否在线

3. **设备B强制获取角色**:
   ```bash
   curl -X POST "http://192.168.50.194:8888/role/generateMQTTToken" \
     -H "x-token: {SERVER_TOKEN}" \
     -H "Content-Type: application/json" \
     -d '{
       "user_id": "test_user_001",
       "role": "transmitter",  
       "device_info": {"device_model": "TestDevice_B"},
       "force_kick": true
     }'
   ```

4. **验证结果**:
   - 设备A的client_id应该从EMQX客户端列表中消失
   - 设备B应该成功获得新的token和client_id

### 6. 关键日志监控

查看服务器日志中的关键信息：
- `开始强制断开EMQX客户端`
- `EMQX API登录成功，获取到Token`
- `发送EMQX客户端断开请求`
- `成功通过EMQX API请求断开客户端连接`

如果出现错误：
- `EMQX API登录认证失败` - 说明密码配置仍有问题
- `发送EMQX断开连接请求失败` - 说明网络或API调用问题

### 7. 验证结论

如果所有步骤都成功：
- ✅ EMQX API密码配置修复成功
- ✅ 角色冲突处理功能正常工作
- ✅ 强制踢出机制有效

## 技术总结

此次修复解决了一个关键的安全问题：
1. **安全机制**: 双重保护（JWT撤销+物理断开）
2. **配置管理**: 统一的API认证配置
3. **错误处理**: 优雅的降级处理（即使EMQX API失败，JWT撤销仍能保证安全）

修复后的系统能够有效防止多设备同时持有相同角色的安全风险。 