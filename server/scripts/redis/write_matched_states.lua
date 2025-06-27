-- write_matched_states.lua
-- 原子性地写入双方配对状态的Lua脚本
-- 用于避免状态写入过程中的竞争条件和部分失败
-- 【关键修复】确保匹配成功后的状态过期时间覆盖等待状态的过期时间

-- KEYS[1]: 当前用户的状态键 (pairing:state:userID)
-- KEYS[2]: 对端用户的状态键 (pairing:state:peerUserID)
-- KEYS[3]: 对端用户的超时键 (pairing:timeout:peerUserID:role)

-- ARGV[1]: 当前用户的角色
-- ARGV[2]: 当前用户的状态JSON
-- ARGV[3]: 对端用户的角色
-- ARGV[4]: 对端用户的状态JSON
-- ARGV[5]: 状态过期时间(秒)

local currentStateKey = KEYS[1]
local peerStateKey = KEYS[2] 
local peerTimeoutKey = KEYS[3]

local currentRole = ARGV[1]
local currentStatusJSON = ARGV[2]
local peerRole = ARGV[3]
local peerStatusJSON = ARGV[4]
local expireSeconds = tonumber(ARGV[5])

-- 【关键修复1】先重置过期时间，确保覆盖等待状态时设置的较短TTL
-- 这是解决5秒内状态丢失问题的核心修复
local currentExpireFirstResult = redis.call('EXPIRE', currentStateKey, expireSeconds)
local peerExpireFirstResult = redis.call('EXPIRE', peerStateKey, expireSeconds)

-- 【关键修复2】再写入状态数据，确保即使键不存在也会创建
local currentHSetResult = redis.call('HSET', currentStateKey, currentRole, currentStatusJSON)
local peerHSetResult = redis.call('HSET', peerStateKey, peerRole, peerStatusJSON)

-- 【关键修复3】再次确保过期时间正确设置，形成双重保障
local currentExpireSecondResult = redis.call('EXPIRE', currentStateKey, expireSeconds)
local peerExpireSecondResult = redis.call('EXPIRE', peerStateKey, expireSeconds)

-- 清理对端的超时键（如果存在的话）
local timeoutDelResult = redis.call('DEL', peerTimeoutKey)

-- 返回详细的操作结果供调试使用
return {
    currentHSetResult,
    currentExpireFirstResult,
    currentExpireSecondResult,
    peerHSetResult, 
    peerExpireFirstResult,
    peerExpireSecondResult,
    timeoutDelResult,
    -- 【诊断信息】返回操作后的TTL状态
    redis.call('TTL', currentStateKey),
    redis.call('TTL', peerStateKey)
} 