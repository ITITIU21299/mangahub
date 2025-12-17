export default function DiscoverPage() {
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
        <label className="flex w-full flex-col">
          <div className="flex h-14 w-full items-center overflow-hidden rounded-full bg-surface-light shadow-sm ring-1 ring-black/5 transition-all focus-within:ring-2 focus-within:ring-primary focus-within:ring-offset-2 dark:bg-surface-dark dark:ring-white/10 dark:focus-within:ring-offset-background-dark">
            <div className="flex items-center justify-center pl-5 text-text-sub-light dark:text-text-sub-dark">
              <span className="material-symbols-outlined text-[24px]">
                search
              </span>
            </div>
            <input
              className="h-full flex-1 border-none bg-transparent px-4 text-base font-medium placeholder:text-text-sub-light/70 focus:outline-none focus:ring-0 dark:placeholder:text-text-sub-dark/70"
              placeholder="Search titles or authors..."
            />
            <button className="mr-2 rounded-full p-2 transition-colors hover:bg-black/5 dark:hover:bg-white/10">
              <span className="material-symbols-outlined text-text-sub-light dark:text-text-sub-dark">
                tune
              </span>
            </button>
          </div>
        </label>
      </div>

      {/* Filter Chips */}
      <div className="no-scrollbar flex w-full gap-3 overflow-x-auto px-6 py-4">
        <button className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-full bg-primary pl-6 pr-6 active:scale-95">
          <span className="text-sm font-bold leading-normal text-black">
            All
          </span>
        </button>
        <button className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-full bg-surface-light pl-6 pr-6 ring-1 ring-black/5 transition-transform active:scale-95 dark:bg-surface-dark dark:ring-white/10">
          <span className="text-sm font-medium leading-normal text-text-main-light dark:text-text-main-dark">
            Action
          </span>
        </button>
        <button className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-full bg-surface-light pl-6 pr-6 ring-1 ring-black/5 transition-transform active:scale-95 dark:bg-surface-dark dark:ring-white/10">
          <span className="text-sm font-medium leading-normal text-text-main-light dark:text-text-main-dark">
            Romance
          </span>
        </button>
        <button className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-full bg-surface-light pl-6 pr-6 ring-1 ring-black/5 transition-transform active:scale-95 dark:bg-surface-dark dark:ring-white/10">
          <span className="text-sm font-medium leading-normal text-text-main-light dark:text-text-main-dark">
            Ongoing
          </span>
        </button>
        <button className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-full bg-surface-light pl-6 pr-6 ring-1 ring-black/5 transition-transform active:scale-95 dark:bg-surface-dark dark:ring-white/10">
          <span className="text-sm font-medium leading-normal text-text-main-light dark:text-text-main-dark">
            Fantasy
          </span>
        </button>
      </div>

      {/* Results Grid (static demo content for now) */}
      <div className="px-6 py-2">
        <h3 className="mb-4 flex items-center gap-2 text-xl font-bold">
          Trending Now
          <span
            className="material-symbols-outlined text-primary"
            style={{ fontVariationSettings: '"FILL" 1' }}
          >
            local_fire_department
          </span>
        </h3>

        <div className="grid grid-cols-2 gap-4 pb-4">
          {/* Example cards – real data will come from the Go backend API later */}
          {/* One Piece */}
          <div className="group flex cursor-pointer flex-col gap-3">
            <div className="relative aspect-[3/4] w-full overflow-hidden rounded-xl shadow-sm transition-all duration-300 group-hover:-translate-y-1 group-hover:shadow-md">
              <div
                className="h-full w-full bg-cover bg-center bg-no-repeat transform transition-transform duration-500 group-hover:scale-110"
                style={{
                  backgroundImage:
                    "url('https://lh3.googleusercontent.com/aida-public/AB6AXuAC76vJiO96a7t_saHP5NRPBjaOFVv-VFBP34HyNRCsiKT9aRGe7pFLdeohzNVp-vY_xJvBP3l1glPKE_S-re_f9f_YXF6jB9MUH6Nrj8L1XVjxJ8OmEE8KMzj498PQbCOfO3UGJdfuAHl3bCq8l7elzNyq9di_aJbMctX8b9Q5atfD8ZCW4z0kOEcBRGtkYV3ShrRIQ13As-UxdFcTNbcvF56OgoTb2qsiwBOFsXu7aLBZH33z8dQpAVeUlp6OeBGZx_xB3HA2f5c')",
                }}
              />
              <div className="absolute left-2 top-2 rounded-full bg-primary px-2 py-1 text-[10px] font-bold uppercase tracking-wide text-black">
                Ongoing
              </div>
            </div>
            <div>
              <p className="line-clamp-1 text-base font-bold leading-tight">
                One Piece
              </p>
              <p className="mt-1 text-xs font-medium leading-normal text-text-sub-light dark:text-text-sub-dark">
                Eiichiro Oda
              </p>
            </div>
          </div>

          {/* Jujutsu Kaisen */}
          <div className="group flex cursor-pointer flex-col gap-3">
            <div className="relative aspect-[3/4] w-full overflow-hidden rounded-xl shadow-sm transition-all duration-300 group-hover:-translate-y-1 group-hover:shadow-md">
              <div
                className="h-full w-full bg-cover bg-center bg-no-repeat transform transition-transform duration-500 group-hover:scale-110"
                style={{
                  backgroundImage:
                    "url('https://lh3.googleusercontent.com/aida-public/AB6AXuDgZLAFKgSR3Egkz2gW4YFu0Q4xEqWwaSapacyfp84nkIcMRulO9nJmm5skwpD7WYTgzVRnN-GgTStveFGJIT608mzyo_BqwSzM5rA1MiVR759ufsX0FzS2eokaWvZaA-0DJmUDvQng0c42RxFvDc-UE4Dcw5OOSeSZH_ylRB2Xv1LbVN8u8AZ6HWBwwfB5YHrsJ5D_MbI77PRaU5a9fGGZr36DKUg5SVX3wKpn4IvA2iqf1SyOu02j2Vr6h0iQPgkoeQXSwXVI8eM')",
                }}
              />
              <div className="absolute left-2 top-2 rounded-full bg-surface-light/90 px-2 py-1 text-[10px] font-bold uppercase tracking-wide text-black dark:bg-black/80 dark:text-white">
                Hot
              </div>
            </div>
            <div>
              <p className="line-clamp-1 text-base font-bold leading-tight">
                Jujutsu Kaisen
              </p>
              <p className="mt-1 text-xs font-medium leading-normal text-text-sub-light dark:text-text-sub-dark">
                Gege Akutami
              </p>
            </div>
          </div>

          {/* Other demo cards omitted for brevity */}
        </div>
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
            href="/auth/signin"
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


