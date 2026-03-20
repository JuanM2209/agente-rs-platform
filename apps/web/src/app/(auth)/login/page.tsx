"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import type { Metadata } from "next";
import { login } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [mfaCode, setMfaCode] = useState("");
  const [step, setStep] = useState<"credentials" | "mfa">("credentials");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleCredentials = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await login({ email, password });
      localStorage.setItem("nucleus_token", res.token);
      localStorage.setItem("nucleus_user", JSON.stringify(res.user));
      router.push("/dashboard");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Login failed";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-background flex">
      {/* Left panel — branding */}
      <div className="hidden lg:flex lg:w-[480px] flex-col justify-between p-12 bg-surface-container-low border-r border-outline-variant/10">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl gradient-primary flex items-center justify-center">
            <span className="material-symbols-outlined text-on-primary text-2xl">hub</span>
          </div>
          <span className="font-headline font-bold text-xl text-on-surface">Nucleus</span>
        </div>

        <div>
          <h1 className="font-headline text-4xl font-extrabold text-on-surface leading-tight mb-4">
            Industrial Remote<br />
            <span className="text-primary">Access Platform</span>
          </h1>
          <p className="text-on-surface-variant text-base leading-relaxed">
            Centralized telemetry and secure remote orchestration for thousands of edge devices.
          </p>

          <div className="mt-10 space-y-4">
            {[
              { icon: "search", text: "Search-first device access" },
              { icon: "lock", text: "Session-based security with full audit trail" },
              { icon: "settings_ethernet", text: "Modbus serial bridge on demand" },
              { icon: "verified_user", text: "Multi-tenant RBAC enforcement" },
            ].map((item) => (
              <div key={item.icon} className="flex items-center gap-3">
                <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
                  <span className="material-symbols-outlined text-primary text-base">{item.icon}</span>
                </div>
                <span className="text-sm text-on-surface-variant">{item.text}</span>
              </div>
            ))}
          </div>
        </div>

        <p className="text-xs text-on-surface-variant/40 font-technical">
          v1.0.0 © 2024 Nucleus Systems
        </p>
      </div>

      {/* Right panel — login form */}
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-full max-w-md">
          {/* Mobile logo */}
          <div className="lg:hidden flex items-center gap-3 mb-8">
            <div className="w-10 h-10 rounded-xl gradient-primary flex items-center justify-center">
              <span className="material-symbols-outlined text-on-primary text-2xl">hub</span>
            </div>
            <span className="font-headline font-bold text-xl text-on-surface">Nucleus Portal</span>
          </div>

          <div className="bg-surface-container-high rounded-xl p-8">
            <h2 className="font-headline text-2xl font-bold text-on-surface mb-1">
              {step === "credentials" ? "Sign in" : "Verify identity"}
            </h2>
            <p className="text-sm text-on-surface-variant mb-8">
              {step === "credentials"
                ? "Enter your credentials to access the portal"
                : `MFA code sent to ${email}`}
            </p>

            {error && (
              <div className="mb-6 p-4 rounded-xl bg-error-container/20 border border-error/20 flex items-center gap-3">
                <span className="material-symbols-outlined text-error text-base">error</span>
                <p className="text-sm text-error">{error}</p>
              </div>
            )}

            {step === "credentials" ? (
              <form onSubmit={handleCredentials} className="space-y-5">
                <div>
                  <label className="block text-sm text-on-surface-variant mb-2 font-medium">
                    Email address
                  </label>
                  <div className="relative">
                    <span className="absolute left-4 top-1/2 -translate-y-1/2 material-symbols-outlined text-outline text-lg">
                      mail
                    </span>
                    <input
                      type="email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      required
                      placeholder="operator@company.com"
                      className="w-full bg-surface-container-highest rounded-xl pl-11 pr-4 py-4 text-on-surface placeholder:text-outline border-b-2 border-transparent focus:border-primary outline-none transition-colors text-sm"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm text-on-surface-variant mb-2 font-medium">
                    Password
                  </label>
                  <div className="relative">
                    <span className="absolute left-4 top-1/2 -translate-y-1/2 material-symbols-outlined text-outline text-lg">
                      lock
                    </span>
                    <input
                      type="password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      required
                      placeholder="••••••••••••"
                      className="w-full bg-surface-container-highest rounded-xl pl-11 pr-4 py-4 text-on-surface placeholder:text-outline border-b-2 border-transparent focus:border-primary outline-none transition-colors text-sm"
                    />
                  </div>
                </div>

                <div className="flex items-center justify-between text-sm">
                  <label className="flex items-center gap-2 text-on-surface-variant cursor-pointer">
                    <input type="checkbox" className="rounded" />
                    Remember this device
                  </label>
                  <button type="button" className="text-primary hover:text-primary-fixed-dim transition-colors">
                    Forgot password?
                  </button>
                </div>

                <button
                  type="submit"
                  disabled={loading}
                  className="w-full gradient-primary text-on-primary font-bold py-4 rounded-xl hover:shadow-primary transition-all active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed font-body text-sm uppercase tracking-wide"
                >
                  {loading ? "Authenticating..." : "Sign In"}
                </button>
              </form>
            ) : (
              <form className="space-y-5">
                <div>
                  <label className="block text-sm text-on-surface-variant mb-2 font-medium">
                    6-digit verification code
                  </label>
                  <input
                    type="text"
                    value={mfaCode}
                    onChange={(e) => setMfaCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                    placeholder="000000"
                    maxLength={6}
                    className="w-full bg-surface-container-highest rounded-xl px-6 py-5 text-center text-2xl font-technical text-on-surface tracking-[0.5em] border-b-2 border-transparent focus:border-primary outline-none transition-colors"
                  />
                </div>
                <button
                  type="submit"
                  className="w-full gradient-primary text-on-primary font-bold py-4 rounded-xl hover:shadow-primary transition-all active:scale-95 text-sm uppercase tracking-wide"
                >
                  Verify
                </button>
                <button
                  type="button"
                  onClick={() => setStep("credentials")}
                  className="w-full text-on-surface-variant text-sm hover:text-on-surface transition-colors"
                >
                  Back to login
                </button>
              </form>
            )}

            <div className="mt-6 pt-6 border-t border-outline-variant/10 text-center">
              <p className="text-xs text-on-surface-variant/60">
                This portal is for authorized personnel only.
                All access is logged and audited.
              </p>
            </div>
          </div>

          {/* Dev quick login hint */}
          {process.env.NODE_ENV === "development" && (
            <div className="mt-4 p-4 rounded-xl bg-primary/5 border border-primary/10">
              <p className="text-xs text-on-surface-variant font-technical">
                DEV — admin@alpha.com / DevPass123!
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
