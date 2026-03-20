"use client";

export default function SupportPage() {
  return (
    <div className="p-8 max-w-3xl mx-auto">
      <h1 className="font-headline text-3xl font-bold text-on-surface mb-2">Support</h1>
      <p className="text-on-surface-variant text-sm mb-8">Get help, report issues, or contact the engineering team.</p>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
        {[
          { icon: "description", title: "Documentation", desc: "Guides, API reference, and runbooks", href: "#" },
          { icon: "bug_report", title: "Report a Bug", desc: "Submit an issue to the engineering team", href: "#" },
          { icon: "chat", title: "Live Chat", desc: "Available weekdays 9am – 6pm CT", href: "#" },
          { icon: "email", title: "Email Support", desc: "support@nucleus.systems", href: "#" },
        ].map((item) => (
          <a key={item.title} href={item.href} className="flex items-start gap-4 p-5 rounded-xl bg-surface-container-high hover:bg-surface-bright transition-colors">
            <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0 mt-0.5">
              <span className="material-symbols-outlined text-primary">{item.icon}</span>
            </div>
            <div>
              <p className="font-medium text-on-surface text-sm">{item.title}</p>
              <p className="text-xs text-on-surface-variant mt-0.5">{item.desc}</p>
            </div>
          </a>
        ))}
      </div>

      <div className="p-5 rounded-xl bg-surface-container-low border border-outline-variant/10">
        <p className="text-xs text-on-surface-variant/60 font-technical">
          Nucleus Remote Access Portal · v1.0.0 · SLA: 99.9%
        </p>
      </div>
    </div>
  );
}
