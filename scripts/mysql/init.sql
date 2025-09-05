-- GoChat 数据库初始化脚本
-- 创建数据库和基础表结构

-- 创建数据库
CREATE DATABASE IF NOT EXISTS gochat CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 使用数据库
USE gochat;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '用户ID',
    username VARCHAR(50) NOT NULL UNIQUE COMMENT '用户名',
    password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希',
    nickname VARCHAR(50) DEFAULT '' COMMENT '昵称',
    avatar_url VARCHAR(255) DEFAULT '' COMMENT '头像URL',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX idx_username (username),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 群组表
CREATE TABLE IF NOT EXISTS groups (
    id BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '群组ID',
    name VARCHAR(50) NOT NULL COMMENT '群组名称',
    owner_id BIGINT UNSIGNED NOT NULL COMMENT '群主ID',
    member_count INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '成员数量',
    avatar_url VARCHAR(255) DEFAULT '' COMMENT '群组头像URL',
    description TEXT COMMENT '群组描述',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX idx_owner_id (owner_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群组表';

-- 群组成员表
CREATE TABLE IF NOT EXISTS group_members (
    id BIGINT UNSIGNED NOT NULL PRIMARY KEY AUTO_INCREMENT COMMENT '记录ID',
    group_id BIGINT UNSIGNED NOT NULL COMMENT '群组ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    role TINYINT NOT NULL DEFAULT 1 COMMENT '角色(1:成员,2:管理员,3:群主)',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '加入时间',
    
    UNIQUE KEY uk_group_user (group_id, user_id),
    INDEX idx_user_id (user_id),
    INDEX idx_joined_at (joined_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群组成员表';

-- 消息表
CREATE TABLE IF NOT EXISTS messages (
    id BIGINT UNSIGNED NOT NULL PRIMARY KEY COMMENT '消息ID',
    conversation_id VARCHAR(64) NOT NULL COMMENT '会话ID',
    sender_id BIGINT UNSIGNED NOT NULL COMMENT '发送者ID',
    message_type TINYINT NOT NULL DEFAULT 1 COMMENT '消息类型(1:文本,2:图片,3:文件,4:系统)',
    content TEXT NOT NULL COMMENT '消息内容',
    seq_id BIGINT UNSIGNED NOT NULL COMMENT '会话内序列号',
    deleted BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否已删除',
    extra TEXT COMMENT '扩展信息(JSON)',
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    UNIQUE KEY uk_conv_seq (conversation_id, seq_id),
    INDEX idx_conv_time (conversation_id, created_at),
    INDEX idx_sender_id (sender_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='消息表';

-- 用户已读位置表
CREATE TABLE IF NOT EXISTS user_read_pointers (
    id BIGINT UNSIGNED NOT NULL PRIMARY KEY AUTO_INCREMENT COMMENT '记录ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    conversation_id VARCHAR(64) NOT NULL COMMENT '会话ID',
    last_read_seq_id BIGINT UNSIGNED NOT NULL COMMENT '最后已读序列号',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    UNIQUE KEY uk_user_conv (user_id, conversation_id),
    INDEX idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户已读位置表';

-- 插入测试数据
INSERT INTO users (id, username, password_hash, nickname, avatar_url) VALUES
(1, 'admin', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iYqiSfFe5ldcIjvQJp/VRaUHKopC', '管理员', ''),
(2, 'user1', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iYqiSfFe5ldcIjvQJp/VRaUHKopC', '用户1', ''),
(3, 'user2', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iYqiSfFe5ldcIjvQJp/VRaUHKopC', '用户2', '')
ON DUPLICATE KEY UPDATE username=VALUES(username);

-- 插入测试群组
INSERT INTO groups (id, name, owner_id, member_count, description) VALUES
(1, '测试群组', 1, 3, '这是一个测试群组')
ON DUPLICATE KEY UPDATE name=VALUES(name);

-- 插入群组成员
INSERT INTO group_members (group_id, user_id, role) VALUES
(1, 1, 3), -- 群主
(1, 2, 1), -- 成员
(1, 3, 1)  -- 成员
ON DUPLICATE KEY UPDATE role=VALUES(role);