# GoChat 数据模型

本文档定义了 GoChat 系统的数据库架构。主数据库是 MySQL。

## 1. 表：`users`

存储用户账户信息。

| 列         | 类型          | 约束      | 描述                               |
| -------------- | ------------- | ---------------- | ----------------------------------------- |
| `id`           | `BIGINT`      | `PRIMARY KEY`    | 用户的唯一标识符。           |
| `username`     | `VARCHAR(50)` | `UNIQUE, NOT NULL` | 用户的登录名。                    |
| `password_hash`| `VARCHAR(255)`| `NOT NULL`       | 用户的哈希密码。               |
| `nickname`     | `VARCHAR(50)` |                  | 用户的显示名称。                  |
| `avatar_url`   | `VARCHAR(255)`|                  | 用户头像图像的 URL。           |
| `is_guest`     | `BOOLEAN`     | `NOT NULL, DEFAULT false` | 如果用户是访客账户则为 `true`。 |
| `created_at`   | `DATETIME`    | `NOT NULL`       | 创建用户的时间戳。   |
| `updated_at`   | `DATETIME`    | `NOT NULL`       | 最后更新的时间戳。             |

## 2. 表：`conversations`

存储有关每个会话的信息，可以是单个聊天或群组聊天。

| 列          | 类型          | 约束   | 描述                               |
| --------------- | ------------- | ------------- | ----------------------------------------- |
| `id`            | `VARCHAR(64)` | `PRIMARY KEY` | 会话的唯一标识符。   |
| `type`          | `INT`         | `NOT NULL`    | 会话的类型（1：单个，2：群组）。 |
| `last_message_id`| `BIGINT`      |               | 会话中最后一条消息的 ID。 |
| `created_at`    | `DATETIME`    | `NOT NULL`    | 创建会话的时间戳。 |
| `updated_at`    | `DATETIME`    | `NOT NULL`    | 最后更新的时间戳。             |

## 3. 表：`groups`

存储有关群组聊天的信息。

| 列        | 类型          | 约束      | 描述                               |
| ------------- | ------------- | ---------------- | ----------------------------------------- |
| `id`          | `BIGINT`      | `PRIMARY KEY`    | 群组的唯一标识符。          |
| `name`        | `VARCHAR(50)` | `NOT NULL`       | 群组的名称。                    |
| `owner_id`    | `BIGINT`      | `NOT NULL`       | 群组所有者的用户 ID。           |
| `member_count`| `INT`         | `NOT NULL, DEFAULT 0` | 群组中的成员数量。       |
| `avatar_url`  | `VARCHAR(255)`|                  | 群组头像图像的 URL。          |
| `description` | `TEXT`        |                  | 群组的描述。               |
| `created_at`  | `DATETIME`    | `NOT NULL`       | 创建群组的时间戳。  |
| `updated_at`  | `DATETIME`    | `NOT NULL`       | 最后更新的时间戳。             |

## 4. 表：`group_members`

将用户映射到他们所属的群组。

| 列     | 类型     | 约束                  | 描述                               |
| ---------- | -------- | ---------------------------- | ----------------------------------------- |
| `id`       | `BIGINT` | `PRIMARY KEY`                | 成员资格记录的唯一标识符。 |
| `group_id` | `BIGINT` | `UNIQUE(group_id, user_id)` | 群组的 ID。                      |
| `user_id`  | `BIGINT` | `UNIQUE(group_id, user_id)` | 用户的 ID。                       |
| `role`     | `INT`    | `NOT NULL, DEFAULT 1`        | 用户在群组中的角色（1：成员，2：管理员，3：所有者）。 |
| `joined_at`| `DATETIME`| `NOT NULL`                   | 用户加入群组的时间戳。 |

## 5. 表：`messages`

存储在会话中发送的所有消息。

| 列          | 类型          | 约束                  | 描述                               |
| --------------- | ------------- | ---------------------------- | ----------------------------------------- |
| `id`            | `BIGINT`      | `PRIMARY KEY`                | 消息的唯一标识符。        |
| `conversation_id`| `VARCHAR(64)` | `UNIQUE(conversation_id, seq_id)` | 消息所属的会话的 ID。 |
| `sender_id`     | `BIGINT`      | `NOT NULL`                   | 发送者的用户 ID。                |
| `message_type`  | `INT`         | `NOT NULL, DEFAULT 1`        | 消息的类型（例如，文本、图像）。  |
| `content`       | `TEXT`        | `NOT NULL`                   | 消息的内容。               |
| `seq_id`        | `BIGINT`      | `UNIQUE(conversation_id, seq_id)` | 会话内的顺序 ID。 |
| `deleted`       | `BOOLEAN`     | `NOT NULL, DEFAULT false`    | 如果消息已被软删除则为 `true`。 |
| `extra`         | `TEXT`        |                              | 消息的额外元数据（JSON）。    |
| `created_at`    | `DATETIME`    | `NOT NULL`                   | 创建消息的时间戳。 |
| `updated_at`    | `DATETIME`    | `NOT NULL`                   | 最后更新的时间戳。             |

## 6. 表：`user_read_pointers`

存储每个用户在每个会话中最后读取的消息序列。

| 列          | 类型          | 约束                  | 描述                               |
| --------------- | ------------- | ---------------------------- | ----------------------------------------- |
| `id`            | `BIGINT`      | `PRIMARY KEY`                | 记录的唯一标识符。         |
| `user_id`       | `BIGINT`      | `UNIQUE(user_id, conversation_id)` | 用户的 ID。                       |
| `conversation_id`| `VARCHAR(64)` | `UNIQUE(user_id, conversation_id)` | 会话的 ID。               |
| `last_read_seq_id`| `BIGINT`      | `NOT NULL`                   | 最后读取消息的序列 ID。 |
| `updated_at`    | `DATETIME`    | `NOT NULL`                   | 最后更新的时间戳。             |