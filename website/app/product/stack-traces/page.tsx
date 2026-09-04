import Link from "next/link";
import { ArrowRight, Bug } from "lucide-react";

import { Chip } from "@/components/chip";
import { SectionHead } from "@/components/section-head";
import { FeatureRow } from "@/components/feature-row";
import { FaqList } from "@/components/faq-list";
import { FinalCTA } from "@/components/final-cta";
import { AuroraBackground } from "@/components/aurora-background";

export default function StackTracesPage() {
  return (
    <main className="relative">
      {/* Hero, left aligned */}
      <section className="hero hero-product relative">
        <AuroraBackground variant="hero" />
        <div className="wrap relative z-10">
          <Chip variant="crit">
            <Bug className="h-3 w-3 inline mr-1" />
            Exceptions / Stack Traces
          </Chip>
          <h1 className="mt-6">
            Find and fix issues <em>before your users notice.</em>
          </h1>
          <p className="hero-sub">
            Every exception, normalized and hashed so thousands of duplicates
            collapse into one issue, paired with the session replay or screen
            recording that caused it.
          </p>
          <div className="hero-cta-row">
            <Link href="https://docs.tracewayapp.com" className="btn btn-accent">
              Get Started <ArrowRight className="h-4 w-4" />
            </Link>
            <Link href="https://cloud.tracewayapp.com/register" className="btn btn-ghost">
              Try Traceway Cloud
            </Link>
          </div>
        </div>
      </section>

      {/* Every exception grouped and ranked, absorbed from home */}
      {/* WHITE BAND: feature sections render on white */}
      <div className="band-light">
        <section className="wrap">
          <FeatureRow
            eyebrow="Grouping"
            title={
              <>
                Every exception, <em>grouped and ranked</em>
              </>
            }
            description="Full stack traces, stable grouping. Thousands of duplicates collapse into one ranked issue so you fix what matters first."
            bullets={[
              "Full stack trace capture with file:line",
              "Intelligent error grouping via a normalized SHA-256 hash",
              "User impact analysis across sessions",
              "Source map resolution for minified JS",
            ]}
            image={{ src: "/images/exceptions-grouped-ranked.png", alt: "Exception tracking interface" }}
          />
        </section>

        {/* Intelligent grouping */}
        <section className="wrap">
          <FeatureRow
            reverse
            eyebrow="Fingerprinting"
            title={
              <>
                Same bug, <em>same group</em>, every time
              </>
            }
            description="Traceway computes its own fingerprint for every error: the stack trace is normalized, then hashed. Memory addresses, UUIDs, IPs, numeric IDs, and dependency versions are stripped first, so runtime noise never splits a group. One root cause means one issue instead of thousands, and alerts key on the same hash, so a new-issue alert fires once per real problem and a regression alert means the same failure is genuinely back."
            bullets={[
              "Error type kept, message text and the user data inside it dropped",
              "Resolved function names excluded, so source maps never reshuffle groups",
              "New-issue and regression alerts fire per group, not per occurrence",
            ]}
            image={{ src: "/images/stack-trace.png", alt: "Error grouping interface" }}
          />
        </section>

        {/* Visual context: pair stack traces with session replay */}
        <section className="wrap">
          <FeatureRow
            eyebrow="Visual context"
            title={
              <>
                Pair every stack trace with the <em>replay that caused it</em>
              </>
            }
            description="When a backend exception fires, Traceway attaches the session replay or mobile recording the user was generating at that moment. Open the stack trace and the replay is right there. See what the user did, what the UI looked like, and where the code blew up, in one pane."
            bullets={[
              "Web DOM replay linked by trace ID",
              "Flutter and React Native screen recording",
              "Jump from stack frame → exact frame of the replay",
              "Frontend + backend context in one pane",
            ]}
            image={{ src: "/images/session-replay-viewer.png", alt: "Stack trace paired with session replay" }}
          />
        </section>
      </div>

      <FinalCTA
        title={
          <>
            Triage <em>faster</em>. Ship <em>safer</em>.
          </>
        }
        description="Connect an SDK, ship an error, see it in Traceway. 5-minute setup."
        primary={{ label: "Get Started", href: "https://docs.tracewayapp.com" }}
      />

      <section className="wrap pt-10 pb-24">
        <div className="max-w-3xl mx-auto">
          <SectionHead align="center" eyebrow="FAQ" title="Questions about stack traces" />
          <div className="mt-4">
            <FaqList
              items={[
                {
                  q: "How does error grouping work?",
                  a: "Every stack trace runs through a normalization pipeline before hashing: the error type is kept while the message text is dropped, absolute paths collapse to file and line, and memory addresses, UUIDs, IPs, emails, numeric IDs, and dependency versions are replaced with placeholders. The result is hashed with SHA-256 into a 16-character fingerprint, so identical logical errors always land in the same issue even when runtime values differ.",
                },
                {
                  q: "Why does grouping matter for alerts?",
                  a: "Alert rules key on the group. A new-issue rule fires the first time a group is seen, not on every occurrence, and a regression rule fires only when an error you already resolved comes back. Stable grouping is what keeps alerts quiet during a flood of duplicates and loud the moment something genuinely new breaks.",
                },
                {
                  q: "How does automatic issue ranking work?",
                  a: "Traceway scores each issue based on how often it occurs, how recently it appeared, and how many users are affected. Issues are continuously re-ranked as new data comes in, so regressions and trending problems surface immediately, with no manual triage required.",
                },
                {
                  q: "How does grouping handle different environments?",
                  a: "The normalization strips everything environment-specific: absolute file paths, server addresses, hostnames in URLs, and dependency version suffixes. The same bug produces the same group whether it fired on one server or fifty, in staging or in production.",
                },
                {
                  q: "Can I track frontend and mobile errors alongside the backend?",
                  a: "Yes. Web (Next.js, Svelte, Remix) and mobile (Flutter, React Native) exceptions land in the same dashboard as your backend ones, and each one carries the session replay or screen recording the user was generating when it fired. Open the stack trace and the exact frame of the replay is one click away. Source maps resolve minified web traces back to original source.",
                },
              ]}
            />
          </div>
        </div>
      </section>
    </main>
  );
}
