import { NextResponse } from "next/server";

const MANGADEX_BASE = "https://api.mangadex.org";

// Rate limiting: delay between requests (200ms = 5 req/sec)
let lastRequestTime = 0;
const MIN_DELAY_MS = 200;

async function rateLimitedFetch(url: string, retries = 3) {
  const now = Date.now();
  const timeSinceLastRequest = now - lastRequestTime;
  if (timeSinceLastRequest < MIN_DELAY_MS) {
    await new Promise((resolve) =>
      setTimeout(resolve, MIN_DELAY_MS - timeSinceLastRequest)
    );
  }
  lastRequestTime = Date.now();

  for (let attempt = 1; attempt <= retries; attempt++) {
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 10000); // 10 second timeout

      const response = await fetch(url, {
        method: "GET",
        headers: {
          "User-Agent": "MangaHub/1.0 (Net Centric Project)",
          Accept: "application/json",
        },
        signal: controller.signal,
        cache: "no-store",
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        if (response.status === 429) {
          throw new Error("Rate limit exceeded. Please try again later.");
        }
        throw new Error(`MangaDex API error: ${response.statusText}`);
      }

      return response.json();
    } catch (error: unknown) {
      const isLastAttempt = attempt === retries;

      if (error instanceof Error && error.name === "AbortError") {
        if (isLastAttempt) {
          throw new Error("Request timed out. Please try again.");
        }
        await new Promise((resolve) => setTimeout(resolve, 1000 * attempt));
        continue;
      }

      const errorMessage =
        error instanceof Error ? error.message : String(error);
      const errorCode = (error as any)?.code;

      const isNetworkError =
        error instanceof TypeError ||
        errorCode === "ECONNREFUSED" ||
        errorCode === "ENOTFOUND" ||
        errorCode === "ETIMEDOUT" ||
        (error instanceof Error &&
          (errorMessage.includes("fetch failed") ||
            errorMessage.includes("ECONNREFUSED") ||
            errorMessage.includes("network") ||
            errorMessage.includes("ENOTFOUND") ||
            errorMessage.includes("ETIMEDOUT")));

      if (isLastAttempt) {
        if (isNetworkError) {
          throw new Error(
            "Unable to connect to MangaDex API. Please check your internet connection and try again."
          );
        }
        throw error;
      }

      await new Promise((resolve) => setTimeout(resolve, 1000 * attempt));
    }
  }

  throw new Error("Failed to fetch from MangaDex API after multiple attempts");
}

interface MangaDexTag {
  id: string;
  type: string;
  attributes: {
    name: Record<string, string>;
    group: string;
    version: number;
  };
}

interface MangaDexTagsResponse {
  result: string;
  response: string;
  data: MangaDexTag[];
  total: number;
}

export async function GET() {
  try {
    // Fetch all tags from MangaDex
    const url = `${MANGADEX_BASE}/manga/tag`;
    const data = (await rateLimitedFetch(url)) as MangaDexTagsResponse;

    if (!data || !data.data || !Array.isArray(data.data)) {
      throw new Error("Invalid response from MangaDex API");
    }

    // Extract genre tags (group === "genre")
    // Also include popular content tags that users might want to filter by
    const genreTags = data.data
      .filter((tag) => tag.attributes.group === "genre")
      .map((tag) => {
        // Get English name, fallback to first available
        const name =
          tag.attributes.name.en ||
          tag.attributes.name[Object.keys(tag.attributes.name)[0]] ||
          tag.id;
        return {
          id: tag.id,
          name: name,
        };
      })
      .sort((a, b) => a.name.localeCompare(b.name)); // Sort alphabetically

    return NextResponse.json({
      genres: genreTags.map((tag) => tag.name),
    });
  } catch (error: unknown) {
    console.error("MangaDex tags API error:", error);

    let errorMessage = "Failed to fetch genres from MangaDex";
    if (error instanceof Error) {
      if (
        error.message.includes("ECONNREFUSED") ||
        error.message.includes("fetch failed")
      ) {
        errorMessage =
          "Unable to connect to MangaDex API. Please check your internet connection.";
      } else if (
        error.message.includes("timeout") ||
        error.message.includes("aborted")
      ) {
        errorMessage = "Request timed out. Please try again.";
      } else {
        errorMessage = error.message;
      }
    }

    return NextResponse.json(
      {
        error: errorMessage,
        genres: [], // Return empty array on error
      },
      { status: 500 }
    );
  }
}

