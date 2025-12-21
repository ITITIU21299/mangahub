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
  data: MangaDexManga[];
  total?: number;
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

      console.log(`[MangaDex] Response status: ${response.status}`);

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
  // mangaItem is already a single manga object from the data array
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

  return {
    id: manga.id,
    title,
    author,
    genres,
    status,
    total_chapters: attributes.lastChapter
      ? parseInt(attributes.lastChapter, 10) || 0
      : 0,
    description,
    cover_url: coverUrl,
    // Store original MangaDex data for reference
    mangadex_id: manga.id,
  };
}

export async function GET(request: NextRequest) {
  try {
    const searchParams = request.nextUrl.searchParams;
    const title = searchParams.get("title") || "";
    const limit = parseInt(searchParams.get("limit") || "20", 10);
    const offset = parseInt(searchParams.get("offset") || "0", 10);
    const genre = searchParams.get("genre");
    const status = searchParams.get("status");

    const hasGenreFilter = genre && genre !== "All";

    // When filtering by genre, we need to fetch more items and filter client-side
    // Strategy: Fetch batches, calculate filter ratio, estimate total
    if (hasGenreFilter) {
      const allFilteredMangas: ReturnType<typeof transformMangaDexToManga>[] =
        [];
      let currentOffset = 0;
      const batchSize = 100; // Fetch 100 at a time from MangaDex
      const maxFetches = 10; // Safety limit: max 10 batches (1000 items)
      const targetCount = offset + limit; // We need at least this many filtered items
      const sampleSize = 300; // Fetch at least 300 items to get a good filter ratio estimate
      let totalFetched = 0;
      let totalFiltered = 0;
      let mangadexTotal: number | null = null;

      // First, fetch enough to get a good sample for ratio calculation
      while (
        (allFilteredMangas.length < targetCount || totalFetched < sampleSize) &&
        currentOffset < maxFetches * batchSize
      ) {
        // Build MangaDex query for this batch
        const params = new URLSearchParams();
        if (title.trim()) {
          params.append("title", title.trim());
        }
        params.append("limit", batchSize.toString());
        params.append("offset", currentOffset.toString());
        params.append("includes[]", "cover_art");
        params.append("includes[]", "author");
        params.append("includes[]", "artist");

        // Add content rating filter
        params.append("contentRating[]", "safe");
        params.append("contentRating[]", "suggestive");
        params.append("contentRating[]", "erotica");

        // Add status filter if provided
        if (status && status !== "All") {
          const statusMap: Record<string, string> = {
            Ongoing: "ongoing",
            Completed: "completed",
            Hiatus: "hiatus",
          };
          if (statusMap[status]) {
            params.append("status[]", statusMap[status]);
          }
        }

        const url = `${MANGADEX_BASE}/manga?${params.toString()}`;
        const data = (await rateLimitedFetch(url)) as MangaDexResponse;

        // Store total from first response
        if (mangadexTotal === null && data.total !== undefined) {
          mangadexTotal = data.total;
        }

        // Check if we have valid data
        if (
          !data ||
          !data.data ||
          !Array.isArray(data.data) ||
          data.data.length === 0
        ) {
          // No more data available
          break;
        }

        // Transform and filter
        const mangas = data.data
          .map(transformMangaDexToManga)
          .filter((m): m is NonNullable<typeof m> => m !== null);

        totalFetched += mangas.length;

        // Filter by genre
        const filtered = mangas.filter((m) =>
          m.genres.some((g) => g.toLowerCase().includes(genre!.toLowerCase()))
        );

        totalFiltered += filtered.length;
        allFilteredMangas.push(...filtered);

        // If we got fewer items than requested, we might have reached the end
        if (data.data.length < batchSize) {
          break;
        }

        currentOffset += batchSize;
      }

      // Calculate filter ratio and estimate total
      const filterRatio = totalFetched > 0 ? totalFiltered / totalFetched : 0;
      let estimatedTotal: number;

      if (mangadexTotal !== null && filterRatio > 0) {
        // Estimate total based on filter ratio
        estimatedTotal = Math.floor(mangadexTotal * filterRatio);
      } else {
        // Fallback: use what we've found so far
        estimatedTotal = allFilteredMangas.length;
      }

      // Now paginate the filtered results
      const paginatedMangas = allFilteredMangas.slice(offset, offset + limit);
      const totalPages = Math.ceil(estimatedTotal / limit);
      const currentPage = Math.floor(offset / limit) + 1;

      return NextResponse.json({
        data: paginatedMangas,
        pagination: {
          page: currentPage,
          limit,
          total: estimatedTotal,
          total_pages: totalPages,
        },
      });
    }

    // No genre filter - use simple pagination
    const params = new URLSearchParams();
    if (title.trim()) {
      params.append("title", title.trim());
    }
    params.append("limit", Math.min(limit, 100).toString());
    params.append("offset", offset.toString());
    params.append("includes[]", "cover_art");
    params.append("includes[]", "author");
    params.append("includes[]", "artist");

    // Add content rating filter (safe for all audiences)
    params.append("contentRating[]", "safe");
    params.append("contentRating[]", "suggestive");
    params.append("contentRating[]", "erotica");

    // Add status filter if provided
    if (status && status !== "All") {
      const statusMap: Record<string, string> = {
        Ongoing: "ongoing",
        Completed: "completed",
        Hiatus: "hiatus",
      };
      if (statusMap[status]) {
        params.append("status[]", statusMap[status]);
      }
    }

    const url = `${MANGADEX_BASE}/manga?${params.toString()}`;
    const data = (await rateLimitedFetch(url)) as MangaDexResponse;

    // Check if we have valid data
    if (!data || !data.data || !Array.isArray(data.data)) {
      console.error("Invalid MangaDex response:", data);
      throw new Error("Invalid response from MangaDex API");
    }

    // Transform MangaDex response to our format
    const mangas = data.data
      .map(transformMangaDexToManga)
      .filter((m): m is NonNullable<typeof m> => m !== null);

    // Calculate pagination
    const total = data.total || mangas.length;
    const totalPages = Math.ceil(total / limit);

    return NextResponse.json({
      data: mangas,
      pagination: {
        page: Math.floor(offset / limit) + 1,
        limit,
        total,
        total_pages: totalPages,
      },
    });
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
        data: [],
        pagination: {
          page: 1,
          limit: 20,
          total: 0,
          total_pages: 0,
        },
      },
      { status: 500 }
    );
  }
}
