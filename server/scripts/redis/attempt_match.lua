-- attempt_match.lua
-- @description: 原子化地尝试匹配或加入NFC配对等待池
-- @keys: KEYS[1] - a hash key for the waiting pool, e.g., "pairing:waiting:pool"
-- @argv: ARGV[1] - the user ID of the client attempting to pair
-- @argv: ARGV[2] - the role of the client (e.g., "transmitter")
-- @argv: ARGV[3] - a JSON string containing the current user's full pairing data
-- @return: Returns the JSON data of the matched peer if a match is found, otherwise returns nil.

local poolKey = KEYS[1]
local currentUserID = ARGV[1]
local currentRole = ARGV[2]
local currentUserDataJSON = ARGV[3]

-- 1. 尝试直接查找该用户是否已经有另一个角色在等待池中
local existingPeerDataJSON = redis.call('HGET', poolKey, currentUserID)

-- 2. 检查是否存在等待的伙伴
if existingPeerDataJSON then
    -- 伙伴存在，解码其数据
    local peerData = cjson.decode(existingPeerDataJSON)

    -- 3. 检查角色是否互补（即不相等），这是成功匹配的条件
    if peerData.role ~= currentRole then
        -- 角色互补，匹配成功！
        -- 3.1. 从等待池中移除该用户条目，因为已经完成匹配
        redis.call('HDEL', poolKey, currentUserID)
        -- 3.2. 返回已匹配伙伴的数据
        return existingPeerDataJSON
    else
        -- 角色相同，说明是同一用户在另一个设备上以后到者为准（Session Takeover）
        -- 用新会话的数据覆盖旧数据
        redis.call('HSET', poolKey, currentUserID, currentUserDataJSON)
        -- 返回nil，表示覆盖后继续等待
        return nil
    end
else
    -- 4. 池中没有该用户的任何角色在等待
    -- 将当前用户加入等待池
    redis.call('HSET', poolKey, currentUserID, currentUserDataJSON)
    -- 返回nil，表示已加入等待，尚未匹配
    return nil
end 