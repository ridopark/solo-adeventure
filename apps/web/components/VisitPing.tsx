"use client";

import { useEffect } from "react";
import { api } from "@/lib/api";

const KEY = "solo-adv:visited";

export function VisitPing() {
  useEffect(() => {
    try {
      if (sessionStorage.getItem(KEY)) return;
      sessionStorage.setItem(KEY, "1");
    } catch {
      /* private mode -- still ping once per load */
    }
    void api.visit(window.location.pathname);
  }, []);
  return null;
}
