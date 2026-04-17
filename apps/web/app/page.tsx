import Link from "next/link";
import { TopicInput } from "@/components/TopicInput";
import { AuthNav } from "@/components/AuthNav";

export default function LandingPage() {
  return (
    <div>
      <nav className="border-b border-[var(--border-cream)]">
        <div className="mx-auto max-w-5xl px-6 py-4 flex items-center justify-between">
          <Link href="/" className="font-serif text-xl tracking-tight">
            solo-adeventure
          </Link>
          <AuthNav />
        </div>
      </nav>

      <section className="mx-auto max-w-3xl px-6 pt-24 pb-16 text-center">
        <p className="uppercase tracking-[0.15em] text-xs text-[var(--stone-gray)] mb-6">
          A choose-your-own-adventure gamebook
        </p>
        <h1 className="font-serif text-5xl md:text-6xl leading-[1.10] tracking-tight mb-8 text-[var(--near-black)]">
          Your story, <br className="hidden md:block" />
          paged by Claude.
        </h1>
        <p className="text-lg md:text-xl leading-[1.60] text-[var(--olive-gray)] max-w-2xl mx-auto mb-12">
          Hand it a topic. Each page arrives written and illustrated as you turn it --
          two or three paths always waiting at the bottom.
        </p>

        <div
          className="mx-auto max-w-xl rounded-2xl p-8"
          style={{ background: "var(--ivory)", border: "1px solid var(--border-cream)", boxShadow: "rgba(0,0,0,0.05) 0px 4px 24px" }}
        >
          <TopicInput />
        </div>
      </section>

      <section className="mx-auto max-w-5xl px-6 py-20">
        <div className="text-center mb-14">
          <p className="uppercase tracking-[0.15em] text-xs text-[var(--stone-gray)] mb-4">How it works</p>
          <h2 className="font-serif text-3xl md:text-4xl leading-[1.20] tracking-tight text-[var(--near-black)]">
            Three beats, a finished tale.
          </h2>
        </div>
        <div className="grid md:grid-cols-3 gap-6">
          {[
            {
              step: "01",
              title: "Pick a topic",
              body: "A lighthouse keeper in 1912. A cartographer who finds a map they drew in a dream. Anything you can phrase in a sentence.",
            },
            {
              step: "02",
              title: "Read the page",
              body: "Claude writes 150-250 words of second-person present tense. FLUX paints the scene in the same style across the whole book.",
            },
            {
              step: "03",
              title: "Choose your path",
              body: "Two or three distinct choices at the foot of the page. Click one; the next page is written around your decision.",
            },
          ].map((c) => (
            <article
              key={c.step}
              className="rounded-2xl p-7"
              style={{ background: "var(--ivory)", border: "1px solid var(--border-cream)" }}
            >
              <p className="text-xs tracking-[0.15em] uppercase text-[var(--stone-gray)] mb-4">{c.step}</p>
              <h3 className="font-serif text-2xl leading-[1.20] mb-3 text-[var(--near-black)]">{c.title}</h3>
              <p className="text-base leading-[1.60] text-[var(--olive-gray)]">{c.body}</p>
            </article>
          ))}
        </div>
      </section>

      <section style={{ background: "var(--near-black)" }}>
        <div className="mx-auto max-w-4xl px-6 py-24">
          <p className="uppercase tracking-[0.15em] text-xs text-[var(--stone-gray)] mb-6">Under the hood</p>
          <h2 className="font-serif text-3xl md:text-5xl leading-[1.10] tracking-tight mb-10" style={{ color: "var(--ivory)" }}>
            Written by Claude. <br />
            Painted by FLUX.
          </h2>
          <div className="grid md:grid-cols-2 gap-10 text-base leading-[1.60]" style={{ color: "var(--warm-silver)" }}>
            <div>
              <p className="font-serif text-xl mb-3" style={{ color: "var(--ivory)" }}>Narrative</p>
              <p>
                Claude Haiku 4.5 generates every page as a structured tool call --
                a narrative, an image prompt, 2-3 divergent choices, and a short running
                summary that becomes the next page's memory.
              </p>
            </div>
            <div>
              <p className="font-serif text-xl mb-3" style={{ color: "var(--ivory)" }}>Art</p>
              <p>
                FLUX.1-schnell renders the illustration. A single style descriptor,
                fixed at story start, keeps every page looking like it came from the
                same book. fal.ai fills in if Together rate-limits.
              </p>
            </div>
            <div>
              <p className="font-serif text-xl mb-3" style={{ color: "var(--ivory)" }}>Safety</p>
              <p>
                Topic blocklist before generation, an explicit PG-13 content policy in
                the system prompt, and an image-prompt validator before anything
                reaches the painters.
              </p>
            </div>
            <div>
              <p className="font-serif text-xl mb-3" style={{ color: "var(--ivory)" }}>Architecture</p>
              <p>
                Go hexagonal backend (ports + adapters) on Oracle Cloud, Next.js static
                export on Cloudflare Pages. Swapping Claude for another provider, or
                FLUX for another painter, is one file.
              </p>
            </div>
          </div>
        </div>
      </section>

      <footer className="border-t border-[var(--border-cream)]">
        <div className="mx-auto max-w-5xl px-6 py-10 text-sm text-[var(--stone-gray)] flex flex-wrap items-center justify-between gap-4">
          <p>solo-adeventure. Built on Claude and FLUX.</p>
          <div className="flex gap-6">
            <a
              href="https://github.com/ridopark/solo-adeventure"
              target="_blank"
              rel="noreferrer"
              className="hover:text-[var(--near-black)] transition"
            >
              GitHub
            </a>
            <a
              href="https://claude.com"
              target="_blank"
              rel="noreferrer"
              className="hover:text-[var(--near-black)] transition"
            >
              Claude
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}
