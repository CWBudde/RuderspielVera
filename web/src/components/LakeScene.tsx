import type { ReactNode } from "react";
import { artUrl } from "@/components/Art";
import { cn } from "@/lib/utils";

/**
 * Maschsee mood illustration: sky, Neues Rathaus silhouette, tree line, water and a gliding
 * four. Replaced by `assets/art/title.jpg` when present.
 */
export function LakeScene({ className, children }: { className?: string; children?: ReactNode }) {
  const splash = artUrl("title");
  return (
    <div className={cn("relative overflow-hidden", className)}>
      {splash ? (
        <img src={splash} alt="" className="absolute inset-0 size-full object-cover" />
      ) : (
        <svg
          viewBox="0 0 400 260"
          preserveAspectRatio="xMidYMax slice"
          className="absolute inset-0 size-full"
          aria-hidden
        >
          <defs>
            <linearGradient id="ls-sky" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="#8cc6ec" />
              <stop offset="1" stopColor="#e4f2fa" />
            </linearGradient>
            <linearGradient id="ls-water" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" stopColor="#3f8fcf" />
              <stop offset="1" stopColor="#1f5f99" />
            </linearGradient>
          </defs>
          <rect width="400" height="260" fill="url(#ls-sky)" />
          <g fill="#fff" opacity="0.85">
            <ellipse cx="70" cy="40" rx="34" ry="9" />
            <ellipse cx="92" cy="34" rx="20" ry="9" />
            <ellipse cx="310" cy="56" rx="40" ry="8" />
            <ellipse cx="330" cy="49" rx="20" ry="8" />
          </g>
          {/* Neues Rathaus */}
          <g fill="#d9c7a6" stroke="#1b2a41" strokeWidth="1.2" strokeLinejoin="round">
            <rect x="160" y="120" width="80" height="34" />
            <rect x="188" y="104" width="24" height="18" />
            <path d="M186 104c0-16 28-16 28 0z" fill="#7fb59b" />
            <path d="M200 82v6" />
            <path d="M163 120l7-12 7 12zM223 120l7-12 7 12z" fill="#b5523b" />
          </g>
          <path d="M276 150V86l3-10 3 10v64z" fill="#e8eef2" stroke="#1b2a41" strokeWidth="1.2" />
          {/* trees */}
          <path
            d="M0 150c8-14 20-16 28-8 6-12 22-14 30-2 8-10 24-8 30 4 10-8 22-6 28 6h0c10-12 26-10 32 2 10-8 24-6 30 6 8-10 22-10 30 0 8-12 24-12 32-2 10-10 24-8 30 4 8-8 22-8 28 2 10-10 24-10 32 0 6-6 18-6 24 0 8-10 20-10 26-2V170H0z"
            fill="#5f9a41"
            stroke="#1b2a41"
            strokeWidth="1.2"
          />
          <rect y="160" width="400" height="100" fill="url(#ls-water)" />
          <g stroke="#fff" strokeOpacity="0.35" strokeWidth="2" strokeLinecap="round" strokeDasharray="18 14">
            <path d="M10 182h380" />
            <path d="M-20 204h420" strokeDashoffset="9" />
            <path d="M0 230h400" strokeDashoffset="20" />
          </g>
          {/* sail boats */}
          <g stroke="#1b2a41" strokeWidth="1.2">
            <path d="M60 172V146l14 24z" fill="#fff" />
            <path d="M52 172h26l-3 4H55z" fill="#c2412d" />
            <path d="M340 168v-22l12 20z" fill="#fff" />
            <path d="M333 168h22l-3 3h-16z" fill="#2b78bb" />
          </g>
          {/* the four */}
          <g className="motion-safe:animate-[glide_14s_linear_infinite]">
            <path
              d="M120 214c30 6 110 6 140 0l-4 4c-30 5-102 5-132 0z"
              fill="#f3c431"
              stroke="#1b2a41"
              strokeWidth="1.4"
            />
            {[0, 1, 2, 3].map((i) => (
              <g key={i} stroke="#1b2a41" strokeWidth="1.3">
                <path d={`M${150 + i * 26} 214l-16 14M${150 + i * 26} 214l14 -2`} />
                <circle cx={150 + i * 26} cy={207} r={4} fill="#2a4f8f" />
              </g>
            ))}
          </g>
        </svg>
      )}
      {children}
    </div>
  );
}
