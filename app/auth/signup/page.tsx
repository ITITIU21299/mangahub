"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";

export default function SignUpPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    if (!username || !email || !password) {
      setError("Please fill in username, email, and password.");
      return;
    }

    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/auth/register`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ username, email, password }),
      });

      const data = await res.json().catch(() => ({}));

      if (!res.ok) {
        setError(
          data.error ||
            "Registration failed. Please check your details and try again.",
        );
        return;
      }

      setSuccess("Account created! Redirecting to sign in...");
      // Small delay so the user sees the message.
      setTimeout(() => {
        router.push("/auth/signin");
      }, 1200);
    } catch {
      setError("Unable to reach server. Please try again.");
    } finally {
      setLoading(false);
    }
  };
  return (
    <div className="flex min-h-screen w-full justify-center bg-background-light text-text-main-light dark:bg-background-dark dark:text-text-main-dark">
      <div className="relative flex h-full min-h-screen w-full max-w-md flex-col overflow-x-hidden bg-white shadow-2xl dark:bg-[#1a190b]">
        {/* Header */}
        <div className="sticky top-0 z-10 flex items-center justify-between bg-white/80 px-4 pb-2 pt-4 backdrop-blur-md dark:bg-[#1a190b]/80">
          <a
            href="/auth/signin"
            className="flex h-10 w-10 items-center justify-center rounded-full bg-background-light text-text-main transition-colors hover:bg-gray-200 dark:bg-background-dark dark:text-white"
          >
            <span className="material-symbols-outlined">arrow_back</span>
          </a>
          <h2 className="flex-1 pr-10 text-center text-lg font-bold leading-tight tracking-[-0.015em]">
            Sign Up
          </h2>
        </div>

        {/* Hero */}
        <div className="px-4 pb-6 pt-2 text-center">
          <div className="mx-auto mb-6 flex h-32 w-32 items-center justify-center overflow-hidden rounded-full bg-primary/20">
            <img
              alt="Manga artwork"
              className="h-full w-full object-cover opacity-90 mix-blend-multiply dark:mix-blend-normal"
              src="https://lh3.googleusercontent.com/aida-public/AB6AXuBLKFNX3mZ7ZmIwXB2ZgMVBw5NHf0NXBqfwGTU13IslilRpX_s9G2md0J1OkgwclTBOIeLTmgfsNTf25WfHIf0VHmpFQ0hi5OpJLSJHgJ3gImE5nOOOdGHC9zSNxTcuozptBaYdUyMCcYiWOf2XZzILYfQQ8E0zdmM-URyJuxx2hPctNs_DeHgLiICrWBT1z4YWHd_qGeC99RFYmJ-6ZGNYNvPBnTVKQwu4vR5uMna79a3or87yIGr6WNtQlIosWPTbrgO23YaaZL0"
            />
          </div>
          <h1 className="px-4 pb-2 text-[32px] font-bold leading-tight tracking-tight">
            Start Your Collection
          </h1>
          <p className="px-4 text-base font-normal leading-normal text-text-subtle dark:text-gray-400">
            Create an account to track, rate, and discover new manga series.
          </p>
        </div>

        {/* Form */}
        <form className="flex flex-col gap-4 px-6 pb-4" onSubmit={handleSubmit}>
          <div className="flex flex-col gap-1">
            <label
              htmlFor="username"
              className="pl-3 text-sm font-medium text-text-main dark:text-white"
            >
              Username
            </label>
            <div className="relative">
              <input
                id="username"
                type="text"
                placeholder="OtakuKing99"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="form-input h-14 w-full rounded-full border-2 border-[#e6e6db] bg-background-light pl-5 pr-12 text-base font-medium text-text-main shadow-sm transition-all placeholder:text-text-subtle focus:border-primary focus:ring-0 dark:border-gray-700 dark:bg-background-dark dark:text-white"
              />
              <span className="material-symbols-outlined absolute right-4 top-1/2 -translate-y-1/2 text-success">
                check_circle
              </span>
            </div>
          </div>

          <div className="flex flex-col gap-1">
            <label
              htmlFor="email"
              className="pl-3 text-sm font-medium text-text-main dark:text-white"
            >
              Email
            </label>
            <div className="relative">
              <input
                id="email"
                type="email"
                placeholder="name@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="form-input h-14 w-full rounded-full border-2 border-[#e6e6db] bg-background-light pl-5 pr-12 text-base font-medium text-text-main shadow-sm transition-all placeholder:text-text-subtle focus:border-primary focus:ring-0 dark:border-gray-700 dark:bg-background-dark dark:text-white"
              />
              <span className="material-symbols-outlined absolute right-4 top-1/2 -translate-y-1/2 text-text-subtle">
                mail
              </span>
            </div>
          </div>

          <div className="flex flex-col gap-1">
            <label
              htmlFor="password"
              className="pl-3 text-sm font-medium text-text-main dark:text-white"
            >
              Password
            </label>
            <div className="relative">
              <input
                id="password"
                type="password"
                placeholder="Min. 8 characters"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="form-input h-14 w-full rounded-full border-2 border-[#e6e6db] bg-background-light pl-5 pr-12 text-base font-medium text-text-main shadow-sm transition-all placeholder:text-text-subtle focus:border-primary focus:ring-0 dark:border-gray-700 dark:bg-background-dark dark:text-white"
              />
              <button
                type="button"
                className="absolute right-4 top-1/2 -translate-y-1/2 text-text-subtle hover:text-text-main dark:hover:text-white"
              >
                <span className="material-symbols-outlined">visibility_off</span>
              </button>
            </div>
          </div>

          {error && (
            <p className="mt-2 rounded-2xl bg-red-100 px-4 py-2 text-sm font-medium text-red-800 dark:bg-red-900/40 dark:text-red-200">
              {error}
            </p>
          )}
          {success && (
            <p className="mt-2 rounded-2xl bg-emerald-100 px-4 py-2 text-sm font-medium text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200">
              {success}
            </p>
          )}
          <button
            type="submit"
            disabled={loading}
            className="group mt-4 flex h-14 w-full items-center justify-center rounded-full bg-primary px-8 text-lg font-bold tracking-tight text-[#181811] shadow-lg shadow-primary/20 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-primary/40 active:translate-y-0 active:shadow-none disabled:cursor-not-allowed disabled:opacity-70"
          >
            {loading ? "Registering..." : "Register"}
            <span className="material-symbols-outlined ml-2 transition-transform group-hover:translate-x-1">
              arrow_forward
            </span>
          </button>
        </form>

        {/* Footer */}
        <div className="safe-pb mt-auto flex flex-col items-center gap-4 px-4 pb-8">
          <p className="text-center text-base font-normal text-text-main dark:text-white">
            Already have an account?{" "}
            <a
              href="/auth/signin"
              className="font-bold text-black underline decoration-2 underline-offset-4 decoration-primary dark:text-primary"
            >
              Log in here
            </a>
          </p>
          <p className="max-w-xs text-center text-xs leading-relaxed text-text-subtle">
            By registering, you agree to our{" "}
            <a className="underline hover:text-text-main dark:hover:text-white">
              Terms of Service
            </a>{" "}
            and{" "}
            <a className="underline hover:text-text-main dark:hover:text-white">
              Privacy Policy
            </a>
            .
          </p>
        </div>
      </div>
    </div>
  );
}


