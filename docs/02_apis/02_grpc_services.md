# GoChat Internal gRPC Services

This document outlines the gRPC services used for internal communication between microservices. For detailed message and service definitions, please refer to the corresponding `.proto` files.

## 1. Overview

Internal communication in GoChat is handled via gRPC, which provides a high-performance, strongly-typed, and language-agnostic RPC framework. This ensures efficient and reliable communication between the microservices.

-   **`im-logic` Service**: Exposes business logic functions.
-   **`im-repo` Service**: Exposes data persistence functions.

## 2. `im-logic` Services

The `im-logic` microservice exposes several gRPC services that encapsulate the core business logic of the application.

-   **Proto File Location**: `api/proto/im_logic/v1/`

### `AuthService`

-   **Description**: Handles all user authentication and token management logic.
-   **Proto File**: [`auth.proto`](../../../api/proto/im_logic/v1/auth.proto)
-   **Key RPCs**:
    -   `Login`: Authenticates a user with a username and password.
    -   `Register`: Creates a new user account.
    -   `GuestLogin`: Creates a temporary guest account.
    -   `ValidateToken`: Validates a JWT access token.

### `ConversationService`

-   **Description**: Manages conversation-related logic, such as fetching conversation lists and messages.
-   **Proto File**: [`conversation.proto`](../../../api/proto/im_logic/v1/conversation.proto)
-   **Key RPCs**:
    -   `GetConversations`: Retrieves a user's conversation list.
    -   `CreateConversation`: Creates a new private conversation.
    -   `GetMessages`: Fetches the message history for a conversation.
    -   `MarkAsRead`: Updates a user's read pointer in a conversation.

### `GroupService`

-   **Description**: Manages group chat logic, including creation, membership, and information retrieval.
-   **Proto File**: [`group.proto`](../../../api/proto/im_logic/v1/group.proto)
-   **Key RPCs**:
    -   `CreateGroup`: Creates a new group.
    -   `GetGroup`: Retrieves detailed information about a group.
    -   `GetGroupMembers`: Fetches the member list of a group.
    -   `JoinGroup`, `LeaveGroup`: Manages group membership.

### `MessageService`

-   **Description**: Handles the logic for sending messages.
-   **Proto File**: [`message.proto`](../../../api/proto/im_logic/v1/message.proto)
-   **Key RPCs**:
    -   `SendMessage`: Processes an outgoing message, saves it, and triggers fan-out.

## 3. `im-repo` Services

The `im-repo` microservice exposes gRPC services for data access, abstracting the database and cache from the business logic layer.

-   **Proto File Location**: `api/proto/im_repo/v1/`

### `UserService`

-   **Description**: Provides CRUD operations for user data.
-   **Proto File**: [`user.proto`](../../../api/proto/im_repo/v1/user.proto)
-   **Key RPCs**:
    -   `CreateUser`: Inserts a new user record into the database.
    -   `GetUser`: Retrieves a user by their ID.
    -   `GetUserByUsername`: Retrieves a user by their username.
    -   `VerifyPassword`: Verifies a user's password hash.

### `ConversationService`

-   **Description**: Provides data access operations for conversations.
-   **Proto File**: [`conversation.proto`](../../../api/proto/im_repo/v1/conversation.proto)
-   **Key RPCs**:
    -   `CreateConversation`: Creates a new conversation record.
    -   `GetUserConversations`: Retrieves the conversation IDs a user is part of.
    -   `UpdateReadPointer`: Updates a user's read progress in the database.

### `GroupService`

-   **Description**: Provides data access operations for groups and their members.
-   **Proto File**: [`group.proto`](../../../api/proto/im_repo/v1/group.proto)
-   **Key RPCs**:
    -   `CreateGroup`: Creates a new group record.
    -   `GetGroup`: Retrieves group information from the database.
    -   `AddGroupMember`, `RemoveGroupMember`: Manages group membership records.

### `MessageService`

-   **Description**: Provides data access operations for messages.
-   **Proto File**: [`message.proto`](../../../api/proto/im_repo/v1/message.proto)
-   **Key RPCs**:
    -   `SaveMessage`: Saves a message to the database.
    -   `GetConversationMessages`: Retrieves a list of messages for a conversation.

### `OnlineStatusService`

-   **Description**: Manages user online status, primarily using Redis.
-   **Proto File**: [`online_status.proto`](../../../api/proto/im_repo/v1/online_status.proto)
-   **Key RPCs**:
    -   `SetUserOnline`, `SetUserOffline`: Updates a user's online status.
    -   `GetUserOnlineStatus`: Retrieves the online status for a user.
