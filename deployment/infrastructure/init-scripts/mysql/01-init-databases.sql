-- GoChat 数据库初始化脚本
-- 创建开发、测试和生产环境数据库

-- 创建开发环境数据库
CREATE DATABASE IF NOT EXISTS `gochat_dev` 
CHARACTER SET utf8mb4 
COLLATE utf8mb4_unicode_ci;

-- 创建测试环境数据库
CREATE DATABASE IF NOT EXISTS `gochat_test` 
CHARACTER SET utf8mb4 
COLLATE utf8mb4_unicode_ci;

-- 创建生产环境数据库（预留）
CREATE DATABASE IF NOT EXISTS `gochat_prod` 
CHARACTER SET utf8mb4 
COLLATE utf8mb4_unicode_ci;

-- 创建专用用户并授权
CREATE USER IF NOT EXISTS 'gochat'@'%' IDENTIFIED BY 'gochat_pass_2024';

-- 授权开发环境数据库
GRANT ALL PRIVILEGES ON `gochat_dev`.* TO 'gochat'@'%';

-- 授权测试环境数据库
GRANT ALL PRIVILEGES ON `gochat_test`.* TO 'gochat'@'%';

-- 授权生产环境数据库（预留）
GRANT ALL PRIVILEGES ON `gochat_prod`.* TO 'gochat'@'%';

-- 刷新权限
FLUSH PRIVILEGES;

-- 显示创建的数据库
SHOW DATABASES LIKE 'gochat_%';