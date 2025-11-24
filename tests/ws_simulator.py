#!/usr/bin/env python3
"""
WebSocket Client Simulator for Oculo Pilot Go Server
Tests Python-compatible handshake protocol and message routing
"""

import asyncio
import json
import sys
import time
import websockets
from typing import Optional

class OculoPilotClient:
    """Simulates Raspberry Pi WebSocket client"""

    def __init__(self, server_url: str, auth_token: str, client_type: str = "control"):
        self.server_url = server_url
        self.auth_token = auth_token
        self.client_type = client_type  # web, video, control, telemetry
        self.ws = None
        self.connection_id = None
        self.connected = False

    async def connect(self):
        """Connect to WebSocket server with JWT authentication"""
        # Add token to URL
        url_with_token = f"{self.server_url}?token={self.auth_token}"

        try:
            print(f"🔌 Connecting to {self.server_url}...")
            self.ws = await websockets.connect(url_with_token)
            print(f"✅ Connected successfully")
            return True
        except Exception as e:
            print(f"❌ Connection failed: {e}")
            return False

    async def handshake(self):
        """Perform handshake protocol (Python-compatible)"""
        try:
            # Wait for handshake request from server
            message = await asyncio.wait_for(self.ws.recv(), timeout=5.0)
            data = json.loads(message)

            if data.get("type") != "handshake_request":
                print(f"❌ Unexpected message: {data.get('type')}")
                return False

            self.connection_id = data.get("connection_id")
            supported_types = data.get("supported_client_types", [])

            print(f"📨 Handshake request received:")
            print(f"   Connection ID: {self.connection_id}")
            print(f"   Supported types: {supported_types}")

            # Validate client type is supported
            if self.client_type not in supported_types:
                print(f"❌ Client type '{self.client_type}' not supported")
                return False

            # Send handshake response
            response = {
                "type": "handshake_response",
                "connection_id": self.connection_id,
                "client_type": self.client_type
            }
            await self.ws.send(json.dumps(response))
            print(f"📤 Sent handshake response: type={self.client_type}")

            # Wait for connection established confirmation
            message = await asyncio.wait_for(self.ws.recv(), timeout=5.0)
            data = json.loads(message)

            if data.get("type") == "connection_established":
                print(f"✅ Handshake completed!")
                print(f"   Status: {data.get('status')}")
                print(f"   Client type: {data.get('client_type')}")
                print(f"   Video available: {data.get('video_clients_available')}")
                self.connected = True
                return True
            else:
                print(f"❌ Unexpected response: {data}")
                return False

        except asyncio.TimeoutError:
            print(f"❌ Handshake timeout")
            return False
        except Exception as e:
            print(f"❌ Handshake error: {e}")
            return False

    async def send_message(self, message_type: str, data: dict = None):
        """Send message to server"""
        if not self.connected:
            print("❌ Not connected")
            return False

        message = {
            "type": message_type,
            "timestamp": time.time()
        }
        if data:
            message.update(data)

        try:
            await self.ws.send(json.dumps(message))
            print(f"📤 Sent: {message_type}")
            return True
        except Exception as e:
            print(f"❌ Send error: {e}")
            return False

    async def send_control_command(self):
        """Send control command (simulating Raspberry Pi control client)"""
        command = {
            "type": "control_command",
            "input_type": "gamepad",
            "data": {
                "action": "move_forward",
                "timestamp": time.time()
            },
            "axes": {
                "left_stick_x": 0.0,
                "left_stick_y": 0.5
            },
            "buttons": {}
        }
        await self.ws.send(json.dumps(command))
        print(f"🎮 Sent control command: move_forward")

    async def send_ping(self):
        """Send ping message"""
        await self.send_message("ping", {"timestamp": time.time()})

    async def listen(self, duration: int = 10):
        """Listen for messages from server"""
        print(f"\n👂 Listening for messages ({duration}s)...")
        start = time.time()

        while time.time() - start < duration:
            try:
                message = await asyncio.wait_for(self.ws.recv(), timeout=1.0)
                data = json.loads(message)
                print(f"📨 Received: {data.get('type')}")

                if data.get("type") == "pong":
                    latency = time.time() - data.get("timestamp", 0)
                    print(f"   Latency: {latency*1000:.2f}ms")

            except asyncio.TimeoutError:
                continue
            except websockets.exceptions.ConnectionClosed:
                print("❌ Connection closed by server")
                break
            except Exception as e:
                print(f"❌ Receive error: {e}")
                break

    async def close(self):
        """Close WebSocket connection"""
        if self.ws:
            await self.ws.close()
            print("👋 Connection closed")


async def test_handshake(server_url: str, token: str):
    """Test 1: Handshake protocol"""
    print("\n" + "="*60)
    print("TEST 1: Handshake Protocol (control client)")
    print("="*60)

    client = OculoPilotClient(server_url, token, client_type="control")

    if await client.connect():
        if await client.handshake():
            await asyncio.sleep(2)
            await client.close()
            return True

    return False


async def test_control_messages(server_url: str, token: str):
    """Test 2: Control messages"""
    print("\n" + "="*60)
    print("TEST 2: Control Messages")
    print("="*60)

    client = OculoPilotClient(server_url, token, client_type="control")

    if await client.connect() and await client.handshake():
        # Send control commands
        for i in range(3):
            await client.send_control_command()
            await asyncio.sleep(1)

        await client.close()
        return True

    return False


async def test_ping_pong(server_url: str, token: str):
    """Test 3: Ping/Pong"""
    print("\n" + "="*60)
    print("TEST 3: Ping/Pong")
    print("="*60)

    client = OculoPilotClient(server_url, token, client_type="web")

    if await client.connect() and await client.handshake():
        # Send pings and measure latency
        for i in range(5):
            await client.send_ping()
            await asyncio.sleep(0.5)

        await client.listen(duration=3)
        await client.close()
        return True

    return False


async def test_video_client(server_url: str, token: str):
    """Test 4: Video client connection"""
    print("\n" + "="*60)
    print("TEST 4: Video Client Connection")
    print("="*60)

    client = OculoPilotClient(server_url, token, client_type="video")

    if await client.connect() and await client.handshake():
        print("✅ Video client registered successfully")
        await asyncio.sleep(2)
        await client.close()
        return True

    return False


async def main():
    """Main test runner"""
    print("\n🧪 Oculo Pilot WebSocket Simulator")
    print("Testing Go Server Python Compatibility")
    print()

    # Configuration
    if len(sys.argv) > 1:
        server_url = sys.argv[1]
    else:
        server_url = "ws://localhost:8080/ws"

    if len(sys.argv) > 2:
        token = sys.argv[2]
    else:
        # Get token by logging in
        print("⚠️  No token provided. Please login first:")
        print(f"   curl -X POST http://localhost:8080/api/login \\")
        print(f'     -H "Content-Type: application/json" \\')
        print(f'     -d \'{{"username":"admin","password":"admin123"}}\'')
        print()
        print("Usage: python ws_simulator.py [ws://server:port/ws] [JWT_TOKEN]")
        return

    print(f"Server: {server_url}")
    print(f"Token: {token[:20]}...")

    # Run tests
    results = []

    try:
        results.append(("Handshake Protocol", await test_handshake(server_url, token)))
        results.append(("Control Messages", await test_control_messages(server_url, token)))
        results.append(("Ping/Pong", await test_ping_pong(server_url, token)))
        results.append(("Video Client", await test_video_client(server_url, token)))

    except KeyboardInterrupt:
        print("\n⚠️  Tests interrupted by user")

    # Print results
    print("\n" + "="*60)
    print("TEST RESULTS")
    print("="*60)

    passed = sum(1 for _, result in results if result)
    total = len(results)

    for test_name, result in results:
        status = "✅ PASS" if result else "❌ FAIL"
        print(f"{status} - {test_name}")

    print(f"\nTotal: {passed}/{total} tests passed")

    if passed == total:
        print("🎉 All tests passed!")
        sys.exit(0)
    else:
        print("❌ Some tests failed")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
