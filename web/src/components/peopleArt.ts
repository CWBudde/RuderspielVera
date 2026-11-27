/**
 * Named portraits: `src/assets/art/people/<card id>.png`, likenesses of real club members
 * (git-ignored, local only). Dev server only: `vite build` aliases this module to
 * peopleArt.stub.ts (vite.config.ts), so the photos never end up in a build or on GitHub Pages.
 */
const peopleFiles = import.meta.glob<string>("../assets/art/people/*.{webp,png,jpg}", {
  eager: true,
  query: "?url",
  import: "default",
});

export const PEOPLE_MANIFEST: Record<string, string> = Object.fromEntries(
  Object.entries(peopleFiles).map(([path, url]) => [path.replace(/^.*\/(.+)\.\w+$/, "$1"), url]),
);
