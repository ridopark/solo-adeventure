"use client";

import { useEffect, useState } from "react";

// Known user-agent fragments for in-app browsers that Google's OAuth policy
// (and many other flows) explicitly rejects. Full-page WebViews in these apps
// won't sign in successfully.
const IN_APP_PATTERNS = [
  "FBAN", "FBAV", "FB_IAB", "FBIOS",
  "Instagram",
  "Barcelona",
  "Messenger",
  "LinkedInApp", "LinkedIn",
  "Twitter", "TwitterAndroid",
  "musical_ly", "BytedanceWebview",
  "KAKAOTALK",
  "Line/",
  "MicroMessenger",
  "Snapchat",
  "Pinterest",
  "Discord",
];

function detectInApp(ua: string): boolean {
  return IN_APP_PATTERNS.some((p) => ua.includes(p));
}

export function InAppBrowserBanner() {
  const [inApp, setInApp] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    const ua = typeof navigator !== "undefined" ? navigator.userAgent : "";
    setInApp(detectInApp(ua));
  }, []);

  if (!inApp || dismissed) return null;

  return (
    <div
      role="alert"
      className="flex items-start justify-between gap-3 border-b px-4 py-3 text-sm"
      style={{
        background: "#fff3cd",
        borderColor: "#e8d992",
        color: "#6b4c00",
      }}
    >
      <div>
        <strong>This browser can&apos;t sign in with Google.</strong>{" "}
        Tap the <span aria-label="menu">&#8230;</span> (or share) menu above and choose
        <em> Open in Browser</em> / <em>Open externally</em> / <em>Open in Safari</em> to continue.
      </div>
      <button
        type="button"
        onClick={() => setDismissed(true)}
        aria-label="Dismiss"
        className="shrink-0 text-lg leading-none opacity-60 hover:opacity-100"
      >
        &times;
      </button>
    </div>
  );
}
