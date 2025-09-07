# GoChat HTTP & WebSocket API

This document defines the RESTful API and WebSocket protocol for the GoChat system. All client applications (web, mobile) should adhere to these specifications.

## 1. General Information

-   **Base URL**: `/api`
-   **WebSocket URL**: `/ws`
-   **Authentication**: All protected endpoints require a `Bearer Token` in the `Authorization` header.
    -   `Authorization: Bearer <jwt_token>`
-   **Content-Type**: `application/json`

## 2. Common Response Formats

### Success Response

```json
{
  "success": true,
  "message": "Operation successful",
  "data": {}
}
```

### Error Response

```json
{
  "success": false,
  "message": "A description of the error",
  "code": "ERROR_CODE"
}
```

## 3. Authentication API (`/auth`)

### `POST /auth/register`

-   **Description**: Registers a new user.
-   **Request Body**:
    ```json
    {
      "username": "string",
      "password": "string"
    }
    ```
-   **Response**: Returns the new user's information.

### `POST /auth/login`

-   **Description**: Logs in a registered user.
-   **Request Body**:
    ```json
    {
      "username": "string",
      "password": "string"
    }
    ```
-   **Response**: Returns a JWT token and user information.

### `POST /auth/guest`

-   **Description**: Logs in a guest user.
-   **Request Body**:
    ```json
    {
      "guestName": "string" // Optional, a random name is generated if not provided
    }
    ```
-   **Response**: Returns a JWT token and guest user information.

### `POST /auth/logout`

-   **Description**: Logs out the current user.
-   **Authentication**: Required.
-   **Response**: Confirms successful logout.

## 4. User API (`/users`)

### `GET /users/info`

-   **Description**: Retrieves the profile of the currently authenticated user.
-   **Authentication**: Required.
-   **Response**: Returns the user's profile information.

## 5. Conversation API (`/conversations`)

### `GET /conversations`

-   **Description**: Retrieves the conversation list for the current user.
-   **Authentication**: Required.
-   **Response**: A list of conversation objects.

### `POST /conversations`

-   **Description**: Creates a new private (one-on-one) conversation.
-   **Authentication**: Required.
-   **Request Body**:
    ```json
    {
      "targetUserId": "string"
    }
    ```
-   **Response**: The newly created conversation object.

### `GET /conversations/{conversationId}/messages`

-   **Description**: Retrieves the message history for a conversation.
-   **Authentication**: Required.
-   **Query Parameters**: `page`, `size`.
-   **Response**: A paginated list of messages.

### `PUT /conversations/{conversationId}/read`

-   **Description**: Marks all messages in a conversation as read.
-   **Authentication**: Required.
-   **Response**: Success confirmation.

## 6. Group API (`/groups`)

### `POST /groups`

-   **Description**: Creates a new group chat.
-   **Authentication**: Required.
-   **Request Body**:
    ```json
    {
      "groupName": "string",
      "members": ["userId1", "userId2"]
    }
    ```
-   **Response**: The newly created group and conversation objects.

### `GET /groups/{groupId}`

-   **Description**: Retrieves detailed information about a group.
-   **Authentication**: Required.
-   **Response**: The group's information and member list.

## 7. WebSocket Protocol

### Connection

-   **URL**: `ws://<host>/ws?token={jwt_token}`
-   The JWT token is passed as a query parameter for authentication.

### Message Types (Client -> Server)

-   **Send Message**:
    ```json
    {
      "type": "send-message",
      "data": {
        "conversationId": "string",
        "content": "string",
        "messageType": "text",
        "tempMessageId": "string" // Client-generated temporary ID
      }
    }
    ```
-   **Heartbeat**:
    ```json
    { "type": "ping" }
    ```

### Message Types (Server -> Client)

-   **New Message**:
    ```json
    {
      "type": "new-message",
      "data": { /* Message Object */ }
    }
    ```
-   **Message Acknowledgment**:
    ```json
    {
      "type": "message-ack",
      "data": {
        "tempMessageId": "string", // Client's temporary ID
        "messageId": "string"      // Server-generated final ID
      }
    }
    ```
-   **Heartbeat Response**:
    ```json
    { "type": "pong" }
    ```
-   **Error Notification**:
    ```json
    {
      "type": "error",
      "data": {
        "message": "Error description",
        "code": "ERROR_CODE"
      }
    }
