import BottomNav from "@/components/BottomNav";

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
      <div className="mx-auto flex max-w-md flex-col items-center gap-6 px-6 text-center">
        <h1 className="text-3xl font-bold tracking-tight">MangaHub</h1>
        <div className="flex w-full flex-col gap-3">
          <a
            href="/discover"
            className="w-full rounded-full bg-primary px-4 py-3 text-sm font-bold text-black hover:opacity-90"
          >
            Open Discover / Search
          </a>
          <a
            href="/chat"
            className="w-full rounded-full bg-primary/10 px-4 py-3 text-sm font-semibold text-text-main-light hover:bg-primary/20 dark:text-text-main-dark"
          >
            Open Chat
          </a>
          <a
            href="/library"
            className="w-full rounded-full border border-text-sub-light/40 px-4 py-3 text-sm font-medium hover:bg-surface-light dark:hover:bg-surface-dark"
          >
            View Library
          </a>
          <a
            href="/auth/signin"
            className="w-full rounded-full px-4 py-3 text-xs font-medium text-text-sub-light dark:text-text-sub-dark hover:underline"
          >
            Sign in / Sign up
          </a>
        </div>
      </div>

      <BottomNav active="home" />
    </div>
  );
}
