import { TopicInput } from "@/components/TopicInput";

export default function Page() {
  return (
    <section className="space-y-8">
      <header className="space-y-3">
        <h1 className="text-4xl tracking-tight">solo-adeventure</h1>
        <p className="text-stone-700 leading-7">
          A choose-your-own-adventure, generated page by page. Pick a topic and see where the tale leads.
        </p>
      </header>
      <TopicInput />
    </section>
  );
}
