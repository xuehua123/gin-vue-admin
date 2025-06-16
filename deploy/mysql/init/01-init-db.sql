-- MySQL 数据库初始化脚本
-- 用于 Gin-Vue-Admin 项目的生产环境

-- 设置字符集
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 确保数据库使用正确的字符集
ALTER DATABASE gin_vue_admin CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

-- 为 gva_user 用户授予必要权限（密码已通过环境变量设置）
GRANT ALL PRIVILEGES ON gin_vue_admin.* TO 'gva_user'@'%';
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, INDEX, ALTER, REFERENCES ON gin_vue_admin.* TO 'gva_user'@'%';

-- 刷新权限
FLUSH PRIVILEGES;

-- 创建一些基础性能优化索引（这些会在应用启动时由 GORM 自动创建，这里只是示例）
-- 注意：实际的表结构将由应用程序的 AutoMigrate 功能创建

-- 设置一些会话级别的优化
SET SESSION sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION';

-- 创建一个健康检查表（可选）
USE gin_vue_admin;
CREATE TABLE IF NOT EXISTS health_check (
    id INT AUTO_INCREMENT PRIMARY KEY,
    status VARCHAR(10) DEFAULT 'OK',
    last_check TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO health_check (status) VALUES ('OK') ON DUPLICATE KEY UPDATE last_check = CURRENT_TIMESTAMP;

SET FOREIGN_KEY_CHECKS = 1; 