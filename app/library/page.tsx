export default function LibraryPage() {
  return (
    <div className="relative flex min-h-screen w-full flex-col overflow-x-hidden pb-24 bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
      {/* Header */}
      <header className="sticky top-0 z-20 bg-background-light/95 px-4 pb-2 pt-4 backdrop-blur-md dark:bg-background-dark/95">
        <div className="flex items-center justify-between">
          <h2 className="flex-1 text-3xl font-bold leading-tight tracking-[-0.015em]">
            My Library
          </h2>
          <div className="flex w-12 items-center justify-end">
            <button className="flex h-10 w-10 items-center justify-center rounded-full bg-surface-light text-text-main-light shadow-sm transition-colors hover:bg-gray-100 dark:bg-surface-dark dark:text-text-main-dark dark:hover:bg-gray-800">
              <span className="material-symbols-outlined">person</span>
            </button>
          </div>
        </div>

        {/* Filter chips (static for now) */}
        <div className="no-scrollbar mt-3 flex w-full gap-3 overflow-x-auto border-b border-gray-100 pb-3 dark:border-gray-800/50">
          <button className="flex h-9 shrink-0 items-center justify-center gap-x-2 rounded-full bg-primary px-5 shadow-sm active:scale-95">
            <p className="text-sm font-bold leading-normal text-black">All</p>
          </button>
          <button className="flex h-9 shrink-0 items-center justify-center gap-x-2 rounded-full border border-gray-100 bg-surface-light px-5 text-sm font-medium shadow-sm active:scale-95 dark:border-gray-800 dark:bg-surface-dark">
            <p>Reading</p>
          </button>
          <button className="flex h-9 shrink-0 items-center justify-center gap-x-2 rounded-full border border-gray-100 bg-surface-light px-5 text-sm font-medium shadow-sm active:scale-95 dark:border-gray-800 dark:bg-surface-dark">
            <p>Completed</p>
          </button>
          <button className="flex h-9 shrink-0 items-center justify-center gap-x-2 rounded-full border border-gray-100 bg-surface-light px-5 text-sm font-medium shadow-sm active:scale-95 dark:border-gray-800 dark:bg-surface-dark">
            <p>Plan to Read</p>
          </button>
        </div>
      </header>

      {/* Continue Reading – static demo content */}
      <section className="mt-4">
        <h2 className="px-4 pb-3 text-[22px] font-bold leading-tight tracking-[-0.015em]">
          Continue Reading
        </h2>
        <div className="no-scrollbar flex gap-4 overflow-y-auto pb-4 pl-4">
          <div className="flex min-w-[280px] w-[280px] snap-start flex-col gap-4 rounded-2xl bg-surface-light p-4 shadow-[0_4px_20px_rgba(0,0,0,0.03)] transition-colors dark:bg-surface-dark">
            <div className="relative aspect-[3/4] w-full overflow-hidden rounded-xl bg-gray-200 dark:bg-gray-800">
              <div
                className="absolute inset-0 bg-cover bg-center transition-transform duration-500 hover:scale-105"
                style={{
                  backgroundImage:
                    "url('https://lh3.googleusercontent.com/aida-public/AB6AXuDMYgnB2XkGDInEh7MqPo73MRYqFHI09nEZfLZYHqIgwVm0FYNVZB2gIs_0QdDhlkrfvbM-g0eVSufBWtpaDAoBEZC_2ER1Jb-LudjI0ikrHi0QtmBN4NfU5v3NQQ_ihcO1eZ51pJ5M8K9X2OQCStqXLRa4_M6WtYAtVk2jXK2mt_UuFQlGhgnj-MqRfTvlo6Jp5vz632gqxHh2ZUstR2nbA2JZYs1P3efs6KbFsWBeeFQ_E5XwZ8Trb9hCtUZXOgGndTAUdsbCQ1A')",
                }}
              />
            </div>
            <div className="flex flex-col gap-3">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="line-clamp-1 text-lg font-bold leading-tight">
                    One Piece
                  </h3>
                  <p className="mt-1 text-sm text-text-sub-light dark:text-text-sub-dark">
                    Eiichiro Oda
                  </p>
                </div>
                <div className="flex flex-col items-end">
                  <span className="text-sm font-bold text-primary">
                    Ch. 1050
                  </span>
                  <span className="text-xs text-text-sub-light dark:text-text-sub-dark">
                    of 1100+
                  </span>
                </div>
              </div>
              <div className="h-2 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-gray-700">
                <div
                  className="h-full rounded-full bg-primary"
                  style={{ width: "85%" }}
                />
              </div>
              <button className="mt-1 flex h-11 w-full cursor-pointer items-center justify-center rounded-full bg-primary text-sm font-bold text-black shadow-sm transition-all active:scale-[0.98] hover:shadow-md">
                Update Progress
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Up Next – static list */}
      <section className="mt-2 px-4">
        <div className="flex items-center justify-between pb-3 pt-2">
          <h2 className="text-[22px] font-bold leading-tight tracking-[-0.015em]">
            Up Next
          </h2>
          <button className="text-sm font-medium text-primary hover:opacity-80">
            See all
          </button>
        </div>

        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-4 rounded-2xl bg-surface-light p-3 shadow-sm transition-all hover:border-primary/20 dark:bg-surface-dark">
            <div className="h-20 w-16 shrink-0 overflow-hidden rounded-lg bg-gray-200">
              <div
                className="h-full w-full bg-cover bg-center"
                style={{
                  backgroundImage:
                    "url('https://lh3.googleusercontent.com/aida-public/AB6AXuCcUU54O0iMnve5Ul0HlHRvFEqnW9_ksl1Y8s5icZBUuzlLFUr0g9tWFze96dn9kxfZedkZC1gv0GwL3QXG17412YQwRFuLzjSyIdUt_oc7w0r24_zCfunN3qVR1bWa1w3J1nqY_XTP9PC8rcvJD6QCiHnnDpqOR6rI2euUz75yYZftzeiBkGzY120KkCBPBj3Y4OgLExtirWmse4jB8bY-oFUR9maKLhsqCQm08vCC7q3LOZdEXZkqyjnv20vbayNkHFzzgwkP6JU')",
                }}
              />
            </div>
            <div className="flex flex-1 flex-col justify-center gap-1">
              <h3 className="text-base font-bold leading-tight">
                Chainsaw Man
              </h3>
              <p className="text-xs text-text-sub-light dark:text-text-sub-dark">
                Tatsuki Fujimoto
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Floating Action Button */}
      <button className="fixed bottom-24 right-5 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-primary text-black shadow-[0_4px_12px_rgba(249,245,6,0.5)] transition-all hover:scale-105 active:scale-95">
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
