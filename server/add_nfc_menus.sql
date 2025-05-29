-- NFC中继管理模块菜单数据插入脚本
-- 适用于gin-vue-admin系统

-- 插入父级菜单：NFC中继管理
INSERT INTO `sys_base_menus` (`created_at`, `updated_at`, `deleted_at`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`, `transition_type`) VALUES 
(NOW(), NOW(), NULL, 0, 'nfc-relay-admin', 'nfc-relay-admin', 0, 'view/nfcRelayAdmin/index.vue', 50, '', 1, 0, 'NFC中继管理', 'Connection', 0, '');

-- 获取刚插入的父级菜单ID（需要在实际执行时替换为真实ID）
-- SET @parent_menu_id = LAST_INSERT_ID();

-- 如果你知道父级菜单的ID，可以直接使用，否则需要先查询
-- 假设父级菜单ID为100（请根据实际情况修改）
SET @parent_menu_id = 100;

-- 插入子菜单：概览仪表盘
INSERT INTO `sys_base_menus` (`created_at`, `updated_at`, `deleted_at`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`, `transition_type`) VALUES 
(NOW(), NOW(), NULL, @parent_menu_id, 'dashboard', 'nfc-relay-dashboard', 0, 'view/nfcRelayAdmin/dashboard/index.vue', 1, '', 1, 0, '概览仪表盘', 'Odometer', 0, '');

-- 插入子菜单：连接管理
INSERT INTO `sys_base_menus` (`created_at`, `updated_at`, `deleted_at`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`, `transition_type`) VALUES 
(NOW(), NOW(), NULL, @parent_menu_id, 'clients', 'nfc-relay-clients', 0, 'view/nfcRelayAdmin/clientManagement/index.vue', 2, '', 1, 0, '连接管理', 'User', 0, '');

-- 插入子菜单：会话管理
INSERT INTO `sys_base_menus` (`created_at`, `updated_at`, `deleted_at`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`, `transition_type`) VALUES 
(NOW(), NOW(), NULL, @parent_menu_id, 'sessions', 'nfc-relay-sessions', 0, 'view/nfcRelayAdmin/sessionManagement/index.vue', 3, '', 1, 0, '会话管理', 'ChatDotRound', 0, '');

-- 插入子菜单：审计日志
INSERT INTO `sys_base_menus` (`created_at`, `updated_at`, `deleted_at`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`, `transition_type`) VALUES 
(NOW(), NOW(), NULL, @parent_menu_id, 'audit-logs', 'nfc-relay-audit-logs', 0, 'view/nfcRelayAdmin/auditLogs/index.vue', 4, '', 1, 0, '审计日志', 'Document', 0, '');

-- 插入子菜单：系统配置
INSERT INTO `sys_base_menus` (`created_at`, `updated_at`, `deleted_at`, `parent_id`, `path`, `name`, `hidden`, `component`, `sort`, `active_name`, `keep_alive`, `default_menu`, `title`, `icon`, `close_tab`, `transition_type`) VALUES 
(NOW(), NOW(), NULL, @parent_menu_id, 'configuration', 'nfc-relay-configuration', 0, 'view/nfcRelayAdmin/configuration/index.vue', 5, '', 1, 0, '系统配置', 'Setting', 0, '');

-- 为超级管理员（authority_id = 888）授权访问这些菜单
-- 首先获取所有刚插入的菜单ID
INSERT INTO `sys_authority_menus` (`sys_authority_authority_id`, `sys_base_menu_id`) 
SELECT 888, `id` FROM `sys_base_menus` WHERE `name` LIKE 'nfc-relay%';

-- 刷新权限缓存的提示
-- 执行完成后，请重新登录系统以刷新菜单权限 