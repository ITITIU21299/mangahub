"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

const LOCAL_API_BASE =
  process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";
const MANGADEX_API = "/api/mangadex";

interface UserProgress {
  user_id: string;
  manga_id: string;
  current_chapter: number;
  status: string;
  updated_at: string;
}

interface Manga {
  id: string;
  title: string;
  author: string;
  genres: string[];
  status: string;
  total_chapters: number;
  description: string;
  cover_url: string;
  mangadex_id?: string;
}

interface LibraryItem extends UserProgress {
  manga?: Manga;
}

export default function LibraryPage() {
  const router = useRouter();
  const [library, setLibrary] = useState<LibraryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedStatus, setSelectedStatus] = useState<string>("All");
  const [page, setPage] = useState<number>(1);

  const PAGE_SIZE = 20;

  const statuses = ["All", "Reading", "Completed", "Plan to Read", "Dropped"];

  useEffect(() => {
    fetchLibrary();
  }, []);

  const fetchLibrary = async () => {
    setLoading(true);
    setError(null);

    try {
      const token = localStorage.getItem("mangahub_token");
      if (!token) {
        router.push("/auth/signin");
        return;
      }

      // Fetch user's library from backend
      const res = await fetch(`${LOCAL_API_BASE}/users/library`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!res.ok) {
        throw new Error("Failed to fetch library");
      }

      const userProgress: UserProgress[] = await res.json();

      // Fetch manga details for each item
      const libraryItems: LibraryItem[] = await Promise.all(
        userProgress.map(async (progress) => {
          let manga: Manga | undefined;

          try {
            // Try to fetch from MangaDex first
            const mangaRes = await fetch(
              `${MANGADEX_API}/manga/${progress.manga_id}`
            );
            if (mangaRes.ok) {
              manga = await mangaRes.json();
            } else {
              // Fallback to local API
              const localRes = await fetch(
                `${LOCAL_API_BASE}/manga/${progress.manga_id}`,
                {
                  headers: {
                    Authorization: `Bearer ${token}`,
                  },
                }
              );
              if (localRes.ok) {
                const localData = await localRes.json();
                manga = {
                  id: localData.id,
                  title: localData.title,
                  author: localData.author,
                  genres: localData.genres || [],
                  status: localData.status,
                  total_chapters: localData.total_chapters || 0,
                  description: localData.description || "",
                  cover_url: localData.cover_url || "",
                };
              }
            }
          } catch (err) {
            console.warn(`Failed to fetch manga ${progress.manga_id}:`, err);
          }

          return {
            ...progress,
            manga,
          };
        })
      );

      setLibrary(libraryItems);
    } catch (err) {
      console.error("Error fetching library:", err);
      setError("Failed to load library. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleStatusFilter = (status: string) => {
    setSelectedStatus(status);
    setPage(1); // reset to first page when filter changes
  };

  const handleMangaClick = (mangaId: string) => {
    router.push(`/manga/${mangaId}`);
  };

  // Filter library by status
  const filteredLibrary =
    selectedStatus === "All"
      ? library
      : library.filter((item) => item.status === selectedStatus);

  // Pagination (client-side) similar to discover page
  const totalPages = Math.max(1, Math.ceil(filteredLibrary.length / PAGE_SIZE));
  const currentPage = Math.min(page, totalPages);
  const paginatedItems = filteredLibrary.slice(
    (currentPage - 1) * PAGE_SIZE,
    currentPage * PAGE_SIZE
  );

  // Calculate progress percentage
  const getProgressPercentage = (item: LibraryItem): number => {
    if (!item.manga || item.manga.total_chapters === 0) return 0;
    return Math.min(
      (item.current_chapter / item.manga.total_chapters) * 100,
      100
    );
  };

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
        <p className="text-text-sub-light dark:text-text-sub-dark">
          Loading library...
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center bg-background-light px-6 text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
        <span className="material-symbols-outlined mb-4 text-6xl text-text-sub-light dark:text-text-sub-dark">
          error
        </span>
        <p className="text-lg font-semibold">{error}</p>
        <button
          onClick={fetchLibrary}
          className="mt-4 rounded-full bg-primary px-6 py-2 text-sm font-bold text-black"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="relative flex min-h-screen w-full flex-col overflow-x-hidden pb-24 bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
      {/* Header */}
      <header className="sticky top-0 z-20 bg-background-light/95 px-4 pb-2 pt-4 backdrop-blur-md dark:bg-background-dark/95">
        <div className="flex items-center justify-between">
          <h2 className="flex-1 text-3xl font-bold leading-tight tracking-[-0.015em]">
            My Library
          </h2>
          <div className="flex w-12 items-center justify-end">
            <button
              onClick={() => router.push("/profile")}
              className="flex h-10 w-10 items-center justify-center rounded-full bg-surface-light text-text-main-light shadow-sm transition-colors hover:bg-gray-100 dark:bg-surface-dark dark:text-text-main-dark dark:hover:bg-gray-800"
            >
              <span className="material-symbols-outlined">person</span>
            </button>
          </div>
        </div>

        {/* Filter chips */}
        <div className="no-scrollbar mt-3 flex w-full gap-3 overflow-x-auto border-b border-gray-100 pb-3 dark:border-gray-800/50">
          {statuses.map((status) => (
            <button
              key={status}
              onClick={() => handleStatusFilter(status)}
              className={`flex h-9 shrink-0 items-center justify-center gap-x-2 rounded-full px-5 shadow-sm active:scale-95 ${
                selectedStatus === status
                  ? "bg-primary"
                  : "border border-gray-100 bg-surface-light dark:border-gray-800 dark:bg-surface-dark"
              }`}
            >
              <p
                className={`text-sm font-medium leading-normal ${
                  selectedStatus === status ? "font-bold text-black" : ""
                }`}
              >
                {status}
              </p>
            </button>
          ))}
        </div>
      </header>

      {filteredLibrary.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center px-6 py-12">
          <span className="material-symbols-outlined mb-4 text-6xl text-text-sub-light dark:text-text-sub-dark">
            {selectedStatus === "All" ? "library_books" : "filter_alt"}
          </span>
          <p className="text-center text-lg font-semibold">
            {selectedStatus === "All"
              ? "Your library is empty"
              : `No manga with status "${selectedStatus}"`}
          </p>
          <p className="mt-2 text-center text-sm text-text-sub-light dark:text-text-sub-dark">
            {selectedStatus === "All"
              ? "Add manga to your library from the discover page"
              : "Try selecting a different status filter"}
          </p>
          <button
            onClick={() => router.push("/discover")}
            className="mt-6 rounded-full bg-primary px-6 py-2 text-sm font-bold text-black"
          >
            Discover Manga
          </button>
        </div>
      ) : (
        <>
          {/* Vertical grid layout similar to Discover page */}
          <section className="px-4 py-4">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-[22px] font-bold leading-tight tracking-[-0.015em]">
                {selectedStatus === "All" ? "All Manga" : selectedStatus}
              </h2>
              <p className="text-sm text-text-sub-light dark:text-text-sub-dark">
                Page {currentPage} of {totalPages} • {filteredLibrary.length}{" "}
                item
                {filteredLibrary.length === 1 ? "" : "s"}
              </p>
            </div>

            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 pb-4">
              {paginatedItems.map((item) => (
                <div
                  key={item.manga_id}
                  onClick={() => handleMangaClick(item.manga_id)}
                  className="group flex cursor-pointer flex-col gap-2"
                >
                  <div className="relative aspect-[3/4] w-full overflow-hidden rounded-lg shadow-sm transition-all duration-300 group-hover:-translate-y-1 group-hover:shadow-lg bg-gray-200 dark:bg-gray-800">
                    {item.manga?.cover_url ? (
                      <img
                        src={item.manga.cover_url}
                        alt={item.manga.title}
                        className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
                        onError={(e) => {
                          (e.target as HTMLImageElement).style.display = "none";
                        }}
                      />
                    ) : (
                      <div className="flex h-full w-full items-center justify-center">
                        <span className="material-symbols-outlined text-2xl text-text-sub-light dark:text-text-sub-dark">
                          image
                        </span>
                      </div>
                    )}
                    <div className="absolute left-1.5 top-1.5 rounded-full bg-primary px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wide text-black">
                      {item.status}
                    </div>
                    {item.manga && item.manga.total_chapters > 0 && (
                      <div className="absolute bottom-1.5 left-1.5 right-1.5 h-1.5 overflow-hidden rounded-full bg-black/30">
                        <div
                          className="h-full rounded-full bg-primary"
                          style={{ width: `${getProgressPercentage(item)}%` }}
                        />
                      </div>
                    )}
                  </div>
                  <div className="px-0.5">
                    <p className="line-clamp-2 text-sm font-bold leading-tight">
                      {item.manga?.title || "Unknown Manga"}
                    </p>
                    <p className="mt-0.5 line-clamp-1 text-[11px] font-medium leading-normal text-text-sub-light dark:text-text-sub-dark">
                      {item.manga?.author || "Unknown Author"}
                    </p>
                    <p className="mt-0.5 text-[11px] text-text-sub-light dark:text-text-sub-dark">
                      Ch. {item.current_chapter}
                      {item.manga?.total_chapters
                        ? ` / ${item.manga.total_chapters}`
                        : ""}
                    </p>
                  </div>
                </div>
              ))}
            </div>

            {/* Pagination controls similar to Discover */}
            {totalPages > 1 && (
              <div className="mt-4 flex items-center justify-center gap-4">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                  className="flex h-10 w-10 items-center justify-center rounded-full bg-surface-light text-text-main-light shadow-sm disabled:opacity-50 dark:bg-surface-dark dark:text-text-main-dark"
                >
                  <span className="material-symbols-outlined">
                    arrow_back_ios
                  </span>
                </button>
                <span className="text-sm font-medium">
                  Page {currentPage} of {totalPages}
                </span>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                  className="flex h-10 w-10 items-center justify-center rounded-full bg-surface-light text-text-main-light shadow-sm disabled:opacity-50 dark:bg-surface-dark dark:text-text-main-dark"
                >
                  <span className="material-symbols-outlined">
                    arrow_forward_ios
                  </span>
                </button>
              </div>
            )}
          </section>
        </>
      )}

      {/* Floating Action Button - Navigate to Discover */}
      <button
        onClick={() => router.push("/discover")}
        className="fixed bottom-24 right-5 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-primary text-black shadow-[0_4px_12px_rgba(249,245,6,0.5)] transition-all hover:scale-105 active:scale-95"
      >
        <span className="material-symbols-outlined text-[32px]">add</span>
      </button>

      {/* Bottom Navigation */}
      <nav className="pb-safe fixed bottom-0 left-0 z-50 w-full border-t border-gray-100 bg-surface-light/95 backdrop-blur-lg pt-2 dark:border-gray-800 dark:bg-surface-dark/95">
        <div className="flex h-16 items-center justify-around px-2">
          <a
            href="/"
            className="flex flex-1 flex-col items-center gap-1 p-2 text-text-sub-light transition-colors hover:text-primary dark:text-text-sub-dark"
          >
            <span className="material-symbols-outlined">home</span>
            <span className="text-[10px] font-medium">Home</span>
          </a>
          <div className="flex flex-1 flex-col items-center gap-1 p-2 text-text-main-light dark:text-text-main-dark">
            <div className="flex flex-col items-center rounded-full bg-primary/20 px-4 py-0.5 dark:bg-primary/10">
              <span className="material-symbols-outlined text-black dark:text-primary">
                library_books
              </span>
            </div>
            <span className="text-[10px] font-bold text-black dark:text-primary">
              Library
            </span>
          </div>
          <a
            href="/discover"
            className="flex flex-1 flex-col items-center gap-1 p-2 text-text-sub-light transition-colors hover:text-primary dark:text-text-sub-dark"
          >
            <span className="material-symbols-outlined">search</span>
            <span className="text-[10px] font-medium">Search</span>
          </a>
          <a
            href="/profile"
            className="flex flex-1 flex-col items-center gap-1 p-2 text-text-sub-light transition-colors hover:text-primary dark:text-text-sub-dark"
          >
            <span className="material-symbols-outlined">person</span>
            <span className="text-[10px] font-medium">Profile</span>
          </a>
        </div>
      </nav>
    </div>
  );
}
