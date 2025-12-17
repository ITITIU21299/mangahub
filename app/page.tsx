export default function Home() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
      <div className="mx-auto flex max-w-md flex-col items-center gap-6 px-6 text-center">
        <h1 className="text-3xl font-bold tracking-tight">MangaHub</h1>
        <p className="text-sm text-text-sub-light dark:text-text-sub-dark">
          Simple manga tracking demo for your Net Centric project. Use the links
          below to explore the sample UI screens.
        </p>
        <div className="flex w-full flex-col gap-3">
          <a
            href="/discover"
            className="w-full rounded-full bg-primary px-4 py-3 text-sm font-bold text-black hover:opacity-90"
          >
            Open Discover / Search
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
    </div>
  );
}
