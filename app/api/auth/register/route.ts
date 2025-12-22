import { NextRequest, NextResponse } from "next/server";

const API_BASE = process.env.MANGAHUB_API_BASE || "http://127.0.0.1:8080";

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const res = await fetch(`${API_BASE}/auth/register`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    });

    const data = await res.json().catch(() => ({}));

    return NextResponse.json(data, {
      status: res.status,
    });
  } catch (err) {
    console.error("Proxy /api/auth/register error:", err);
    return NextResponse.json(
      { error: "Unable to reach API server. Please try again." },
      { status: 502 }
    );
  }
}
