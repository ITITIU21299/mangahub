"use client";

import Link from "next/link";

type NavKey = "home" | "discover" | "chat" | "library" | "profile";

interface BottomNavProps {
  active: NavKey;
}

const items: { key: NavKey; href: string; label: string; icon: string }[] = [
  { key: "home", href: "/", label: "Home", icon: "home" },
  { key: "discover", href: "/discover", label: "Discover", icon: "explore" },
  { key: "chat", href: "/chat", label: "Chat", icon: "chat" },
  { key: "library", href: "/library", label: "Library", icon: "library_books" },
  { key: "profile", href: "/profile", label: "Profile", icon: "person" },
];

export default function BottomNav({ active }: BottomNavProps) {
  return (
    <nav className="pb-safe fixed bottom-0 left-0 z-50 w-full border-t border-black/5 bg-surface-light/90 backdrop-blur-lg dark:border-white/10 dark:bg-background-dark/90">
      <div className="flex h-16 items-center justify-around px-2">
        {items.map((item) => {
          const isActive = item.key === active;
          const baseClasses =
            "flex flex-1 flex-col items-center gap-1 p-2 text-[10px] transition-colors";
          const textClasses = isActive
            ? "font-bold text-black dark:text-primary"
            : "font-medium text-text-sub-light hover:text-text-main-light dark:text-text-sub-dark dark:hover:text-white";

          return (
            <Link
              key={item.key}
              href={item.href}
              className={`${baseClasses} ${isActive ? "text-text-main-light dark:text-text-main-dark" : ""}`}
            >
              <div
                className={`flex items-center justify-center rounded-full px-3 py-0.5 ${
                  isActive
                    ? "bg-primary/20 dark:bg-primary/10"
                    : "bg-transparent"
                }`}
              >
                <span
                  className="material-symbols-outlined text-[22px]"
                  style={
                    isActive
                      ? { fontVariationSettings: '"FILL" 1' }
                      : undefined
                  }
                >
                  {item.icon}
                </span>
              </div>
              <span className={textClasses}>{item.label}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}


