import { NextRequest, NextResponse } from "next/server";

const MANGADEX_BASE = "https://api.mangadex.org";

// Type definitions for MangaDex API responses
interface MangaDexManga {
  id: string;
  type: string;
  attributes: {
    title?: Record<string, string>;
    description?: Record<string, string>;
    status?: string;
    lastChapter?: string;
    tags?: Array<{
      attributes?: {
        name?: Record<string, string>;
      };
    }>;
  };
  relationships?: Array<{
    type: string;
    attributes?: {
      fileName?: string;
      name?: string;
    };
  }>;
}

interface MangaDexResponse {
  result: string;
  response: string;
  data: MangaDexManga;
}

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

      // Log the URL being fetched for debugging
      console.log(`[MangaDex] Fetching: ${url}`);

      const response = await fetch(url, {
        method: "GET",
        headers: {
          "User-Agent": "MangaHub/1.0 (Net Centric Project)",
          Accept: "application/json",
        },
        signal: controller.signal,
        // Next.js fetch configuration
        cache: "no-store",
      });

      clearTimeout(timeoutId);

      console.log(`[MangaDex] Response status: ${response.status}`);

      if (!response.ok) {
        if (response.status === 429) {
          throw new Error("Rate limit exceeded. Please try again later.");
        }
        throw new Error(`MangaDex API error: ${response.statusText}`);
      }

      return response.json();
    } catch (error: unknown) {
      const isLastAttempt = attempt === retries;

      // Log error details for debugging
      console.error(`[MangaDex] Attempt ${attempt}/${retries} failed:`, error);

      // Check for abort signal (timeout)
      if (error instanceof Error && error.name === "AbortError") {
        if (isLastAttempt) {
          throw new Error("Request timed out. Please try again.");
        }
        // Retry on timeout
        await new Promise((resolve) => setTimeout(resolve, 1000 * attempt));
        continue;
      }

      // Check for network errors (including ECONNREFUSED)
      const errorMessage =
        error instanceof Error ? error.message : String(error);
      const errorCode =
        typeof (error as { code?: unknown }).code === "string"
          ? (error as { code?: unknown }).code
          : undefined;

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

      // Wait before retrying (exponential backoff)
      await new Promise((resolve) => setTimeout(resolve, 1000 * attempt));
    }
  }

  throw new Error("Failed to fetch from MangaDex API after multiple attempts");
}

function transformMangaDexToManga(mangaItem: MangaDexManga): {
  id: string;
  title: string;
  author: string;
  genres: string[];
  status: string;
  total_chapters: number;
  description: string;
  cover_url: string;
  mangadex_id: string;
} | null {
  if (!mangaItem || !mangaItem.attributes) {
    console.warn("Invalid manga item:", mangaItem);
    return null;
  }

  const manga = mangaItem;
  const attributes = manga.attributes || {};

  // Get title (prefer English, fallback to first available)
  const title =
    attributes.title?.en ||
    attributes.title?.[Object.keys(attributes.title || {})[0]] ||
    "Untitled";

  // Get description
  const description =
    attributes.description?.en ||
    attributes.description?.[Object.keys(attributes.description || {})[0]] ||
    "";

  // Get cover image from relationships
  let coverUrl = "";
  const coverArt = manga.relationships?.find((rel) => rel.type === "cover_art");
  if (coverArt?.attributes?.fileName) {
    const mangaId = manga.id;
    const fileName = coverArt.attributes.fileName;
    coverUrl = `https://uploads.mangadex.org/covers/${mangaId}/${fileName}.256.jpg`;
  }

  // Get author from relationships
  let author = "Unknown Author";
  const authorRel = manga.relationships?.find((rel) => rel.type === "author");
  if (authorRel?.attributes?.name) {
    author = authorRel.attributes.name;
  }

  // Get tags/genres
  const genres: string[] = [];
  if (attributes.tags && Array.isArray(attributes.tags)) {
    attributes.tags.forEach((tag) => {
      const tagName =
        tag.attributes?.name?.en ||
        tag.attributes?.name?.[Object.keys(tag.attributes?.name || {})[0]];
      if (tagName) {
        genres.push(tagName);
      }
    });
  }

  // Map status
  let status = "Unknown";
  if (attributes.status) {
    const statusMap: Record<string, string> = {
      ongoing: "Ongoing",
      completed: "Completed",
      hiatus: "Hiatus",
      cancelled: "Cancelled",
    };
    status = statusMap[attributes.status] || attributes.status;
  }

  // Note: MangaDex doesn't provide lastChapter in manga attributes
  // We'll fetch it separately in the GET handler
  return {
    id: manga.id,
    title,
    author,
    genres,
    status,
    total_chapters: 0, // Will be set by fetching chapters separately
    description,
    cover_url: coverUrl,
    mangadex_id: manga.id,
  };
}

async function fetchChapterCount(mangaId: string): Promise<number> {
  try {
    // First, try to get total count from chapters endpoint (more accurate)
    const chaptersUrl = `${MANGADEX_BASE}/chapter?manga=${mangaId}&limit=1&translatedLanguage[]=en`;
    const chaptersData = (await rateLimitedFetch(chaptersUrl)) as {
      result: string;
      response: string;
      total?: number;
    };

    // If we have a total count, use it
    if (chaptersData?.total !== undefined && chaptersData.total > 0) {
      return chaptersData.total;
    }

    // Fallback: try to get the highest chapter number
    const chaptersOrderedUrl = `${MANGADEX_BASE}/chapter?manga=${mangaId}&limit=1&order[chapter]=desc&translatedLanguage[]=en`;
    const chaptersOrderedData = (await rateLimitedFetch(
      chaptersOrderedUrl
    )) as {
      result: string;
      response: string;
      data?: Array<{
        attributes?: {
          chapter?: string;
        };
      }>;
      total?: number;
    };

    if (chaptersOrderedData?.data && chaptersOrderedData.data.length > 0) {
      const chapterNum = chaptersOrderedData.data[0]?.attributes?.chapter;
      if (chapterNum && !isNaN(parseFloat(chapterNum))) {
        return Math.floor(parseFloat(chapterNum));
      }
    }

    // If we still have a total from the ordered query, use it
    if (
      chaptersOrderedData?.total !== undefined &&
      chaptersOrderedData.total > 0
    ) {
      return chaptersOrderedData.total;
    }

    return 0;
  } catch (error) {
    console.warn(`Failed to fetch chapter count for manga ${mangaId}:`, error);
    return 0;
  }
}

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const { id: mangaId } = await params;

    if (!mangaId) {
      return NextResponse.json(
        { error: "Manga ID is required" },
        { status: 400 }
      );
    }

    // Build MangaDex API URL
    const queryParams = new URLSearchParams();
    queryParams.append("includes[]", "cover_art");
    queryParams.append("includes[]", "author");
    queryParams.append("includes[]", "artist");

    const url = `${MANGADEX_BASE}/manga/${mangaId}?${queryParams.toString()}`;
    const data = (await rateLimitedFetch(url)) as MangaDexResponse;

    // Check if we have valid data
    if (!data || !data.data) {
      console.error("Invalid MangaDex response:", data);
      throw new Error("Invalid response from MangaDex API");
    }

    // Transform MangaDex response to our format
    const manga = transformMangaDexToManga(data.data);
    if (!manga) {
      return NextResponse.json(
        { error: "Failed to transform manga data" },
        { status: 500 }
      );
    }

    // Fetch chapter count separately
    const chapterCount = await fetchChapterCount(mangaId);
    manga.total_chapters = chapterCount;

    return NextResponse.json(manga);
  } catch (error: unknown) {
    console.error("MangaDex API error:", error);

    // Provide more helpful error messages
    let errorMessage = "Failed to fetch manga from MangaDex";
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
      },
      { status: 500 }
    );
  }
}
