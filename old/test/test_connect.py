import asyncio
import json
import time
from datetime import datetime
import websockets
import logging

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('gochat-client')


class GoChatClient:
    def __init__(self, server_url, user_id, room_id, token):
        """
        Initialize GoChat WebSocket client

        Args:
            server_url (str): WebSocket server URL (e.g., 'ws://localhost:8080/ws')
            user_id (int): User ID for authentication
            room_id (int): Room ID to join
            token (str): Authentication token
        """
        self.server_url = server_url
        self.user_id = user_id
        self.room_id = room_id
        self.token = token
        self.websocket = None
        self.last_activity = time.time()
        self.connected = False
        self.running = False
        self.message_handlers = {
            "success": self._handle_connection_success,
            "fail": self._handle_connection_failure,
            "pushmsg": self._handle_chat_message,
            "roominfo": self._handle_room_info,
        }

    async def connect(self):
        """Establish connection with the WebSocket server"""
        try:
            self.websocket = await websockets.connect(self.server_url)
            self.running = True

            # Send connection request
            connect_msg = {
                "user_id": self.user_id,
                "room_id": self.room_id,
                "token": self.token,
                "msg_type": "connect"
            }
            await self.websocket.send(json.dumps(connect_msg))
            logger.info(
                f"Connection request sent for user {self.user_id} to room {self.room_id}")

            # Start tasks
            tasks = [
                asyncio.create_task(self._message_listener()),
                asyncio.create_task(self._heartbeat_sender())
            ]

            await asyncio.gather(*tasks)

        except Exception as e:
            logger.error(f"Connection error: {e}")
            self.running = False
            if self.websocket:
                await self.websocket.close()

    async def disconnect(self):
        """Gracefully disconnect from the server"""
        logger.info("Disconnecting from server...")
        self.running = False
        if self.websocket:
            await self.websocket.close()

    async def send_message(self, message):
        """
        Send a custom message through the connection

        Args:
            message (dict): Message to send
        """
        if not self.connected:
            logger.warning("Cannot send message: Not connected")
            return

        await self.websocket.send(json.dumps(message))
        self.last_activity = time.time()

    async def _message_listener(self):
        """Listen for incoming messages from the server"""
        while self.running:
            try:
                message = await self.websocket.recv()
                self.last_activity = time.time()

                # Parse the message
                try:
                    data = json.loads(message)

                    # Handle ping/pong messages
                    if isinstance(data, dict) and "type" in data:
                        if data["type"] == "ping":
                            await self._handle_ping()
                            continue

                    # Process other messages
                    if "msg_type" in data:
                        msg_type = data["msg_type"]
                        if msg_type in self.message_handlers:
                            await self.message_handlers[msg_type](data)
                        else:
                            logger.warning(f"Unknown message type: {msg_type}")
                    else:
                        logger.warning(
                            f"Received message without msg_type: {data}")

                except json.JSONDecodeError:
                    logger.error(f"Invalid JSON received: {message}")

            except websockets.exceptions.ConnectionClosed:
                logger.info("Connection closed by server")
                self.running = False
                break

            except Exception as e:
                logger.error(f"Error in message listener: {e}")

    async def _heartbeat_sender(self):
        """Send periodic heartbeat messages to keep the connection alive"""
        while self.running:
            try:
                # Send heartbeat slightly before server's 30-second interval
                await asyncio.sleep(25)

                if not self.running:
                    break

                # Check if connection is still active
                if time.time() - self.last_activity > 55:  # Before server's 60-second timeout
                    logger.warning("Connection seems inactive, sending ping")

                # Send pong proactively
                pong_msg = {"type": "pong"}
                await self.websocket.send(json.dumps(pong_msg))
                logger.debug("Sent heartbeat pong")

            except Exception as e:
                logger.error(f"Error in heartbeat sender: {e}")

    async def _handle_ping(self):
        """Handle ping message from server"""
        try:
            pong_msg = {"type": "pong"}
            await self.websocket.send(json.dumps(pong_msg))
            logger.debug("Responded to ping with pong")
        except Exception as e:
            logger.error(f"Error sending pong: {e}")

    async def _handle_connection_success(self, data):
        """Handle successful connection"""
        self.connected = True
        logger.info(f"Successfully connected to room {self.room_id}")

    async def _handle_connection_failure(self, data):
        """Handle connection failure"""
        self.connected = False
        logger.error("Connection failed")
        self.running = False

    async def _handle_chat_message(self, data):
        """Handle incoming chat message"""
        message_data = data.get("data", {})
        sender_id = message_data.get("from_user_id")
        sender_name = message_data.get("from_user_name")
        message = message_data.get("message")

        logger.info(f"Message from {sender_name} ({sender_id}): {message}")
        # Implement your custom message handling here

    async def _handle_room_info(self, data):
        """Handle room information update"""
        room_data = data.get("data", {})
        room_id = room_data.get("room_id")
        user_count = room_data.get("count")
        user_info = room_data.get("user_info", {})

        logger.info(f"Room {room_id} update: {user_count} users")
        logger.info(f"Users in room: {', '.join(user_info.values())}")
        # Implement your custom room info handling here


async def main():
    # Configuration
    server_url = "ws://localhost:8081/ws"  # Replace with your server address
    user_id = 123                          # Replace with your user ID
    room_id = 456                          # Replace with your room ID
    token = "your-auth-token"              # Replace with your auth token

    # Create and run client
    client = GoChatClient(server_url, user_id, room_id, token)

    try:
        await client.connect()
    except KeyboardInterrupt:
        logger.info("Interrupted by user")
    finally:
        await client.disconnect()


if __name__ == "__main__":
    # Run the client
    asyncio.run(main())
