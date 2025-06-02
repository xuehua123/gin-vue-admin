# JWT生成与验证模块使用指南

## 概述

本模块实现了符合《开发手册：安全卡片中继系统 V2.0》要求的完整JWT认证系统，包括JWT生成、验证、撤销和管理功能。

## 核心特性

### ✅ 已完成的功能

1. **完整的JWT生命周期管理**
   - JWT创建和签名
   - JWT解析和验证
   - JWT撤销机制
   - JWT刷新逻辑

2. **开发手册V2.0合规性**
   - 符合jwt:active Redis Key约定
   - 包含必需的userID、clientID、jti字段
   - 实现客户端会话管理

3. **安全特性**
   - JWT活跃状态验证
   - 防重放机制
   - 多设备登录管理
   - 强制下线功能

4. **可观测性**
   - 完整的日志记录
   - 错误追踪
   - 性能监控

## 架构设计

### 核心组件

```
├── model/system/request/jwt.go     # JWT Claims结构定义
├── utils/jwt.go                    # JWT核心工具类  
├── middleware/jwt.go               # JWT认证中间件
├── service/system/jwt_black_list.go # JWT服务层
└── utils/jwt_test.go              # 单元测试
```

### Redis Key约定

```
jwt:active:{userID}:{jti} -> clientID
```

- `userID`: 用户UUID字符串
- `jti`: JWT唯一标识符  
- `clientID`: 客户端会话ID

## API参考

### JWT工具类 (utils.JWT)

#### 创建JWT

```go
j := utils.NewJWT()

baseClaims := request.BaseClaims{
    UUID:        userUUID,
    ID:          userID,
    Username:    username,
    NickName:    nickname,
    AuthorityId: authorityId,
    ClientID:    clientID,
}

claims := j.CreateClaims(baseClaims)
token, err := j.CreateToken(claims)
```

#### 解析JWT

```go
j := utils.NewJWT()
claims, err := j.ParseToken(tokenString)
if err != nil {
    // 处理错误：过期、无效等
}
```

#### 验证JWT活跃状态

```go
j := utils.NewJWT()
isActive, err := j.IsJWTActive(claims)
if !isActive {
    // JWT已被撤销或不存在
}
```

#### 撤销JWT

```go
j := utils.NewJWT()

// 撤销单个JWT
err := j.RevokeJWT(claims)

// 撤销用户所有JWT
err := j.RevokeAllUserJWTs(userID)
```

#### 查询用户活跃JWT

```go
j := utils.NewJWT()
activeJWTs, err := j.GetUserActiveJWTs(userID)
// 返回: map[string]string (redisKey -> clientID)
```

### JWT服务层 (JwtService)

```go
jwtService := system.JwtServiceApp

// 撤销活跃JWT
err := jwtService.RevokeActiveJWT(claims)

// 强制用户下线
err := jwtService.RevokeAllUserJWTs(userID)

// 检查JWT状态
isActive, err := jwtService.IsJWTActive(claims)

// 获取用户活跃会话
sessions, err := jwtService.GetUserActiveJWTs(userID)
```

### JWT中间件

```go
// 在路由中使用
router.Use(middleware.JWTAuth())

// 在处理函数中获取用户信息
func handler(c *gin.Context) {
    claims := utils.GetUserInfo(c)
    userID := claims.GetUserID()
    clientID := claims.GetClientID()
}
```

## Claims结构

### CustomClaims

```go
type CustomClaims struct {
    BaseClaims
    BufferTime int64 `json:"buffer_time"`
    jwt.RegisteredClaims
}
```

### BaseClaims

```go
type BaseClaims struct {
    UUID        uuid.UUID `json:"user_id"`     // 用户UUID
    ID          uint      `json:"id"`          // 用户数据库ID  
    Username    string    `json:"username"`    // 用户名
    NickName    string    `json:"nick_name"`   // 用户昵称
    AuthorityId uint      `json:"authority_id"` // 权限角色ID
    ClientID    string    `json:"client_id"`   // 客户端会话ID
}
```

### 辅助方法

```go
// 获取用户ID字符串
userID := claims.GetUserID()

// 获取客户端ID
clientID := claims.GetClientID()

// 获取JWT唯一标识符
jti := claims.GetJTI()

// 验证Claims有效性
isValid := claims.IsValid()
```

## 配置说明

### config.yaml

```yaml
jwt:
  signing-key: "your-secret-signing-key"    # JWT签名密钥
  expires-time: 7d                          # 过期时间
  buffer-time: 1d                           # 刷新缓冲时间
  issuer: "GVA"                            # 签发者
```

## 错误处理

### 错误类型

```go
var (
    TokenValid            = errors.New("未知错误")
    TokenExpired          = errors.New("token已过期")
    TokenNotValidYet      = errors.New("token尚未激活")
    TokenMalformed        = errors.New("这不是一个token")
    TokenSignatureInvalid = errors.New("无效签名")
    TokenInvalid          = errors.New("无法处理此token")
    TokenNotActive        = errors.New("token未激活或已被撤销")
    TokenClaimsInvalid    = errors.New("token声明信息无效")
)
```

### 错误处理最佳实践

```go
claims, err := j.ParseToken(token)
if err != nil {
    switch {
    case errors.Is(err, utils.TokenExpired):
        // 引导用户重新登录
    case errors.Is(err, utils.TokenNotActive):
        // Token已被撤销，强制登出
    case errors.Is(err, utils.TokenSignatureInvalid):
        // 安全威胁，记录并阻止
    default:
        // 其他错误处理
    }
}
```

## 安全考虑

### 1. JWT刷新机制

- 当JWT接近过期时自动刷新
- 生成新的JTI和ClientID
- 旧JWT可选择立即撤销或自然过期

### 2. 多设备管理

- 每个设备登录生成唯一的ClientID
- 支持查看用户所有活跃会话
- 支持强制下线特定或全部设备

### 3. 防重放攻击

- 每个JWT包含唯一的JTI
- Redis存储确保JWT一次性使用
- 严格的时间戳验证

### 4. 日志审计

- 完整的JWT生命周期日志
- 安全事件记录
- 错误追踪和分析

## 性能优化

### 1. Redis优化

```go
// 使用Pipeline批量操作
pipe := global.GVA_REDIS.TxPipeline()
// ... 批量命令
_, err := pipe.Exec()
```

### 2. 缓存策略

- JWT Claims缓存
- 用户权限缓存  
- 合理的TTL设置

### 3. 监控指标

- JWT创建/验证次数
- Redis响应时间
- 错误率统计

## 测试

### 运行单元测试

```bash
# 运行所有JWT测试
go test ./utils -v -run TestJWT

# 运行特定测试
go test ./utils -v -run TestJWT_CreateToken

# 查看测试覆盖率
go test ./utils -v -cover
```

### 测试环境要求

- Redis服务器 (localhost:6379)
- 测试数据库 (DB 15)

## 集成示例

### 完整的用户登录流程

```go
func Login(c *gin.Context) {
    // 1. 验证用户凭据
    user, err := validateUser(username, password)
    if err != nil {
        response.FailWithMessage("登录失败", c)
        return
    }
    
    // 2. 生成ClientID
    clientID := uuid.New().String()
    
    // 3. 创建JWT
    j := utils.NewJWT()
    claims := j.CreateClaims(request.BaseClaims{
        UUID:        user.UUID,
        ID:          user.ID,
        Username:    user.Username,
        NickName:    user.NickName,
        AuthorityId: user.AuthorityId,
        ClientID:    clientID,
    })
    
    token, err := j.CreateToken(claims)
    if err != nil {
        response.FailWithMessage("生成token失败", c)
        return
    }
    
    // 4. 返回登录成功
    response.OkWithDetailed(LoginResponse{
        User:      user,
        Token:     token,
        ExpiresAt: claims.ExpiresAt.Unix() * 1000,
    }, "登录成功", c)
}
```

### 用户登出流程

```go
func Logout(c *gin.Context) {
    claims := utils.GetUserInfo(c)
    
    jwtService := system.JwtServiceApp
    err := jwtService.RevokeActiveJWT(claims)
    if err != nil {
        global.GVA_LOG.Error("撤销JWT失败", zap.Error(err))
    }
    
    utils.ClearToken(c)
    response.OkWithMessage("登出成功", c)
}
```

## 故障排除

### 常见问题

1. **Redis连接失败**
   - 检查Redis服务状态
   - 验证连接配置
   - 检查网络连通性

2. **JWT验证失败**
   - 检查签名密钥配置
   - 验证时间同步
   - 检查Claims结构

3. **性能问题**
   - 监控Redis响应时间
   - 检查JWT大小
   - 优化批量操作

### 调试技巧

1. **启用详细日志**
```go
global.GVA_LOG.Debug("JWT操作", 
    zap.String("operation", "create"),
    zap.String("userID", userID),
    zap.String("jti", jti))
```

2. **使用Redis监控**
```bash
redis-cli monitor | grep "jwt:active"
```

3. **性能分析**
```go
import _ "net/http/pprof"
// 在开发环境启用pprof
```

## 版本历史

- **v2.0.0** - 完全重构，符合开发手册V2.0规范
  - 新增jwt:active机制
  - 增强安全性
  - 改进错误处理
  - 完善测试覆盖

- **v1.x** - 传统黑名单机制（已废弃）

## 贡献指南

1. 提交前运行完整测试套件
2. 遵循代码风格规范
3. 更新相关文档
4. 添加必要的单元测试

---

**注意**: 本模块是安全卡片中继系统的核心组件，任何修改都应经过充分测试和安全审查。 