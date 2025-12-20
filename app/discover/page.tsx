"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";

interface Manga {
  id: string;
  title: string;
  author: string;
  genres: string[];
  status: string;
  total_chapters: number;
  description: string;
  cover_url: string;
}

interface Pagination {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

interface SearchResponse {
  data: Manga[];
  pagination: Pagination;
}

export default function DiscoverPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedGenre, setSelectedGenre] = useState<string | null>(null);
  const [selectedStatus, setSelectedStatus] = useState<string | null>(null);
  const [mangas, setMangas] = useState<Manga[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pagination, setPagination] = useState<Pagination>({
    page: 1,
    limit: 20,
    total: 0,
    total_pages: 0,
  });

  const genres = [
    "All",
    "Action",
    "Romance",
    "Fantasy",
    "Comedy",
    "Drama",
    "Horror",
    "Sci-Fi",
  ];
  const statuses = ["All", "Ongoing", "Completed", "Hiatus"];

  const fetchManga = async (page: number = 1) => {
    setLoading(true);
    setError(null);

    try {
      const token = localStorage.getItem("mangahub_token");
      if (!token) {
        router.push("/auth/signin");
        return;
      }

      const params = new URLSearchParams();
      if (searchQuery.trim()) {
        params.append("q", searchQuery.trim());
      }
      if (selectedGenre && selectedGenre !== "All") {
        params.append("genre", selectedGenre);
      }
      if (selectedStatus && selectedStatus !== "All") {
        params.append("status", selectedStatus);
      }
      params.append("page", page.toString());
      params.append("limit", "20");

      const res = await fetch(`${API_BASE}/manga?${params.toString()}`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (res.status === 401) {
        localStorage.removeItem("mangahub_token");
        router.push("/auth/signin");
        return;
      }

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.error || "Failed to load manga");
        setMangas([]);
        return;
      }

      const data: SearchResponse = await res.json();
      setMangas(data.data || []);
      setPagination(data.pagination || pagination);
    } catch (err) {
      setError("Unable to reach server. Please try again.");
      setMangas([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchManga(1);
  }, [selectedGenre, selectedStatus]);

  const handleSearch = (e: FormEvent) => {
    e.preventDefault();
    fetchManga(1);
  };

  const handleGenreClick = (genre: string) => {
    if (genre === "All") {
      setSelectedGenre(null);
    } else {
      setSelectedGenre(genre);
    }
  };

  const handleStatusClick = (status: string) => {
    if (status === "All") {
      setSelectedStatus(null);
    } else {
      // Toggle: if clicking the same status, unselect it; otherwise select the new one
      if (selectedStatus === status) {
        setSelectedStatus(null);
      } else {
        setSelectedStatus(status);
      }
    }
  };

  const handleMangaClick = (mangaId: string) => {
    // Navigate to manga details page (to be implemented later)
    router.push(`/manga/${mangaId}`);
  };

  return (
    <div className="relative flex min-h-screen w-full flex-col overflow-x-hidden pb-24 bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark font-sans">
      {/* Top App Bar */}
      <header className="sticky top-0 z-20 flex items-center justify-between bg-background-light/90 px-6 py-4 pt-6 backdrop-blur-md dark:bg-background-dark/90">
        <h2 className="flex-1 text-3xl font-bold leading-tight tracking-tight">
          Discover
        </h2>
        <button className="relative flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-surface-light shadow-sm ring-1 ring-black/5 dark:bg-surface-dark dark:ring-white/10">
          <div
            className="h-full w-full bg-cover bg-center"
            style={{
              backgroundImage:
                "url('https://lh3.googleusercontent.com/aida-public/AB6AXuDyvF4nlb0yARpGLfYqMnx4Tn2ke4BKDGXKAHd2JHbCh8aTERDO5a82iS0653MoTrriyrPjnymxZa_ll9ZBL07diWnDFalt7o1ZE8dm-qTSJ6wWnJB89LBv8zaGkdqY8OoeoxmL_YEsnS8w5BGNo4WOWTBDQbgbcAidNtOFI5T7rj_1H1P2T9VywB9NLnkYVG3XdUrnsbfVmRncZWs-Z35KEY3onJeTvclYVN5pxGaTPfTukX8pThu71cL55sQdvzRWnKEYAqguJuE')",
            }}
          />
        </button>
      </header>

      {/* Search Bar */}
      <div className="px-6 py-2">
        <form onSubmit={handleSearch}>
          <label className="flex w-full flex-col">
            <div className="flex h-14 w-full items-center overflow-hidden rounded-full bg-surface-light shadow-sm ring-1 ring-black/5 transition-all focus-within:ring-2 focus-within:ring-primary focus-within:ring-offset-2 dark:bg-surface-dark dark:ring-white/10 dark:focus-within:ring-offset-background-dark">
              <div className="flex items-center justify-center pl-5 text-text-sub-light dark:text-text-sub-dark">
                <span className="material-symbols-outlined text-[24px]">
                  search
                </span>
              </div>
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="h-full flex-1 border-none bg-transparent px-4 text-base font-medium placeholder:text-text-sub-light/70 focus:outline-none focus:ring-0 dark:placeholder:text-text-sub-dark/70"
                placeholder="Search titles or authors..."
              />
              <button
                type="submit"
                className="mr-2 rounded-full p-2 transition-colors hover:bg-black/5 dark:hover:bg-white/10"
              >
                <span className="material-symbols-outlined text-text-sub-light dark:text-text-sub-dark">
                  search
                </span>
              </button>
            </div>
          </label>
        </form>
      </div>

      {/* Filter Chips */}
      <div className="no-scrollbar flex w-full gap-3 overflow-x-auto px-6 py-4">
        <button
          onClick={() => handleGenreClick("All")}
          className={`flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-full pl-6 pr-6 transition-transform active:scale-95 ${
            !selectedGenre
              ? "bg-primary"
              : "bg-surface-light ring-1 ring-black/5 dark:bg-surface-dark dark:ring-white/10"
          }`}
        >
          <span
            className={`text-sm font-bold leading-normal ${
              !selectedGenre
                ? "text-black"
                : "text-text-main-light dark:text-text-main-dark"
            }`}
          >
            All
          </span>
        </button>
        {genres.slice(1).map((genre) => (
          <button
            key={genre}
            onClick={() => handleGenreClick(genre)}
            className={`flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-full pl-6 pr-6 ring-1 ring-black/5 transition-transform active:scale-95 dark:ring-white/10 ${
              selectedGenre === genre
                ? "bg-primary"
                : "bg-surface-light dark:bg-surface-dark"
            }`}
          >
            <span
              className={`text-sm font-medium leading-normal ${
                selectedGenre === genre
                  ? "font-bold text-black"
                  : "text-text-main-light dark:text-text-main-dark"
              }`}
            >
              {genre}
            </span>
          </button>
        ))}
        {statuses.slice(1).map((status) => (
          <button
            key={status}
            onClick={() => handleStatusClick(status)}
            className={`flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-full pl-6 pr-6 ring-1 ring-black/5 transition-transform active:scale-95 dark:ring-white/10 ${
              selectedStatus === status
                ? "bg-primary"
                : "bg-surface-light dark:bg-surface-dark"
            }`}
          >
            <span
              className={`text-sm font-medium leading-normal ${
                selectedStatus === status
                  ? "font-bold text-black"
                  : "text-text-main-light dark:text-text-main-dark"
              }`}
            >
              {status}
            </span>
          </button>
        ))}
      </div>

      {/* Results Grid */}
      <div className="px-6 py-2">
        {loading && (
          <div className="flex items-center justify-center py-12">
            <p className="text-text-sub-light dark:text-text-sub-dark">
              Loading...
            </p>
          </div>
        )}

        {error && (
          <div className="rounded-2xl bg-red-100 px-4 py-3 text-sm font-medium text-red-800 dark:bg-red-900/40 dark:text-red-200">
            {error}
          </div>
        )}

        {!loading && !error && mangas.length === 0 && (
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <span className="material-symbols-outlined mb-4 text-6xl text-text-sub-light dark:text-text-sub-dark">
              search_off
            </span>
            <p className="text-lg font-semibold text-text-main-light dark:text-text-main-dark">
              No results found
            </p>
            <p className="mt-2 text-sm text-text-sub-light dark:text-text-sub-dark">
              Try adjusting your search or filters
            </p>
          </div>
        )}

        {!loading && !error && mangas.length > 0 && (
          <>
            <h3 className="mb-4 flex items-center gap-2 text-xl font-bold">
              {searchQuery || selectedGenre || selectedStatus
                ? "Search Results"
                : "All Manga"}
              {pagination.total > 0 && (
                <span className="text-sm font-normal text-text-sub-light dark:text-text-sub-dark">
                  ({pagination.total}{" "}
                  {pagination.total === 1 ? "result" : "results"})
                </span>
              )}
            </h3>

            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3 pb-4">
              {mangas.map((manga) => (
                <div
                  key={manga.id}
                  onClick={() => handleMangaClick(manga.id)}
                  className="group flex cursor-pointer flex-col gap-2"
                >
                  <div className="relative aspect-[3/4] w-full overflow-hidden rounded-lg shadow-sm transition-all duration-300 group-hover:-translate-y-1 group-hover:shadow-lg">
                    {manga.cover_url ? (
                      <img
                        src={manga.cover_url}
                        alt={manga.title}
                        className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
                        onError={(e) => {
                          (e.target as HTMLImageElement).style.display = "none";
                        }}
                      />
                    ) : (
                      <div className="flex h-full w-full items-center justify-center bg-gray-200 dark:bg-gray-800">
                        <span className="material-symbols-outlined text-2xl text-text-sub-light dark:text-text-sub-dark">
                          image
                        </span>
                      </div>
                    )}
                    {manga.status && (
                      <div className="absolute left-1.5 top-1.5 rounded-full bg-primary px-1.5 py-0.5 text-[8px] font-bold uppercase tracking-wide text-black">
                        {manga.status}
                      </div>
                    )}
                  </div>
                  <div className="px-0.5">
                    <p className="line-clamp-2 text-sm font-bold leading-tight">
                      {manga.title}
                    </p>
                    <p className="mt-0.5 text-[10px] font-medium leading-normal text-text-sub-light dark:text-text-sub-dark line-clamp-1">
                      {manga.author || "Unknown Author"}
                    </p>
                  </div>
                </div>
              ))}
            </div>

            {/* Pagination */}
            {pagination.total_pages > 1 && (
              <div className="mt-6 flex items-center justify-center gap-2 pb-4">
                <button
                  onClick={() => fetchManga(pagination.page - 1)}
                  disabled={pagination.page <= 1}
                  className="rounded-full bg-surface-light px-4 py-2 text-sm font-medium ring-1 ring-black/5 transition-colors disabled:opacity-50 dark:bg-surface-dark dark:ring-white/10"
                >
                  Previous
                </button>
                <span className="text-sm text-text-sub-light dark:text-text-sub-dark">
                  Page {pagination.page} of {pagination.total_pages}
                </span>
                <button
                  onClick={() => fetchManga(pagination.page + 1)}
                  disabled={pagination.page >= pagination.total_pages}
                  className="rounded-full bg-surface-light px-4 py-2 text-sm font-medium ring-1 ring-black/5 transition-colors disabled:opacity-50 dark:bg-surface-dark dark:ring-white/10"
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
      </div>

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
            className="relative flex h-full w-full flex-col items-center justify-center gap-1 text-text-main-light dark:text-white"
          >
            <div className="mb-1 flex items-center justify-center rounded-full bg-primary/20 px-5 py-1 dark:bg-primary/10">
              <span
                className="material-symbols-outlined text-[26px] font-bold text-black dark:text-primary"
                style={{ fontVariationSettings: '"FILL" 1' }}
              >
                explore
              </span>
            </div>
            <span className="text-[10px] font-bold">Discover</span>
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
