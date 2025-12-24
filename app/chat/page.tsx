"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import BottomNav from "@/components/BottomNav";

type ChatMessageType =
  | "chat"
  | "typing"
  | "join"
  | "leave"
  | "friend_request"
  | "friend_response"
  | "system";

type ChatMessage = {
  type: ChatMessageType;
  username: string;
  message?: string;
  timestamp: number;
  room?: string;
  target?: string;
};

const WS_URL =
  process.env.NEXT_PUBLIC_WS_URL?.replace(/\/$/, "") ||
  "ws://localhost:9093/ws";

function formatTime(ts: number) {
  const d = new Date(ts * 1000);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

type Conversation = {
  id: string;
  label: string;
  description: string;
  type: "general" | "group" | "dm";
};

export default function ChatPage() {
  const [username, setUsername] = useState("");
  const [connected, setConnected] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [typingUsers, setTypingUsers] = useState<Set<string>>(new Set());
  const [selectedRoom, setSelectedRoom] = useState<string>("general");
  const [conversations, setConversations] = useState<Conversation[]>([
    {
      id: "general",
      label: "#general",
      description: "Everyone in MangaHub",
      type: "general",
    },
  ]);

  const wsRef = useRef<WebSocket | null>(null);
  const typingTimeouts = useRef<Record<string, NodeJS.Timeout>>({});

  const activeName = useMemo(() => {
    const trimmed = username.trim();
    return trimmed.length > 0 ? trimmed : "guest";
  }, [username]);

  // Load username from JWT or fallback keys
  useEffect(() => {
    if (typeof window === "undefined") return;

    let name: string | null = null;
    const token = window.localStorage.getItem("mangahub_token");
    if (token) {
      try {
        const payloadPart = token.split(".")[1];
        if (payloadPart) {
          const decoded = JSON.parse(
            atob(payloadPart.replace(/-/g, "+").replace(/_/g, "/"))
          );
          if (decoded && typeof decoded.usr === "string") {
            name = decoded.usr;
          }
        }
      } catch {}
    }

    if (!name) {
      name =
        window.localStorage.getItem("mangahub_username") ||
        window.localStorage.getItem("mangahub_chat_name") ||
        "guest";
    }

    setUsername(name);
  }, []);

  const connect = () => {
    // Wait for username to be loaded
    if (!username || username.trim() === "") {
      return;
    }
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) return;

    const url = `${WS_URL}?username=${encodeURIComponent(activeName)}`;
    console.log("[Chat] Connecting with username:", activeName);
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
    };

    ws.onclose = () => {
      setConnected(false);
      wsRef.current = null;
    };

    ws.onerror = () => {
      setConnected(false);
    };

    ws.onmessage = (event) => {
      try {
        const payload: ChatMessage = JSON.parse(event.data);
        const room = payload.room || "general";
        console.log(
          "[Chat] Received message:",
          payload.type,
          "from",
          payload.username,
          "in room",
          room
        );

        if (payload.type === "typing") {
          if (payload.username !== activeName) {
            handleTypingIndicator(room, payload.username);
          }
          return;
        }

        if (
          payload.type === "friend_request" &&
          payload.target === activeName
        ) {
          setMessages((prev) => [...prev, payload]);
          return;
        }

        // Friend response: add DM conversation for both parties
        if (payload.type === "friend_response") {
          const friend =
            payload.username === activeName ? payload.target : payload.username;
          if (friend) {
            const id = `dm:${friend}`;
            const conv: Conversation = {
              id,
              label: friend,
              description: "Direct message",
              type: "dm",
            };
            setConversations((prev) => {
              if (prev.some((c) => c.id === id)) return prev;
              return [conv, ...prev];
            });
          }
          setMessages((prev) => [...prev, payload]);
          return;
        }

        // Normal chat/join/leave/system messages
        setTypingUsers((prev) => {
          const next = new Set(prev);
          next.delete(`${room}:${payload.username}`);
          return next;
        });
        setMessages((prev) => [...prev, payload]);
      } catch (e) {
        console.error("Bad message", e);
      }
    };
  };

  useEffect(() => {
    connect();
    return () => {
      wsRef.current?.close();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [username, activeName]);

  const handleTypingIndicator = (room: string, user: string) => {
    const key = `${room}:${user}`;
    setTypingUsers((prev) => {
      const next = new Set(prev);
      next.add(key);
      return next;
    });

    if (typingTimeouts.current[key]) {
      clearTimeout(typingTimeouts.current[key]);
    }
    typingTimeouts.current[key] = setTimeout(() => {
      setTypingUsers((prev) => {
        const next = new Set(prev);
        next.delete(key);
        return next;
      });
    }, 2000);
  };

  const sendMessage = (
    type: ChatMessageType,
    message?: string,
    target?: string
  ) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.log("[Chat] Cannot send: WebSocket not open");
      return;
    }
    const room = selectedRoom === "friends" ? "general" : selectedRoom;
    const payload: ChatMessage = {
      type,
      username: activeName,
      message,
      timestamp: Math.floor(Date.now() / 1000),
      room,
      target,
    };
    console.log(
      "[Chat] Sending message:",
      type,
      "as",
      activeName,
      "to room",
      room
    );
    wsRef.current.send(JSON.stringify(payload));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim()) return;
    sendMessage("chat", input.trim());
    setInput("");
  };

  const handleTyping = (value: string) => {
    setInput(value);
    sendMessage("typing");
  };

  const typingText = useMemo(() => {
    const currentRoomKeys = Array.from(typingUsers).filter((key) =>
      key.startsWith(`${selectedRoom}:`)
    );
    if (currentRoomKeys.length === 0) return "";
    const names = currentRoomKeys.map((key) => key.split(":")[1]);
    const others = names.filter((n) => n !== activeName);
    if (others.length === 0) return "";
    if (others.length === 1) {
      return `${others[0]} is typing...`;
    }
    if (others.length <= 3) {
      return `${others.join(", ")} are typing...`;
    }
    return "Multiple users are typing...";
  }, [typingUsers, selectedRoom, activeName]);

  const handleAcceptFriend = (from: string) => {
    const id = `dm:${from}`;
    const conv: Conversation = {
      id,
      label: from,
      description: "Direct message",
      type: "dm",
    };
    setConversations((prev) => {
      if (prev.some((c) => c.id === id)) return prev;
      return [conv, ...prev];
    });
    setSelectedRoom(id);
    sendMessage("friend_response", `Friend request accepted`, from);
  };

  const handleDeclineFriend = (from: string) => {
    setMessages((prev) => [
      ...prev,
      {
        type: "system",
        username: "System",
        message: `You declined ${from}'s friend request`,
        timestamp: Math.floor(Date.now() / 1000),
        room: "system",
      },
    ]);
  };

  const visibleMessages = messages.filter((msg) => {
    // Friend/system events in "system" room are always visible regardless of selected room
    if (
      msg.type === "friend_request" ||
      msg.type === "friend_response" ||
      msg.type === "system"
    ) {
      return true;
    }
    // Handle "friends" view - show DMs or general room messages
    if (selectedRoom === "friends") {
      // Show general room messages in friends view, or DMs if they exist
      const msgRoom = msg.room || "general";
      return msgRoom === "general" || msgRoom.startsWith("dm:");
    }
    // For DM rooms, only show messages in that specific DM
    if (selectedRoom.startsWith("dm:")) {
      return (msg.room || "general") === selectedRoom;
    }
    // For other rooms, match exactly
    return (msg.room || "general") === selectedRoom;
  });

  return (
    <div className="flex min-h-screen flex-col bg-background-light pb-24 text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
      {/* Header */}
      <header className="sticky top-0 z-20 flex items-center justify-between border-b border-black/5 bg-background-light/90 px-4 py-3 backdrop-blur dark:border-white/10 dark:bg-background-dark/90">
        <div className="flex flex-col">
          <span className="text-xs font-semibold text-text-sub-light dark:text-text-sub-dark">
            MangaHub Chat
          </span>
          <span className="text-lg font-bold leading-tight">
            {conversations.find((c) => c.id === selectedRoom)?.label || "Chat"}
          </span>
          <span className="text-xs text-text-sub-light dark:text-text-sub-dark">
            {connected ? "Connected" : "Disconnected"} as {activeName}
          </span>
        </div>
      </header>

      {/* Main layout */}
      <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-3 px-4 py-4 md:flex-row">
        {/* Conversation list */}
        <aside className="mb-3 flex w-full flex-none flex-col gap-3 rounded-2xl border border-black/5 bg-surface-light p-3 shadow-sm dark:border-white/10 dark:bg-surface-dark md:mb-0 md:w-80">
          <div className="flex items-center justify-between px-1">
            <span className="text-sm font-semibold">Chats</span>
          </div>
          <div className="no-scrollbar mt-1 flex flex-col gap-1 overflow-y-auto">
            {conversations.map((conv) => (
              <button
                key={conv.id}
                onClick={() => setSelectedRoom(conv.id)}
                className={`flex flex-col items-start rounded-2xl px-3 py-2 text-left transition ${
                  selectedRoom === conv.id
                    ? "bg-primary/20 text-text-main-light dark:text-text-main-dark"
                    : "hover:bg-black/5 dark:hover:bg-white/10 text-text-main-light dark:text-text-main-dark"
                }`}
              >
                <span className="text-sm font-semibold">{conv.label}</span>
                <span className="text-[11px] text-text-sub-light dark:text-text-sub-dark line-clamp-1">
                  {conv.description}
                </span>
              </button>
            ))}
          </div>
        </aside>

        {/* Messages and input */}
        <div className="flex min-h-[360px] flex-1 flex-col overflow-hidden rounded-2xl border border-black/5 bg-surface-light shadow-sm dark:border-white/10 dark:bg-surface-dark">
          {/* Messages list */}
          <div className="flex-1 space-y-2 overflow-y-auto px-4 py-4">
            {visibleMessages.map((msg, idx) => {
              if (msg.type === "friend_request" && msg.target === activeName) {
                return (
                  <div
                    key={idx}
                    className="flex flex-col gap-1 rounded-2xl bg-primary/10 px-3 py-2 text-sm"
                  >
                    <span className="text-xs text-text-sub-light dark:text-text-sub-dark">
                      Friend request from{" "}
                      <span className="font-semibold">{msg.username}</span>
                    </span>
                    <div className="mt-1 flex gap-2">
                      <button
                        onClick={() => handleAcceptFriend(msg.username)}
                        className="rounded-full bg-primary px-3 py-1 text-xs font-bold text-black"
                      >
                        Accept
                      </button>
                      <button
                        onClick={() => handleDeclineFriend(msg.username)}
                        className="rounded-full border border-black/10 px-3 py-1 text-xs font-medium text-text-main-light dark:border-white/10 dark:text-text-main-dark"
                      >
                        Decline
                      </button>
                    </div>
                  </div>
                );
              }

              if (msg.type === "system") {
                return (
                  <div
                    key={idx}
                    className="flex justify-center text-xs text-text-sub-light dark:text-text-sub-dark"
                  >
                    {msg.message}
                  </div>
                );
              }

              if (msg.type === "join" || msg.type === "leave") {
                const verb = msg.type === "join" ? "has joined" : "has left";
                return (
                  <div
                    key={idx}
                    className="flex justify-center text-xs text-text-sub-light dark:text-text-sub-dark"
                  >
                    {msg.username} {verb} ({formatTime(msg.timestamp)})
                  </div>
                );
              }

              return (
                <div key={idx} className="flex flex-col gap-1">
                  <div className="flex items-center gap-2 text-xs text-text-sub-light dark:text-text-sub-dark">
                    <span className="font-semibold text-text-main-light dark:text-text-main-dark">
                      {msg.username}
                    </span>
                    <span>{formatTime(msg.timestamp)}</span>
                  </div>
                  {msg.type === "chat" ? (
                    <div className="inline-block max-w-2xl rounded-2xl bg-primary/10 px-3 py-2 text-sm text-text-main-light dark:text-text-main-dark">
                      {msg.message}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>

          {/* Typing indicator */}
          {typingText && (
            <div className="px-4 pb-2 text-xs text-text-sub-light dark:text-text-sub-dark">
              {typingText}
            </div>
          )}

          {/* Input bar */}
          <form
            onSubmit={handleSubmit}
            className="flex items-center gap-3 border-t border-black/5 px-4 py-3 dark:border-white/10"
          >
            <input
              value={input}
              onChange={(e) => handleTyping(e.target.value)}
              placeholder="Type a message..."
              className="flex-1 rounded-full border border-black/10 bg-background-light px-4 py-2 text-sm outline-none ring-0 transition focus:border-primary dark:border-white/10 dark:bg-background-dark"
            />
            <button
              type="submit"
              className="rounded-full bg-primary px-4 py-2 text-sm font-bold text-black shadow-sm hover:opacity-90"
            >
              Send
            </button>
          </form>
        </div>
      </div>

      <BottomNav active="chat" />
    </div>
  );
}
