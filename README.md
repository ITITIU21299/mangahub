# MangaHub

A full-stack manga tracking and management system with real-time synchronization, notifications, and community chat features.

## Features

- 🔍 **Manga Discovery**: Search and browse manga from MangaDex API
- 📚 **Library Management**: Track reading progress, manage your collection
- 🔄 **Real-time Sync**: Multi-device synchronization via TCP
- 🔔 **Notifications**: Chapter release notifications via UDP
- 💬 **Chat System**: Real-time community chat via WebSocket
- 🔐 **Authentication**: Secure JWT-based user authentication
- 📊 **Progress Tracking**: Detailed reading progress with status management

## Tech Stack

- **Frontend**: Next.js 16, React 19, TypeScript, Tailwind CSS
- **Backend**: Go 1.21+, Gin Framework, SQLite
- **Protocols**: HTTP REST, TCP, UDP, gRPC, WebSocket
- **External API**: MangaDex API

## Quick Start

### Prerequisites

- Go 1.21 or later
- Node.js 18.x or later
- Ports 8080, 9090, 9091, 9092, 9093 available

### Backend Setup

```bash
# Install dependencies
go mod download

# Build server
go build -o all-servers ./cmd/all-servers

# Run server (starts all services)
./all-servers
```

### Frontend Setup

```bash
# Install dependencies
npm install

# Run development server
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

## Project Structure

```
mangahub/
├── app/                    # Next.js frontend pages
│   ├── api/               # Next.js API routes
│   ├── auth/              # Authentication pages
│   ├── discover/          # Manga discovery page
│   ├── library/           # User library page
│   ├── manga/             # Manga details page
│   └── chat/              # Chat page
├── cmd/                    # Go executables
│   ├── all-servers/       # Main server (all services)
│   ├── grpc-client/       # gRPC client example
│   └── udp-client/        # UDP client example
├── internal/              # Backend services
│   ├── auth/              # Authentication service
│   ├── manga/             # Manga service
│   ├── user/              # User/library service
│   ├── database/          # Database layer
│   ├── tcp/               # TCP server
│   ├── udp/               # UDP server
│   ├── grpc/              # gRPC server
│   └── mangadex/          # MangaDex API client
├── pkg/                   # Shared packages
│   └── models/            # Data models
└── proto/                 # gRPC protocol definitions
```

## Services

The backend runs 5 concurrent services:

- **HTTP API** (Port 8080): REST API for web frontend
- **TCP Server** (Port 9090): Real-time progress synchronization
- **UDP Server** (Port 9091): Chapter release notifications
- **gRPC Server** (Port 9092): Internal service communication
- **WebSocket Server** (Port 9093): Real-time chat

## Setup Instructions

### Prerequisites

1. Install Go: https://golang.org/dl/
2. Install Node.js: https://nodejs.org/
3. Ensure ports 8080, 9090, 9091, 9092, 9093 are available

### Backend Setup

1. **Clone and Navigate**

   ```bash
   cd /path/to/mangahub
   ```

2. **Install Go Dependencies**

   ```bash
   go mod download
   ```

3. **Build the Server**

   ```bash
   go build -o all-servers ./cmd/all-servers
   ```

4. **Configure Environment Variables** (Optional)

   ```bash
   NEXT_PUBLIC_API_BASE=http://10.238.58.210:8080
   NEXT_PUBLIC_WS_URL=ws://10.238.58.210:9093/ws
   MANGAHUB_CORS_ORIGINS=http://10.238.58.210:3000,http://localhost:3000,http://0.0.0.0:3000
   ```

   Change the ip(10.238.58.210) to your ip.

5. **Run the Server**

   ```bash
   ./all-servers
   # Or on Windows:
   all-servers.exe
   ```

   The server will start all services:

   - ✅ HTTP API server listening on :8080
   - ✅ TCP server listening on :9090
   - ✅ UDP server listening on :9091
   - ✅ gRPC server listening on :9092
   - ✅ WebSocket server listening on :9093/ws

### Frontend Setup

1. **Navigate to Project Root**

   ```bash
   cd /path/to/mangahub
   ```

2. **Install Dependencies**

   ```bash
   npm ci
   ```

3. **Run Development Server**

   ```bash
   npx next build

   npx next start
   ```

   Frontend will be available at: http://<YOUR_IP>:3000

### Database Initialization

The database is automatically initialized on first server start. The schema includes:

- `users` table
- `manga` table
- `user_progress` table
- `user_notifications` table

Database file location: `./mangahub.db` (or path specified in `MANGAHUB_DB_PATH`)

---

## Running and Testing Guide

### Sign in and sign up

User can sign in and sign on the website.

### Complete System Startup

To run the entire MangaHub system:

1. **Terminal 1: Start Backend Server**

   ```bash
   cd /path/to/mangahub
   go build -o all-servers ./cmd/all-servers
   ./all-servers
   ```

   Expected output:

   ```
   ✅ HTTP API server listening on :8080
   ✅ TCP server listening on :9090
   ✅ UDP server listening on :9091
   ✅ gRPC server listening on :9092
   ✅ WebSocket server listening on :9093/ws
   ```

2. **Terminal 2: Start Frontend**

   ```bash
   cd /path/to/mangahub
   npx next build
   ```

   Expected output:

   ```
   ▲ Next.js 16.0.10
   - Network:        http://<YOUR_IP>:3000
   ```

3. **Verify Services**
   - All 5 services should be running

---

### Testing Chat (WebSocket)

The chat system uses WebSocket on port 9093. You can test it through the web interface or programmatically.

#### Method: Web Interface Testing

1. **Start the system** (backend + frontend)

2. **Open Chat Page**

   - Navigate to http://<YOUR_IP>:3000/chat (must be in Lan together)
   - Or click "Chat" in the bottom navigation

3. **Test Basic Chat**

   - The page automatically connects using your username from JWT token
   - Type a message in the input field
   - Press Enter or click "Send"
   - Your message should appear in the chat

4. **Test Multiple Users**

   - Open http://<YOUR_IP>:3000/chat in multiple browser windows/tabs
   - Or use incognito/private windows
   - Messages sent from one window should appear in all windows
   - Each user will see "join" and "leave" messages

5. **Test Typing Indicators**
   - Start typing in one window
   - Other windows should show "username is typing..." indicator
   - Indicator disappears after 2 seconds of inactivity

**Expected Behavior:**

- Messages appear in real-time across all connected clients
- Join/leave notifications when users connect/disconnect
- Typing indicators show when users are typing
- Message history is maintained (last 50 messages)

---

### Testing gRPC

The gRPC server runs on port 9092. Use the provided client tool to test it.

#### Build gRPC Client

```bash
cd /path/to/mangahub
go build -o grpc-client ./cmd/grpc-client
```

#### Test 1: Get Manga by ID

First, you need a valid manga ID. You can get one from the MangaDex API or use a known ID.

```bash
# Example: Get manga information
./bin/grpc-client -action=get -manga-id="manga-123"
```

**Expected Output:**

```
✅ Retrieved manga via gRPC:
   ID: mangadex-a1c7c817-4e59-43b7-9365-09675a149a6f
   Title: One Piece
   Author: Oda Eiichiro
   Status: ongoing
   Chapters: 1100
```

#### Test 2: Search Manga

```bash
# Search for manga
./bin/grpc-client -action=search -query="One Piece" -page=1

# Search with genre filter
./bin/grpc-client -action=search -genre="Action" -page=1
```

**Expected Output:**

```
✅ Search results via gRPC:
   Total: 150
   Page: 1/8
   Results: 20

   1. One Piece by Oda Eiichiro
   2. One Piece - Romance Dawn by Oda Eiichiro
   3. One Piece - Ace's Story by Oda Eiichiro
   ... and 17 more
```

#### Test 3: Update Progress

First, ensure you have:

1. A registered user (get user ID from registration)
2. A manga added to your library

```bash
# Update reading progress
# Update user progress (requires valid user_id and manga_id)
./bin/grpc-client -action=update -user-id="user-123" -manga-id="manga-456" -chapter=10
```

**Expected Output:**

```
✅ Progress updated via gRPC:
   Success: true
   Message: Progress updated successfully
   Broadcast sent: true
```

**Note:** The `broadcast_sent` field indicates whether the TCP broadcast was successfully sent for real-time synchronization.

### Testing Notifications (UDP)

The UDP notification server runs on port 9091. Test it using the provided UDP client.

#### Build UDP Client

```bash
cd /path/to/mangahub
go build -o udp-client ./cmd/udp-client
```

**Note:** If you run all the servers you don't need to run this command.

#### Test 1: Register for Notifications

Register a user to receive notifications:

```bash
# Register user for notifications
go run cmd/udp-client/main.go \
  -mode=register \
  -user="user_johndoe" \
  -addr=localhost:9091
```

**Expected Output:**

```
Registration response: registered
```

**What happens:**

- User is registered with the UDP server
- Server stores the user's address for future notifications
- User will receive notifications for subscribed manga

#### Test 2: Send Notification (Admin/Testing)

Simulate a chapter release notification:

```bash
# Send a notification (simulates admin action)
go run cmd/udp-client/main.go \
  -mode=notify \
  -addr=localhost:9091 \
  -manga=d68ceffd-ac56-45db-9129-3413dd0d7063 \
  -title="Isekai de Te ni Ireta Seisan Skill wa Saikyou datta You desu ~Souzou & Kiyou no W Chiuto de Musou Suru~" \
  -chapter=61 \
  -message="New chapter 61 released"
  -addr=localhost:9091
```

**Expected Output:**

```
Notification sent.
```

**What happens:**

- Server broadcasts notification to all registered users
- Users subscribed to this manga receive the notification
- Check server logs to see delivery status

**Expected Behavior:**

- Subscription is stored in database
- UDP registration happens automatically
- Notifications are delivered to registered users
- Unsubscribe removes both database entry and UDP registration

---

### Testing TCP Progress Synchronization

The TCP server (port 9090) handles real-time progress synchronization.

#### Test 1: Integration with Website

1. **User open website**

2. **User choose website to update progress**

3. **User recieve a successful message**

**Expected output**

```
Progress updated succesfully
```

#### Test 2: Integration with HTTP API

The TCP broadcast is automatically triggered when updating progress via HTTP API:

1. **Connect TCP client**

2. **Update progress via HTTP API:**

   ```bash
   curl -X PUT http://localhost:8080/users/progress \
     -H "Authorization: Bearer <your-token>" \
     -H "Content-Type: application/json" \
     -d '{
       "manga_id": "mangadex-test123",
       "current_chapter": 101,
       "status": "Reading"
     }'
   ```

3. **TCP client should receive broadcast:**
   ```json
   {
     "type": "progress",
     "user_id": "user_johndoe",
     "manga_id": "mangadex-test123",
     "chapter": 101,
     "timestamp": 1705766400
   }
   ```

**Expected Behavior:**

- HTTP API updates database
- HTTP API triggers TCP broadcast
- All connected TCP clients for that user receive the update
- Real-time synchronization across devices

---
