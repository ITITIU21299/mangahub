"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import BottomNav from "@/components/BottomNav";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";

interface UserInfo {
  id: string;
  username: string;
  email: string;
}

export default function ProfilePage() {
  const router = useRouter();
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [libraryCount, setLibraryCount] = useState(0);

  useEffect(() => {
    const token = localStorage.getItem("mangahub_token");
    if (!token) {
      router.push("/auth/signin");
      return;
    }

    // Decode JWT to get user info (simple client-side decode)
    try {
      const payload = JSON.parse(atob(token.split(".")[1]));
      setUserInfo({
        id: payload.sub || "",
        username: payload.usr || "User",
        email: payload.email || "",
      });

      // Fetch library count
      fetchLibraryCount(token);
    } catch (err) {
      console.error("Failed to decode token:", err);
      localStorage.removeItem("mangahub_token");
      router.push("/auth/signin");
    } finally {
      setLoading(false);
    }
  }, [router]);

  const fetchLibraryCount = async (token: string) => {
    try {
      const res = await fetch(`${API_BASE}/users/library`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (res.ok) {
        const data = await res.json();
        setLibraryCount(Array.isArray(data) ? data.length : 0);
      }
    } catch (err) {
      console.error("Failed to fetch library count:", err);
      setLibraryCount(0);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem("mangahub_token");
    router.push("/auth/signin");
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

  if (!userInfo) {
    return null;
  }

  return (
    <div className="relative flex min-h-screen w-full flex-col overflow-x-hidden pb-24 bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
      {/* Header */}
      <header className="sticky top-0 z-20 flex items-center justify-between bg-background-light/90 px-6 py-4 pt-6 backdrop-blur-md dark:bg-background-dark/90">
        <h2 className="flex-1 text-3xl font-bold leading-tight tracking-tight">
          Profile
        </h2>
      </header>

      {/* Profile Content */}
      <div className="flex flex-col px-6 py-6">
        {/* Profile Card */}
        <div className="mb-6 rounded-2xl bg-surface-light p-6 shadow-sm ring-1 ring-black/5 dark:bg-surface-dark dark:ring-white/10">
          <div className="flex flex-col items-center gap-4">
            {/* Avatar */}
            <div className="relative flex h-24 w-24 items-center justify-center overflow-hidden rounded-full bg-primary ring-4 ring-background-light dark:ring-background-dark">
              <span className="text-4xl font-bold text-black">
                {userInfo.username.charAt(0).toUpperCase()}
              </span>
            </div>

            {/* User Info */}
            <div className="flex flex-col items-center gap-1">
              <h3 className="text-2xl font-bold">{userInfo.username}</h3>
              <p className="text-sm text-text-sub-light dark:text-text-sub-dark">
                {userInfo.email}
              </p>
            </div>
          </div>
        </div>

        {/* Stats Card */}
        <div className="mb-6 grid grid-cols-2 gap-4">
          <div className="rounded-xl bg-surface-light p-4 shadow-sm ring-1 ring-black/5 dark:bg-surface-dark dark:ring-white/10">
            <p className="text-sm font-medium text-text-sub-light dark:text-text-sub-dark">
              Library
            </p>
            <p className="mt-1 text-2xl font-bold">{libraryCount}</p>
            <p className="mt-0.5 text-xs text-text-sub-light dark:text-text-sub-dark">
              Manga
            </p>
          </div>
          <div className="rounded-xl bg-surface-light p-4 shadow-sm ring-1 ring-black/5 dark:bg-surface-dark dark:ring-white/10">
            <p className="text-sm font-medium text-text-sub-light dark:text-text-sub-dark">
              Reading
            </p>
            <p className="mt-1 text-2xl font-bold">-</p>
            <p className="mt-0.5 text-xs text-text-sub-light dark:text-text-sub-dark">
              Active
            </p>
          </div>
        </div>

        {/* Menu Items */}
        <div className="flex flex-col gap-2">
          <a
            href="/library"
            className="flex items-center gap-4 rounded-xl bg-surface-light p-4 shadow-sm ring-1 ring-black/5 transition-colors hover:bg-gray-100 dark:bg-surface-dark dark:ring-white/10 dark:hover:bg-gray-800"
          >
            <span className="material-symbols-outlined text-2xl text-text-main-light dark:text-text-main-dark">
              collections_bookmark
            </span>
            <div className="flex-1">
              <p className="font-semibold text-text-main-light dark:text-text-main-dark">
                My Library
              </p>
              <p className="text-xs text-text-sub-light dark:text-text-sub-dark">
                View your saved manga
              </p>
            </div>
            <span className="material-symbols-outlined text-text-sub-light dark:text-text-sub-dark">
              chevron_right
            </span>
          </a>

          <a
            href="/discover"
            className="flex items-center gap-4 rounded-xl bg-surface-light p-4 shadow-sm ring-1 ring-black/5 transition-colors hover:bg-gray-100 dark:bg-surface-dark dark:ring-white/10 dark:hover:bg-gray-800"
          >
            <span className="material-symbols-outlined text-2xl text-text-main-light dark:text-text-main-dark">
              explore
            </span>
            <div className="flex-1">
              <p className="font-semibold text-text-main-light dark:text-text-main-dark">
                Discover
              </p>
              <p className="text-xs text-text-sub-light dark:text-text-sub-dark">
                Find new manga
              </p>
            </div>
            <span className="material-symbols-outlined text-text-sub-light dark:text-text-sub-dark">
              chevron_right
            </span>
          </a>

          <div className="flex items-center gap-4 rounded-xl bg-surface-light p-4 shadow-sm ring-1 ring-black/5 transition-colors hover:bg-gray-100 dark:bg-surface-dark dark:ring-white/10 dark:hover:bg-gray-800">
            <span className="material-symbols-outlined text-2xl text-text-main-light dark:text-text-main-dark">
              settings
            </span>
            <div className="flex-1">
              <p className="font-semibold text-text-main-light dark:text-text-main-dark">
                Settings
              </p>
              <p className="text-xs text-text-sub-light dark:text-text-sub-dark">
                App preferences
              </p>
            </div>
            <span className="material-symbols-outlined text-text-sub-light dark:text-text-sub-dark">
              chevron_right
            </span>
          </div>
        </div>

        {/* Logout Button */}
        <button
          onClick={handleLogout}
          className="mt-6 flex w-full items-center justify-center gap-2 rounded-xl bg-red-100 px-4 py-4 text-base font-semibold text-red-800 transition-colors hover:bg-red-200 dark:bg-red-900/40 dark:text-red-200 dark:hover:bg-red-900/60"
        >
          <span className="material-symbols-outlined">logout</span>
          Log Out
        </button>
      </div>

      {/* Bottom Navigation */}
      <BottomNav active="profile" />
    </div>
  );
}
