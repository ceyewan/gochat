-- im-repo 数据库初始化脚本

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS im_repo CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 使用数据库
USE im_repo;

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE COMMENT '用户名',
    password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希',
    nickname VARCHAR(100) NOT NULL COMMENT '昵称',
    avatar_url VARCHAR(500) DEFAULT '' COMMENT '头像URL',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX idx_username (username),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- 创建群组表
CREATE TABLE IF NOT EXISTS groups (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '群组名称',
    description TEXT COMMENT '群组描述',
    owner_id BIGINT UNSIGNED NOT NULL COMMENT '群主ID',
    group_type INT NOT NULL DEFAULT 1 COMMENT '群组类型：1-普通群，2-超级群',
    member_count INT NOT NULL DEFAULT 0 COMMENT '成员数量',
    max_members INT NOT NULL DEFAULT 500 COMMENT '最大成员数',
    avatar VARCHAR(500) DEFAULT '' COMMENT '群头像URL',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX idx_owner_id (owner_id),
    INDEX idx_group_type (group_type),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群组表';

-- 创建群组成员表
CREATE TABLE IF NOT EXISTS group_members (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    group_id BIGINT UNSIGNED NOT NULL COMMENT '群组ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    role INT NOT NULL DEFAULT 1 COMMENT '角色：1-普通成员，2-管理员，3-群主',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '加入时间',
    
    UNIQUE KEY uk_group_user (group_id, user_id),
    INDEX idx_group_id (group_id),
    INDEX idx_user_id (user_id),
    INDEX idx_role (role),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群组成员表';

-- 创建消息表
CREATE TABLE IF NOT EXISTS messages (
    id BIGINT UNSIGNED PRIMARY KEY COMMENT '消息ID',
    conversation_id VARCHAR(100) NOT NULL COMMENT '会话ID',
    sender_id BIGINT UNSIGNED NOT NULL COMMENT '发送者ID',
    message_type INT NOT NULL DEFAULT 1 COMMENT '消息类型：1-文本，2-图片，3-语音，4-视频，5-文件',
    content TEXT NOT NULL COMMENT '消息内容',
    seq_id BIGINT UNSIGNED NOT NULL COMMENT '序列号',
    extra JSON COMMENT '扩展字段',
    deleted BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否已删除',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    INDEX idx_conversation_seq (conversation_id, seq_id),
    INDEX idx_sender_id (sender_id),
    INDEX idx_message_type (message_type),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='消息表';

-- 创建用户已读位置表
CREATE TABLE IF NOT EXISTS user_read_pointers (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    conversation_id VARCHAR(100) NOT NULL COMMENT '会话ID',
    last_read_seq_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后已读序列号',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    
    UNIQUE KEY uk_user_conversation (user_id, conversation_id),
    INDEX idx_conversation_id (conversation_id),
    INDEX idx_updated_at (updated_at),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户已读位置表';

-- 插入测试数据
INSERT INTO users (username, password_hash, nickname) VALUES
('admin', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iYqiSuAiPiVpbpbOIWCKYZYzO2Iq', '管理员'),
('user1', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iYqiSuAiPiVpbpbOIWCKYZYzO2Iq', '用户1'),
('user2', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iYqiSuAiPiVpbpbOIWCKYZYzO2Iq', '用户2');

-- 插入测试群组
INSERT INTO groups (name, description, owner_id, group_type, max_members) VALUES
('测试群组', '这是一个测试群组', 1, 1, 500);

-- 插入群组成员
INSERT INTO group_members (group_id, user_id, role) VALUES
(1, 1, 3), -- 管理员作为群主
(1, 2, 1), -- 用户1作为普通成员
(1, 3, 1); -- 用户2作为普通成员

-- 更新群组成员数量
UPDATE groups SET member_count = 3 WHERE id = 1;