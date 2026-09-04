"use client";

import { useEffect, useState } from "react";
import Image from "next/image";

const SLIDES = [
  {
    src: "/images/home-hero-issues.png",
    alt: "Traceway dashboard: exception groups with trends and event counts",
  },
  {
    src: "/images/home-hero-overview.png",
    alt: "Traceway dashboard: endpoints overview with impact scoring",
  },
  {
    src: "/images/metrics-application-dashboard.png",
    alt: "Traceway dashboard: custom metric widgets with charts and aggregations",
  },
];

const INTERVAL_MS = 27000;

const POSITIONS = [
  {
    transform: "translateX(-50%) scale(1)",
    zIndex: 30,
    boxShadow: "0 24px 64px rgba(0, 0, 0, 0.55)",
  },
  {
    transform: "translateX(-39%) scale(0.9)",
    zIndex: 10,
    boxShadow: "none",
  },
  {
    transform: "translateX(-61%) scale(0.9)",
    zIndex: 10,
    boxShadow: "none",
  },
];

export function HeroCarousel() {
  const [front, setFront] = useState(0);
  const [paused, setPaused] = useState(false);

  useEffect(() => {
    if (paused) return;
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    const timer = setInterval(() => {
      setFront((i) => (i + 1) % SLIDES.length);
    }, INTERVAL_MS);
    return () => clearInterval(timer);
  }, [paused, front]);

  return (
    <div
      className="relative max-w-[1180px] mx-auto"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      <div className="w-[78%] mx-auto aspect-[2336/1532]" />
      {SLIDES.map((slide, i) => {
        const position = POSITIONS[(i - front + SLIDES.length) % SLIDES.length];
        const isFront = front === i;
        return (
          <button
            key={slide.src}
            onClick={() => setFront(i)}
            aria-label={isFront ? slide.alt : `Show ${slide.alt}`}
            tabIndex={isFront ? -1 : 0}
            className={`absolute left-1/2 top-0 w-[78%] aspect-[2336/1532] overflow-hidden rounded-[12px] border border-hair-2 bg-ink-1 ${
              isFront ? "" : "cursor-pointer"
            }`}
            style={{
              ...position,
              transition:
                "transform 300ms cubic-bezier(0.4, 0, 0.2, 1), box-shadow 300ms ease",
            }}
          >
            <Image
              src={slide.src}
              alt={slide.alt}
              fill
              priority={i === 0}
              sizes="(min-width: 1240px) 920px, 78vw"
              className="object-cover object-top"
            />
          </button>
        );
      })}
    </div>
  );
}
