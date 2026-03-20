"use client";

export default function SettingsPage() {
  return (
    <div className="p-8 max-w-3xl mx-auto">
      <h1 className="font-headline text-3xl font-bold text-on-surface mb-2">Settings</h1>
      <p className="text-on-surface-variant text-sm mb-8">Manage your account preferences and security settings.</p>

      <div className="space-y-4">
        {[
          { icon: "person", title: "Profile", desc: "Display name, avatar, contact info" },
          { icon: "lock", title: "Security", desc: "Password, MFA, active sessions" },
          { icon: "notifications", title: "Notifications", desc: "Alert thresholds, email digests" },
          { icon: "api", title: "API Tokens", desc: "Personal access tokens for CLI and integrations" },
        ].map((item) => (
          <div key={item.title} className="flex items-center gap-5 p-5 rounded-xl bg-surface-container-high hover:bg-surface-bright transition-colors cursor-pointer">
            <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
              <span className="material-symbols-outlined text-primary">{item.icon}</span>
            </div>
            <div>
              <p className="font-medium text-on-surface text-sm">{item.title}</p>
              <p className="text-xs text-on-surface-variant mt-0.5">{item.desc}</p>
            </div>
            <span className="material-symbols-outlined text-outline ml-auto">chevron_right</span>
          </div>
        ))}
      </div>
    </div>
  );
}
