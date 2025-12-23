"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

export default function SignInPage() {
  const router = useRouter();
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!identifier || !password) {
      setError("Please enter your email/username and password.");
      return;
    }

    setLoading(true);
    try {
      const body: Record<string, string> = { password };
      // Simple heuristic: looks like an email → email, otherwise username.
      if (identifier.includes("@")) {
        body.email = identifier;
      } else {
        body.username = identifier;
      }

      // Call Next.js API route, which proxies to the Go backend.
      const res = await fetch(`/api/auth/login`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
      });

      const data = await res.json().catch(() => ({}));

      if (!res.ok) {
        setError(data.error || "Login failed. Please check your credentials.");
        return;
      }

      if (data.token) {
        localStorage.setItem("mangahub_token", data.token);
      }

      router.push("/discover");
    } catch {
      setError("Unable to reach server. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen w-full justify-center bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
      <div className="relative flex h-full min-h-screen w-full max-w-md flex-col overflow-hidden border-x border-neutral-100 bg-white shadow-xl dark:border-neutral-800 dark:bg-[#1a1a0b] sm:my-8 sm:min-h-0 sm:h-[850px] sm:rounded-[3rem]">
        {/* Decorative header */}
        <div className="absolute top-0 left-0 z-0 h-64 w-full overflow-hidden rounded-b-[3rem] bg-background-light dark:bg-background-dark">
          <div
            className="absolute inset-0 bg-[length:100px_100px] bg-repeat opacity-[0.05] dark:opacity-[0.1]"
            style={{
              backgroundImage:
                "url('https://lh3.googleusercontent.com/aida-public/AB6AXuABEFtOtatLquMo1VadqcITInVo_W2P4mMZtqJVZXPXutDIueVu4IFAqZTjCBeUyLgEAYbonOoi5GLdRhQMekrb1P6175TZmFvaG2uyie9N45GvleY3pHg2LIy6lUQ_bb5sWej_apRstySDEX0jLbdvsIsYjIRxwvas0NynaGJGytAlEYqXQe_c1mdFrppO_-SeGwfBHHAPJ9cVOu3arJ6TyKMiIARCEhvKgAdZxvc4w_dpbc6f7Uj5oHMffMu8vIiTUFx7F2Ah29Q')",
            }}
          />
          <div className="absolute -bottom-8 left-1/2 flex h-48 w-48 -translate-x-1/2 items-center justify-center rounded-full bg-primary/20 p-4">
            <div
              className="h-full w-full rounded-full border-4 border-white bg-cover bg-center shadow-inner dark:border-[#1a1a0b]"
              style={{
                backgroundImage:
                  "url('https://lh3.googleusercontent.com/aida-public/AB6AXuDWzM25XkTBcSfzTLyj50n18JFppDaWSRR3ppdWdeDn1TcaLSQjRZDjH7rnb-dUw_FC4xSezpt8XamezshRHwjhURiujtATGVHD1PiDuTsAD_Vbvba_StqqlZEzHQLEjiOplq4v9prTEM7HsRqO6TVpszMA-7zSodxACGceVjOLpfsIaIT3AOv2Xc6g9b6hg_3ttI82D4GMR7fx5F2lrBYdvKx6q2yYR4XeFhWzFHXV-2VvBh1_0D_Uc6CVRhMkEBjou_OpImmjmkY')",
              }}
            />
          </div>
        </div>

        {/* Content */}
        <div className="relative z-10 flex h-full flex-col px-6 pb-8 pt-60">
          <div className="mb-8 flex flex-col items-center space-y-2 text-center">
            <h1 className="text-3xl font-bold tracking-tight">Welcome Back!</h1>
            <p className="text-base font-normal text-neutral-medium dark:text-gray-400">
              Track your reading journey, one chapter at a time.
            </p>
          </div>

          <form className="flex w-full flex-col gap-5" onSubmit={handleSubmit}>
            <div className="group flex flex-col gap-2">
              <label
                htmlFor="email"
                className="ml-4 text-sm font-semibold text-neutral-dark dark:text-gray-200"
              >
                Email or Username
              </label>
              <div className="relative flex items-center">
                <input
                  id="email"
                  type="text"
                  placeholder="otaku@example.com or otaku99"
                  value={identifier}
                  onChange={(e) => setIdentifier(e.target.value)}
                  className="h-14 w-full rounded-full border border-transparent bg-background-light pl-5 pr-4 text-neutral-dark outline-none transition-all placeholder:text-neutral-medium/50 focus:border-primary focus:ring-2 focus:ring-primary/50 dark:bg-background-dark dark:text-white"
                />
                <span className="material-symbols-outlined absolute right-5 text-neutral-medium">
                  person
                </span>
              </div>
            </div>

            <div className="group flex flex-col gap-2">
              <label
                htmlFor="password"
                className="ml-4 text-sm font-semibold text-neutral-dark dark:text-gray-200"
              >
                Password
              </label>
              <div className="relative flex items-center">
                <input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="h-14 w-full rounded-full border border-transparent bg-background-light pl-5 pr-12 text-neutral-dark outline-none transition-all placeholder:text-neutral-medium/50 focus:border-primary focus:ring-2 focus:ring-primary/50 dark:bg-background-dark dark:text-white"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((prev) => !prev)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center justify-center rounded-full p-2 text-neutral-medium transition-colors hover:bg-black/5 dark:hover:bg-white/10"
                >
                  <span className="material-symbols-outlined">
                    {showPassword ? "visibility_off" : "visibility"}
                  </span>
                </button>
              </div>
              <div className="mt-1 flex justify-end px-2">
                <button
                  type="button"
                  className="text-sm font-medium text-neutral-medium transition-colors hover:text-neutral-dark dark:text-gray-400 dark:hover:text-primary"
                >
                  Forgot Password?
                </button>
              </div>
            </div>

            <div className="mt-4 flex flex-col gap-4">
              {error && (
                <p className="rounded-2xl bg-red-100 px-4 py-2 text-sm font-medium text-red-800 dark:bg-red-900/40 dark:text-red-200">
                  {error}
                </p>
              )}
              <button
                type="submit"
                disabled={loading}
                className="h-14 w-full rounded-full bg-primary text-lg font-bold tracking-wide text-neutral-dark shadow-md transition-all hover:bg-[#e6e205] hover:shadow-lg active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-70"
              >
                {loading ? "Logging in..." : "LOG IN"}
              </button>

              <div className="relative flex items-center justify-center py-2">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-neutral-light dark:border-neutral-700" />
                </div>
                <span className="relative bg-white px-4 text-sm text-neutral-medium dark:bg-[#1a1a0b]">
                  Or continue as{" "}
                  <a
                    href="/discover"
                    className="font-semibold text-primary underline decoration-primary/60 underline-offset-4 hover:opacity-80"
                  >
                    guest
                  </a>
                </span>
              </div>
            </div>
          </form>

          <div className="mt-auto pt-8 text-center">
            <p className="text-neutral-dark dark:text-gray-300">
              Don&apos;t have an account?{" "}
              <a
                href="/auth/signup"
                className="font-bold text-neutral-dark underline decoration-primary decoration-2 underline-offset-4 transition-opacity hover:opacity-80 dark:text-primary"
              >
                Sign Up
              </a>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
