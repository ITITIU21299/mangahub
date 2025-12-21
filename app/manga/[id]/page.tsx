"use client";

import { useEffect, useState } from "react";
import { useRouter, useParams } from "next/navigation";

const MANGADEX_API = "/api/mangadex";
const LOCAL_API_BASE =
  process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";

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

interface UserProgress {
  user_id: string;
  manga_id: string;
  current_chapter: number;
  status: string;
  updated_at: string;
}

interface MangaDetailsResponse extends Manga {
  user_progress: UserProgress | null;
}

export default function MangaDetailsPage() {
  const router = useRouter();
  const params = useParams();
  const mangaId = params.id as string;

  const [manga, setManga] = useState<Manga | null>(null);
  const [userProgress, setUserProgress] = useState<UserProgress | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showUpdateModal, setShowUpdateModal] = useState(false);
  const [selectedStatus, setSelectedStatus] = useState("Reading");
  const [currentChapter, setCurrentChapter] = useState(0);
  const [actionLoading, setActionLoading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const statusOptions = ["Reading", "Completed", "Plan to Read", "Dropped"];

  useEffect(() => {
    if (!mangaId) return;
    fetchMangaDetails();
  }, [mangaId]);

  const fetchMangaDetails = async () => {
    setLoading(true);
    setError(null);

    try {
      const token = localStorage.getItem("mangahub_token");

      // Try MangaDex API first
      let res = await fetch(`${MANGADEX_API}/manga/${mangaId}`);

      let mangaData: Manga | null = null;
      let progressData: UserProgress | null = null;

      // If MangaDex fails, fallback to local API
      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}));
        const isNetworkError =
          res.status === 500 &&
          (errorData.error?.includes("Unable to connect") ||
            errorData.error?.includes("ECONNREFUSED") ||
            errorData.error?.includes("fetch failed"));

        if (isNetworkError && token) {
          console.log("[Manga Details] MangaDex unavailable, using local API");
          res = await fetch(`${LOCAL_API_BASE}/manga/${mangaId}`, {
            headers: {
              Authorization: `Bearer ${token}`,
            },
          });

          if (res.ok) {
            const localData: MangaDetailsResponse = await res.json();
            mangaData = localData;
            progressData = localData.user_progress || null;
          } else {
            const localError = await res.json().catch(() => ({}));
            setError(localError.error || "Manga not found");
            return;
          }
        } else {
          setError(errorData.error || "Failed to load manga");
          return;
        }
      } else {
        // MangaDex succeeded
        mangaData = await res.json();

        // Fetch user progress from local API if logged in
        if (token) {
          try {
            const progressRes = await fetch(`${LOCAL_API_BASE}/users/library`, {
              headers: {
                Authorization: `Bearer ${token}`,
              },
            });
            if (progressRes.ok) {
              const library: UserProgress[] | null = await progressRes
                .json()
                .catch(() => null);
              if (library && Array.isArray(library)) {
                const progress = library.find((p) => p.manga_id === mangaId);
                progressData = progress || null;
              }
            }
          } catch (err) {
            console.error("Failed to fetch user progress:", err);
          }
        }
      }

      if (mangaData) {
        setManga(mangaData);
        setUserProgress(progressData);
        if (progressData) {
          setCurrentChapter(progressData.current_chapter);
          setSelectedStatus(progressData.status);
        } else {
          setCurrentChapter(0);
        }
      }
    } catch (err) {
      console.error("Error fetching manga:", err);
      setError("Unable to load manga details. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleAddToLibrary = async () => {
    if (!manga) return;

    setActionLoading(true);
    setActionError(null);
    setSuccessMessage(null);

    try {
      const token = localStorage.getItem("mangahub_token");
      if (!token) {
        router.push("/auth/signin");
        return;
      }

      const res = await fetch(`${LOCAL_API_BASE}/users/library`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          manga_id: manga.id,
          current_chapter: currentChapter,
          status: selectedStatus,
        }),
      });

      const data = await res.json().catch(() => ({}));

      if (!res.ok) {
        setActionError(data.error || "Failed to add to library");
        return;
      }

      setSuccessMessage("Added to library successfully!");
      setShowAddModal(false);
      // Refresh manga details to get updated progress
      setTimeout(() => {
        fetchMangaDetails();
      }, 500);
    } catch (err) {
      setActionError("Unable to add to library. Please try again.");
    } finally {
      setActionLoading(false);
    }
  };

  const handleUpdateProgress = async () => {
    if (!manga) return;

    setActionLoading(true);
    setActionError(null);
    setSuccessMessage(null);

    try {
      const token = localStorage.getItem("mangahub_token");
      if (!token) {
        router.push("/auth/signin");
        return;
      }

      const res = await fetch(`${LOCAL_API_BASE}/users/progress`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          manga_id: manga.id,
          current_chapter: currentChapter,
          status: selectedStatus,
        }),
      });

      const data = await res.json().catch(() => ({}));

      if (!res.ok) {
        setActionError(data.error || "Failed to update progress");
        return;
      }

      setSuccessMessage("Progress updated successfully!");
      setShowUpdateModal(false);
      // Refresh manga details to get updated progress
      setTimeout(() => {
        fetchMangaDetails();
      }, 500);
    } catch (err) {
      setActionError("Unable to update progress. Please try again.");
    } finally {
      setActionLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
        <p className="text-text-sub-light dark:text-text-sub-dark">
          Loading...
        </p>
      </div>
    );
  }

  if (error || !manga) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center bg-background-light px-6 text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
        <span className="material-symbols-outlined mb-4 text-6xl text-text-sub-light dark:text-text-sub-dark">
          error
        </span>
        <p className="text-lg font-semibold">{error || "Manga not found"}</p>
        <button
          onClick={() => router.back()}
          className="mt-4 rounded-full bg-primary px-6 py-2 text-sm font-bold text-black"
        >
          Go Back
        </button>
      </div>
    );
  }

  return (
    <div className="relative flex min-h-screen w-full flex-col overflow-x-hidden pb-24 bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
      {/* Header with Back Button */}
      <header className="sticky top-0 z-20 flex items-center justify-between bg-background-light/90 px-6 py-4 pt-6 backdrop-blur-md dark:bg-background-dark/90">
        <button
          onClick={() => router.back()}
          className="flex h-10 w-10 items-center justify-center rounded-full bg-surface-light text-text-main transition-colors hover:bg-gray-200 dark:bg-surface-dark dark:text-white"
        >
          <span className="material-symbols-outlined">arrow_back</span>
        </button>
        <h2 className="flex-1 text-center text-xl font-bold">Manga Details</h2>
        <div className="w-10" /> {/* Spacer for centering */}
      </header>

      {/* Hero Section */}
      <div className="px-6 pt-4">
        <div className="flex flex-col gap-4 md:flex-row">
          {/* Cover Image */}
          <div className="flex-shrink-0">
            <div className="relative aspect-[3/4] w-full max-w-[200px] overflow-hidden rounded-xl shadow-lg md:w-[200px]">
              {manga.cover_url ? (
                <img
                  src={manga.cover_url}
                  alt={manga.title}
                  className="h-full w-full object-cover"
                  onError={(e) => {
                    (e.target as HTMLImageElement).style.display = "none";
                  }}
                />
              ) : (
                <div className="flex h-full w-full items-center justify-center bg-gray-200 dark:bg-gray-800">
                  <span className="material-symbols-outlined text-4xl text-text-sub-light dark:text-text-sub-dark">
                    image
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Title and Basic Info */}
          <div className="flex flex-1 flex-col gap-3">
            <div>
              <h1 className="text-2xl font-bold leading-tight">
                {manga.title}
              </h1>
              <p className="mt-1 text-base text-text-sub-light dark:text-text-sub-dark">
                {manga.author || "Unknown Author"}
              </p>
            </div>

            {/* Status Badge */}
            {manga.status && (
              <div className="inline-flex w-fit rounded-full bg-primary px-3 py-1 text-xs font-bold uppercase tracking-wide text-black">
                {manga.status}
              </div>
            )}

            {/* Genres */}
            {manga.genres && manga.genres.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {manga.genres.slice(0, 5).map((genre) => (
                  <span
                    key={genre}
                    className="rounded-full bg-surface-light px-3 py-1 text-xs font-medium ring-1 ring-black/5 dark:bg-surface-dark dark:ring-white/10"
                  >
                    {genre}
                  </span>
                ))}
              </div>
            )}

            {/* Chapter Count */}
            <div className="text-sm text-text-sub-light dark:text-text-sub-dark">
              <span className="font-semibold">
                {manga.total_chapters > 0 ? manga.total_chapters : "Unknown"}
              </span>{" "}
              {manga.total_chapters === 1 ? "chapter" : "chapters"}
            </div>

            {/* User Progress Display */}
            {userProgress && (
              <div className="rounded-xl bg-surface-light p-3 ring-1 ring-black/5 dark:bg-surface-dark dark:ring-white/10">
                <p className="text-xs font-medium text-text-sub-light dark:text-text-sub-dark">
                  Your Progress
                </p>
                <p className="mt-1 text-base font-bold">
                  Chapter {userProgress.current_chapter} /{" "}
                  {manga.total_chapters > 0 ? manga.total_chapters : "Unknown"}
                </p>
                <p className="mt-1 text-sm text-text-sub-light dark:text-text-sub-dark">
                  Status: {userProgress.status}
                </p>
              </div>
            )}

            {/* Action Buttons */}
            <div className="mt-2 flex gap-3">
              {!userProgress ? (
                <button
                  onClick={() => setShowAddModal(true)}
                  className="flex-1 rounded-full bg-primary px-4 py-3 text-sm font-bold text-black shadow-sm transition-all hover:shadow-md active:scale-95"
                >
                  Add to Library
                </button>
              ) : (
                <button
                  onClick={() => {
                    setCurrentChapter(userProgress.current_chapter);
                    setSelectedStatus(userProgress.status);
                    setShowUpdateModal(true);
                  }}
                  className="flex-1 rounded-full bg-primary px-4 py-3 text-sm font-bold text-black shadow-sm transition-all hover:shadow-md active:scale-95"
                >
                  Update Progress
                </button>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Description */}
      <div className="mt-6 px-6">
        <h3 className="mb-3 text-lg font-bold">Description</h3>
        <p className="leading-relaxed text-text-main-light dark:text-text-main-dark">
          {manga.description || "No description available."}
        </p>
      </div>

      {/* Add to Library Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-md rounded-2xl bg-surface-light p-6 shadow-xl dark:bg-surface-dark">
            <h3 className="mb-4 text-xl font-bold">Add to Library</h3>

            <div className="mb-4 flex flex-col gap-3">
              <div>
                <label className="mb-1 block text-sm font-medium">Status</label>
                <select
                  value={selectedStatus}
                  onChange={(e) => setSelectedStatus(e.target.value)}
                  className="w-full rounded-lg border-2 border-gray-200 bg-background-light px-4 py-2 dark:border-gray-700 dark:bg-background-dark"
                >
                  {statusOptions.map((status) => (
                    <option key={status} value={status}>
                      {status}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium">
                  Current Chapter
                </label>
                <input
                  type="number"
                  min="0"
                  max={manga.total_chapters || 9999}
                  value={currentChapter}
                  onChange={(e) =>
                    setCurrentChapter(parseInt(e.target.value) || 0)
                  }
                  className="w-full rounded-lg border-2 border-gray-200 bg-background-light px-4 py-2 dark:border-gray-700 dark:bg-background-dark"
                />
              </div>
            </div>

            {actionError && (
              <p className="mb-3 rounded-lg bg-red-100 px-3 py-2 text-sm text-red-800 dark:bg-red-900/40 dark:text-red-200">
                {actionError}
              </p>
            )}

            {successMessage && (
              <p className="mb-3 rounded-lg bg-emerald-100 px-3 py-2 text-sm text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200">
                {successMessage}
              </p>
            )}

            <div className="flex gap-3">
              <button
                onClick={() => {
                  setShowAddModal(false);
                  setActionError(null);
                  setSuccessMessage(null);
                }}
                className="flex-1 rounded-full border-2 border-gray-300 px-4 py-2 text-sm font-medium transition-colors hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800"
              >
                Cancel
              </button>
              <button
                onClick={handleAddToLibrary}
                disabled={actionLoading}
                className="flex-1 rounded-full bg-primary px-4 py-2 text-sm font-bold text-black shadow-sm transition-all disabled:opacity-50"
              >
                {actionLoading ? "Adding..." : "Add"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Update Progress Modal */}
      {showUpdateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-md rounded-2xl bg-surface-light p-6 shadow-xl dark:bg-surface-dark">
            <h3 className="mb-4 text-xl font-bold">Update Progress</h3>

            <div className="mb-4 flex flex-col gap-3">
              <div>
                <label className="mb-1 block text-sm font-medium">Status</label>
                <select
                  value={selectedStatus}
                  onChange={(e) => setSelectedStatus(e.target.value)}
                  className="w-full rounded-lg border-2 border-gray-200 bg-background-light px-4 py-2 dark:border-gray-700 dark:bg-background-dark"
                >
                  {statusOptions.map((status) => (
                    <option key={status} value={status}>
                      {status}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium">
                  Current Chapter
                </label>
                <input
                  type="number"
                  min="0"
                  max={manga.total_chapters || 9999}
                  value={currentChapter}
                  onChange={(e) =>
                    setCurrentChapter(parseInt(e.target.value) || 0)
                  }
                  className="w-full rounded-lg border-2 border-gray-200 bg-background-light px-4 py-2 dark:border-gray-700 dark:bg-background-dark"
                />
              </div>
            </div>

            {actionError && (
              <p className="mb-3 rounded-lg bg-red-100 px-3 py-2 text-sm text-red-800 dark:bg-red-900/40 dark:text-red-200">
                {actionError}
              </p>
            )}

            {successMessage && (
              <p className="mb-3 rounded-lg bg-emerald-100 px-3 py-2 text-sm text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200">
                {successMessage}
              </p>
            )}

            <div className="flex gap-3">
              <button
                onClick={() => {
                  setShowUpdateModal(false);
                  setActionError(null);
                  setSuccessMessage(null);
                }}
                className="flex-1 rounded-full border-2 border-gray-300 px-4 py-2 text-sm font-medium transition-colors hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800"
              >
                Cancel
              </button>
              <button
                onClick={handleUpdateProgress}
                disabled={actionLoading}
                className="flex-1 rounded-full bg-primary px-4 py-2 text-sm font-bold text-black shadow-sm transition-all disabled:opacity-50"
              >
                {actionLoading ? "Updating..." : "Update"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Bottom Navigation */}
      <nav className="pb-safe fixed bottom-0 z-30 w-full border-t border-black/5 bg-surface-light/80 backdrop-blur-lg dark:border-white/5 dark:bg-background-dark/80">
        <div className="flex h-20 items-center justify-around px-2 pb-2">
          <a
            href="/"
            className="flex h-full w-full flex-col items-center justify-center gap-1 text-text-sub-light transition-colors hover:text-text-main-light dark:text-text-sub-dark dark:hover:text-white"
          >
            <span className="material-symbols-outlined text-[26px]">home</span>
            <span className="text-[10px] font-medium">Home</span>
          </a>
          <a
            href="/discover"
            className="flex h-full w-full flex-col items-center justify-center gap-1 text-text-sub-light transition-colors hover:text-text-main-light dark:text-text-sub-dark dark:hover:text-white"
          >
            <span className="material-symbols-outlined text-[26px]">
              explore
            </span>
            <span className="text-[10px] font-medium">Discover</span>
          </a>
          <a
            href="/library"
            className="flex h-full w-full flex-col items-center justify-center gap-1 text-text-sub-light transition-colors hover:text-text-main-light dark:text-text-sub-dark dark:hover:text-white"
          >
            <span className="material-symbols-outlined text-[26px]">
              collections_bookmark
            </span>
            <span className="text-[10px] font-medium">Library</span>
          </a>
          <a
            href="/profile"
            className="flex h-full w-full flex-col items-center justify-center gap-1 text-text-sub-light transition-colors hover:text-text-main-light dark:text-text-sub-dark dark:hover:text-white"
          >
            <span className="material-symbols-outlined text-[26px]">
              person
            </span>
            <span className="text-[10px] font-medium">Profile</span>
          </a>
        </div>
      </nav>
    </div>
  );
}
