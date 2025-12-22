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
  const [canRetry, setCanRetry] = useState(false);
  const [notificationsEnabled, setNotificationsEnabled] = useState(false);
  const [notificationsLoading, setNotificationsLoading] = useState(false);

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
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 5000); // 5 second timeout

            const progressRes = await fetch(`${LOCAL_API_BASE}/users/library`, {
              headers: {
                Authorization: `Bearer ${token}`,
              },
              signal: controller.signal,
            });

            clearTimeout(timeoutId);

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
            // Silently fail if API server is not available
            // This is expected if Go API server is not running
            // Only log if it's not a network/abort error
            if (err instanceof Error) {
              const isNetworkError =
                err.name === "AbortError" ||
                err.message.includes("Failed to fetch") ||
                err.message.includes("NetworkError") ||
                err.message.includes("ECONNREFUSED");

              if (!isNetworkError) {
                console.warn("Failed to fetch user progress:", err.message);
              }
            }
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

        // Fetch notification subscription status (UC-009)
        if (token) {
          try {
            setNotificationsLoading(true);
            const res = await fetch(
              `${LOCAL_API_BASE}/users/notifications/${mangaId}`,
              {
                headers: {
                  Authorization: `Bearer ${token}`,
                },
              }
            );
            if (res.ok) {
              const data = await res.json().catch(() => null);
              if (data && typeof data.subscribed === "boolean") {
                setNotificationsEnabled(data.subscribed);
              }
            }
          } catch (err) {
            // ignore notification errors in UI for now
          } finally {
            setNotificationsLoading(false);
          }
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
    setCanRetry(false);

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
        // UC-005 Alternative Flow A2: Database error - show retry option
        const isRetryable =
          data.retry === true || data.type === "database_error";
        setActionError(data.error || "Failed to add to library");
        setCanRetry(isRetryable);
        return;
      }

      // UC-005 Main Success Scenario: Confirm addition
      // UC-005 Alternative Flow A1: Handle already_exists status
      if (data.status === "already_exists") {
        setSuccessMessage(
          data.message ||
            "Manga already in library. Progress updated successfully!"
        );
      } else {
        setSuccessMessage(data.message || "Added to library successfully!");
      }

      // Close modal and refresh UI
      setTimeout(() => {
        setShowAddModal(false);
        setActionError(null);
        setSuccessMessage(null);
        fetchMangaDetails(); // Update UI with new progress
      }, 1500);
    } catch (err) {
      // UC-005 Alternative Flow A2: Network error - show retry option
      setActionError(
        "Network error. Please check your connection and try again."
      );
      setCanRetry(true);
    } finally {
      setActionLoading(false);
    }
  };

  const handleUpdateProgress = async () => {
    if (!manga) return;

    setActionLoading(true);
    setActionError(null);
    setSuccessMessage(null);
    setCanRetry(false);

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
        // UC-006 Alternative Flow A1: Invalid chapter number - show validation error
        const errorType = data.type || "unknown_error";
        const isValidationError = errorType === "validation_error";
        const isRetryable =
          data.retry === true || errorType === "database_error";

        setActionError(data.error || "Failed to update progress");
        setCanRetry(isRetryable && !isValidationError); // Don't retry validation errors
        return;
      }

      // UC-006 Main Success Scenario Step 5: Confirm update to user
      let successMsg = data.message || "Progress updated successfully!";

      // UC-006 Alternative Flow A2: TCP server unavailable - inform user but confirm local update
      if (data.warning) {
        successMsg += ` (${data.warning})`;
      }

      setSuccessMessage(successMsg);
      setTimeout(() => {
        setShowUpdateModal(false);
        setActionError(null);
        setSuccessMessage(null);
        setCanRetry(false);
        // Refresh manga details to get updated progress
        fetchMangaDetails();
      }, 1500);
    } catch (err) {
      setActionError(
        "Network error: Unable to update progress. Please check your connection and try again."
      );
      setCanRetry(true);
    } finally {
      setActionLoading(false);
    }
  };

  const handleToggleNotifications = async () => {
    if (!manga) return;

    setNotificationsLoading(true);
    try {
      const token = localStorage.getItem("mangahub_token");
      if (!token) {
        router.push("/auth/signin");
        return;
      }

      let res: Response;

      if (!notificationsEnabled) {
        // Subscribe
        res = await fetch(`${LOCAL_API_BASE}/users/notifications`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({ manga_id: manga.id }),
        });
      } else {
        // Unsubscribe
        res = await fetch(`${LOCAL_API_BASE}/users/notifications/${manga.id}`, {
          method: "DELETE",
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });
      }

      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        console.error(
          "Failed to toggle notifications:",
          data.error || res.statusText
        );
        return;
      }

      setNotificationsEnabled(!notificationsEnabled);
    } catch (err) {
      console.error("Error subscribing to notifications:", err);
    } finally {
      setNotificationsLoading(false);
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
            <div className="mt-2 flex flex-col gap-3 sm:flex-row">
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
              <button
                onClick={handleToggleNotifications}
                disabled={notificationsLoading}
                className="flex-1 rounded-full bg-surface-light px-4 py-3 text-sm font-bold text-text-main-light shadow-sm ring-1 ring-black/10 transition-all hover:bg-black/5 active:scale-95 dark:bg-surface-dark dark:text-text-main-dark dark:ring-white/10 dark:hover:bg-white/5 disabled:opacity-60"
              >
                {notificationsEnabled
                  ? notificationsLoading
                    ? "Disabling..."
                    : "Disable notifications"
                  : notificationsLoading
                  ? "Enabling..."
                  : "Notify me on new chapters"}
              </button>
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
                <select
                  value={currentChapter}
                  onChange={(e) =>
                    setCurrentChapter(parseInt(e.target.value) || 0)
                  }
                  className="w-full rounded-lg border-2 border-gray-200 bg-background-light px-4 py-2 dark:border-gray-700 dark:bg-background-dark"
                >
                  {(() => {
                    const totalChapters = manga.total_chapters || 0;
                    const maxChapters =
                      totalChapters > 0 ? totalChapters : 1000;

                    // Generate chapter options from 0 to latest chapter
                    return Array.from(
                      { length: Math.min(maxChapters + 1, 1001) },
                      (_, i) => i
                    ).map((chapter) => (
                      <option key={chapter} value={chapter}>
                        Chapter {chapter}
                        {chapter === 0 ? " (Not Started)" : ""}
                        {chapter === totalChapters && totalChapters > 0
                          ? " (Latest)"
                          : ""}
                      </option>
                    ));
                  })()}
                </select>
              </div>
            </div>

            {actionError && (
              <div className="mb-3 rounded-lg bg-red-100 px-3 py-2 text-sm text-red-800 dark:bg-red-900/40 dark:text-red-200">
                <p>{actionError}</p>
              </div>
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
                  setCanRetry(false);
                }}
                className="flex-1 rounded-full border-2 border-gray-300 px-4 py-2 text-sm font-medium transition-colors hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800"
                disabled={actionLoading}
              >
                {successMessage ? "Close" : "Cancel"}
              </button>
              {/* UC-005 Alternative Flow A2: Retry button for database errors */}
              {canRetry && actionError ? (
                <button
                  onClick={handleAddToLibrary}
                  disabled={actionLoading}
                  className="flex-1 rounded-full bg-orange-500 px-4 py-2 text-sm font-bold text-white shadow-sm transition-all hover:bg-orange-600 disabled:opacity-50"
                >
                  {actionLoading ? "Retrying..." : "Retry"}
                </button>
              ) : (
                <button
                  onClick={handleAddToLibrary}
                  disabled={actionLoading || !!successMessage}
                  className="flex-1 rounded-full bg-primary px-4 py-2 text-sm font-bold text-black shadow-sm transition-all disabled:opacity-50"
                >
                  {actionLoading ? "Adding..." : "Add"}
                </button>
              )}
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
                <select
                  value={currentChapter}
                  onChange={(e) =>
                    setCurrentChapter(parseInt(e.target.value) || 0)
                  }
                  className="w-full rounded-lg border-2 border-gray-200 bg-background-light px-4 py-2 dark:border-gray-700 dark:bg-background-dark"
                >
                  {(() => {
                    const totalChapters = manga.total_chapters || 0;
                    const maxChapters =
                      totalChapters > 0 ? totalChapters : 1000;

                    // Generate chapter options from 0 to latest chapter
                    return Array.from(
                      { length: Math.min(maxChapters + 1, 1001) },
                      (_, i) => i
                    ).map((chapter) => (
                      <option key={chapter} value={chapter}>
                        Chapter {chapter}
                        {chapter === 0 ? " (Not Started)" : ""}
                        {chapter === totalChapters && totalChapters > 0
                          ? " (Latest)"
                          : ""}
                      </option>
                    ));
                  })()}
                </select>
              </div>
            </div>

            {actionError && (
              <div className="mb-3 rounded-lg bg-red-100 px-3 py-2 text-sm text-red-800 dark:bg-red-900/40 dark:text-red-200">
                <p>{actionError}</p>
              </div>
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
                  setCanRetry(false);
                }}
                className="flex-1 rounded-full border-2 border-gray-300 px-4 py-2 text-sm font-medium transition-colors hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800"
                disabled={actionLoading}
              >
                {successMessage ? "Close" : "Cancel"}
              </button>
              {canRetry && actionError ? (
                <button
                  onClick={handleUpdateProgress}
                  disabled={actionLoading}
                  className="flex-1 rounded-full bg-orange-500 px-4 py-2 text-sm font-bold text-white shadow-sm transition-all hover:bg-orange-600 disabled:opacity-50"
                >
                  {actionLoading ? "Retrying..." : "Retry"}
                </button>
              ) : (
                <button
                  onClick={handleUpdateProgress}
                  disabled={actionLoading || !!successMessage}
                  className="flex-1 rounded-full bg-primary px-4 py-2 text-sm font-bold text-black shadow-sm transition-all disabled:opacity-50"
                >
                  {actionLoading ? "Updating..." : "Update"}
                </button>
              )}
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
