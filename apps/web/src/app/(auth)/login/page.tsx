"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { login } from "@/lib/api";
import { NetworkSphere } from "@/components/login/NetworkSphere";

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
      {/* Left panel — 3D branding */}
      <div className="hidden lg:flex lg:w-[520px] flex-col relative overflow-hidden bg-surface-container-low border-r border-outline-variant/10">
        {/* Logo */}
        <div className="flex items-center gap-3 p-10 z-10 relative">
          <div className="w-10 h-10 rounded-xl gradient-primary flex items-center justify-center">
            <span className="material-symbols-outlined text-on-primary text-2xl">hub</span>
          </div>
          <span className="font-headline font-bold text-xl text-on-surface">Nucleus</span>
        </div>

        {/* Headline */}
        <div className="px-10 z-10 relative">
          <h1 className="font-headline text-4xl font-extrabold text-on-surface leading-tight mb-3">
            Nucleus Remote<br />
            <span className="text-primary">Access Portal</span>
          </h1>
          <p className="text-on-surface-variant text-sm leading-relaxed max-w-xs">
            Secure, curated gateway for enterprise industrial operations. Centralized telemetry and remote command center.
          </p>
          {/* Security badges */}
          <div className="flex gap-3 mt-5">
            <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full border border-tertiary/30 bg-tertiary/5">
              <span className="material-symbols-outlined text-tertiary text-sm">verified_user</span>
              <span className="text-xs text-tertiary font-medium">ISO 27001</span>
            </div>
            <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full border border-primary/30 bg-primary/5">
              <span className="material-symbols-outlined text-primary text-sm">lock</span>
              <span className="text-xs text-primary font-medium">AES-256</span>
            </div>
          </div>
        </div>

        {/* 3D Sphere — fills remaining space */}
        <div className="flex-1 relative min-h-0 mt-4">
          <NetworkSphere />
        </div>

        {/* Status bar */}
        <div className="px-10 pb-8 z-10 relative flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-tertiary animate-pulse" />
          <span className="text-xs font-technical text-tertiary tracking-widest uppercase">System Status: Nominal</span>
        </div>
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
              {step === "credentials" ? "Operator Authentication" : "Verify identity"}
            </h2>
            <p className="text-sm text-on-surface-variant mb-8">
              {step === "credentials"
                ? "Enter your credentials to access the portal."
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
                  <label className="block text-xs text-on-surface-variant mb-2 font-medium uppercase tracking-wider">
                    Work Email
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
                      placeholder="operator@nexus-corp.com"
                      className="w-full bg-surface-container-highest rounded-xl pl-11 pr-4 py-4 text-on-surface placeholder:text-outline border-b-2 border-transparent focus:border-primary outline-none transition-colors text-sm"
                    />
                  </div>
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <label className="block text-xs text-on-surface-variant font-medium uppercase tracking-wider">
                      Password
                    </label>
                    <button type="button" className="text-xs text-primary hover:text-primary-fixed-dim transition-colors">
                      Forgot Password?
                    </button>
                  </div>
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

                <label className="flex items-center gap-2 text-sm text-on-surface-variant cursor-pointer">
                  <input type="checkbox" className="rounded" />
                  Remember device for 30 days
                </label>

                <button
                  type="submit"
                  disabled={loading}
                  className="w-full gradient-primary text-on-primary font-bold py-4 rounded-xl hover:shadow-primary transition-all active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed font-body text-sm uppercase tracking-wide"
                >
                  {loading ? "Authenticating..." : "Sign In"}
                </button>

                <div className="relative flex items-center gap-3">
                  <div className="flex-1 h-px bg-outline-variant/20" />
                  <span className="text-xs text-on-surface-variant/40 uppercase tracking-wider">or identity provider</span>
                  <div className="flex-1 h-px bg-outline-variant/20" />
                </div>

                <button
                  type="button"
                  className="w-full flex items-center justify-center gap-2 py-3.5 rounded-xl border border-outline-variant/20 bg-surface-container text-on-surface-variant text-sm hover:bg-surface-container-high transition-colors"
                >
                  <span className="material-symbols-outlined text-base">shield</span>
                  Sign in with SAML
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
                Unauthorized access is strictly prohibited and monitored.<br />
                All sessions are logged for audit compliance.
              </p>
            </div>
          </div>

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
