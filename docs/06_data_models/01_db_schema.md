# GoChat Data Models

This document defines the database schema for the GoChat system. The primary database is MySQL.

## 1. Table: `users`

Stores user account information.

| Column         | Type          | Constraints      | Description                               |
| -------------- | ------------- | ---------------- | ----------------------------------------- |
| `id`           | `BIGINT`      | `PRIMARY KEY`    | Unique identifier for the user.           |
| `username`     | `VARCHAR(50)` | `UNIQUE, NOT NULL` | The user's login name.                    |
| `password_hash`| `VARCHAR(255)`| `NOT NULL`       | The user's hashed password.               |
| `nickname`     | `VARCHAR(50)` |                  | The user's display name.                  |
| `avatar_url`   | `VARCHAR(255)`|                  | URL to the user's avatar image.           |
| `is_guest`     | `BOOLEAN`     | `NOT NULL, DEFAULT false` | `true` if the user is a guest account. |
| `created_at`   | `DATETIME`    | `NOT NULL`       | Timestamp of when the user was created.   |
| `updated_at`   | `DATETIME`    | `NOT NULL`       | Timestamp of the last update.             |

## 2. Table: `conversations`

Stores information about each conversation, which can be a single chat or a group chat.

| Column          | Type          | Constraints   | Description                               |
| --------------- | ------------- | ------------- | ----------------------------------------- |
| `id`            | `VARCHAR(64)` | `PRIMARY KEY` | Unique identifier for the conversation.   |
| `type`          | `INT`         | `NOT NULL`    | The type of conversation (1: single, 2: group). |
| `last_message_id`| `BIGINT`      |               | The ID of the last message in the conversation. |
| `created_at`    | `DATETIME`    | `NOT NULL`    | Timestamp of when the conversation was created. |
| `updated_at`    | `DATETIME`    | `NOT NULL`    | Timestamp of the last update.             |

## 3. Table: `groups`

Stores information about group chats.

| Column        | Type          | Constraints      | Description                               |
| ------------- | ------------- | ---------------- | ----------------------------------------- |
| `id`          | `BIGINT`      | `PRIMARY KEY`    | Unique identifier for the group.          |
| `name`        | `VARCHAR(50)` | `NOT NULL`       | The name of the group.                    |
| `owner_id`    | `BIGINT`      | `NOT NULL`       | The user ID of the group owner.           |
| `member_count`| `INT`         | `NOT NULL, DEFAULT 0` | The number of members in the group.       |
| `avatar_url`  | `VARCHAR(255)`|                  | URL to the group's avatar image.          |
| `description` | `TEXT`        |                  | A description of the group.               |
| `created_at`  | `DATETIME`    | `NOT NULL`       | Timestamp of when the group was created.  |
| `updated_at`  | `DATETIME`    | `NOT NULL`       | Timestamp of the last update.             |

## 4. Table: `group_members`

Maps users to the groups they are members of.

| Column     | Type     | Constraints                  | Description                               |
| ---------- | -------- | ---------------------------- | ----------------------------------------- |
| `id`       | `BIGINT` | `PRIMARY KEY`                | Unique identifier for the membership record. |
| `group_id` | `BIGINT` | `UNIQUE(group_id, user_id)` | The ID of the group.                      |
| `user_id`  | `BIGINT` | `UNIQUE(group_id, user_id)` | The ID of the user.                       |
| `role`     | `INT`    | `NOT NULL, DEFAULT 1`        | The user's role in the group (1: member, 2: admin, 3: owner). |
| `joined_at`| `DATETIME`| `NOT NULL`                   | Timestamp of when the user joined the group. |

## 5. Table: `messages`

Stores all messages sent in conversations.

| Column          | Type          | Constraints                  | Description                               |
| --------------- | ------------- | ---------------------------- | ----------------------------------------- |
| `id`            | `BIGINT`      | `PRIMARY KEY`                | Unique identifier for the message.        |
| `conversation_id`| `VARCHAR(64)` | `UNIQUE(conversation_id, seq_id)` | The ID of the conversation the message belongs to. |
| `sender_id`     | `BIGINT`      | `NOT NULL`                   | The user ID of the sender.                |
| `message_type`  | `INT`         | `NOT NULL, DEFAULT 1`        | The type of message (e.g., text, image).  |
| `content`       | `TEXT`        | `NOT NULL`                   | The content of the message.               |
| `seq_id`        | `BIGINT`      | `UNIQUE(conversation_id, seq_id)` | A sequential ID within the conversation. |
| `deleted`       | `BOOLEAN`     | `NOT NULL, DEFAULT false`    | `true` if the message has been soft-deleted. |
| `extra`         | `TEXT`        |                              | Extra metadata for the message (JSON).    |
| `created_at`    | `DATETIME`    | `NOT NULL`                   | Timestamp of when the message was created. |
| `updated_at`    | `DATETIME`    | `NOT NULL`                   | Timestamp of the last update.             |

## 6. Table: `user_read_pointers`

Stores the last read message sequence for each user in each conversation.

| Column          | Type          | Constraints                  | Description                               |
| --------------- | ------------- | ---------------------------- | ----------------------------------------- |
| `id`            | `BIGINT`      | `PRIMARY KEY`                | Unique identifier for the record.         |
| `user_id`       | `BIGINT`      | `UNIQUE(user_id, conversation_id)` | The ID of the user.                       |
| `conversation_id`| `VARCHAR(64)` | `UNIQUE(user_id, conversation_id)` | The ID of the conversation.               |
| `last_read_seq_id`| `BIGINT`      | `NOT NULL`                   | The sequence ID of the last read message. |
| `updated_at`    | `DATETIME`    | `NOT NULL`                   | Timestamp of the last update.             |
